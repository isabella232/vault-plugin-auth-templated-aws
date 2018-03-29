package awsauth

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathConfigVault(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "config/vault$",
		Fields: map[string]*framework.FieldSchema{
			"token": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "Vault token to use for creating roles and policies",
			},
		},

		ExistenceCheck: b.pathConfigClientExistenceCheck,

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.CreateOperation: b.pathVaultClientCreateUpdate,
			logical.UpdateOperation: b.pathVaultClientCreateUpdate,
			logical.DeleteOperation: b.pathVaultClientDelete,
			logical.ReadOperation:   b.pathVaultClientRead,
		},

		HelpSynopsis:    pathVaultClientHelpSyn,
		HelpDescription: pathVaultClientHelpDesc,
	}
}

// Establishes dichotomy of request operation between CreateOperation and UpdateOperation.
// Returning 'true' forces an UpdateOperation, CreateOperation otherwise.
func (b *backend) pathVaultClientExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	entry, err := b.lockedClientConfigEntry(ctx, req.Storage)
	if err != nil {
		return false, err
	}
	return entry != nil, nil
}

func (b *backend) lockedVaultConfigEntry(ctx context.Context, s logical.Storage) (*vaultConfig, error) {
	b.configMutex.RLock()
	defer b.configMutex.RUnlock()

	return b.nonLockedVaultConfigEntry(ctx, s)
}

// Fetch the client configuration required to access the vault API.
func (b *backend) nonLockedVaultConfigEntry(ctx context.Context, s logical.Storage) (*vaultConfig, error) {
	entry, err := s.Get(ctx, "config/vault")
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var result vaultConfig
	if err := entry.DecodeJSON(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *backend) pathVaultClientRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	vaultConfig, err := b.lockedVaultConfigEntry(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if vaultConfig == nil {
		return nil, nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"token": vaultConfig.Token,
		},
	}, nil
}

func (b *backend) pathVaultClientDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	if err := req.Storage.Delete(ctx, "config/client"); err != nil {
		return nil, err
	}

	// Remove all the cached EC2 client objects in the backend.
	b.flushCachedEC2Clients()

	// Remove all the cached EC2 client objects in the backend.
	b.flushCachedIAMClients()

	// unset the cached default AWS account ID
	b.defaultAWSAccountID = ""

	return nil, nil
}

// pathConfigClientCreateUpdate is used to register the 'aws_secret_key' and 'aws_access_key'
// that can be used to interact with AWS EC2 API.
func (b *backend) pathVaultClientCreateUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	configEntry, err := b.nonLockedClientConfigEntry(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if configEntry == nil {
		configEntry = &clientConfig{}
	}

	// changedCreds is whether we need to flush the cached AWS clients and store in the backend
	changedCreds := false
	// changedOtherConfig is whether other config has changed that requires storing in the backend
	// but does not require flushing the cached clients
	changedOtherConfig := false

	accessKeyStr, ok := data.GetOk("access_key")
	if ok {
		if configEntry.AccessKey != accessKeyStr.(string) {
			changedCreds = true
			configEntry.AccessKey = accessKeyStr.(string)
		}
	} else if req.Operation == logical.CreateOperation {
		// Use the default
		configEntry.AccessKey = data.Get("access_key").(string)
	}

	secretKeyStr, ok := data.GetOk("secret_key")
	if ok {
		if configEntry.SecretKey != secretKeyStr.(string) {
			changedCreds = true
			configEntry.SecretKey = secretKeyStr.(string)
		}
	} else if req.Operation == logical.CreateOperation {
		configEntry.SecretKey = data.Get("secret_key").(string)
	}

	endpointStr, ok := data.GetOk("endpoint")
	if ok {
		if configEntry.Endpoint != endpointStr.(string) {
			changedCreds = true
			configEntry.Endpoint = endpointStr.(string)
		}
	} else if req.Operation == logical.CreateOperation {
		configEntry.Endpoint = data.Get("endpoint").(string)
	}

	iamEndpointStr, ok := data.GetOk("iam_endpoint")
	if ok {
		if configEntry.IAMEndpoint != iamEndpointStr.(string) {
			changedCreds = true
			configEntry.IAMEndpoint = iamEndpointStr.(string)
		}
	} else if req.Operation == logical.CreateOperation {
		configEntry.IAMEndpoint = data.Get("iam_endpoint").(string)
	}

	stsEndpointStr, ok := data.GetOk("sts_endpoint")
	if ok {
		if configEntry.STSEndpoint != stsEndpointStr.(string) {
			// We don't directly cache STS clients as they are ever directly used.
			// However, they are potentially indirectly used as credential providers
			// for the EC2 and IAM clients, and thus we would be indirectly caching
			// them there. So, if we change the STS endpoint, we should flush those
			// cached clients.
			changedCreds = true
			configEntry.STSEndpoint = stsEndpointStr.(string)
		}
	} else if req.Operation == logical.CreateOperation {
		configEntry.STSEndpoint = data.Get("sts_endpoint").(string)
	}

	headerValStr, ok := data.GetOk("iam_server_id_header_value")
	if ok {
		if configEntry.IAMServerIdHeaderValue != headerValStr.(string) {
			// NOT setting changedCreds here, since this isn't really cached
			configEntry.IAMServerIdHeaderValue = headerValStr.(string)
			changedOtherConfig = true
		}
	} else if req.Operation == logical.CreateOperation {
		configEntry.IAMServerIdHeaderValue = data.Get("iam_server_id_header_value").(string)
	}

	maxRetriesInt, ok := data.GetOk("max_retries")
	if ok {
		configEntry.MaxRetries = maxRetriesInt.(int)
	} else if req.Operation == logical.CreateOperation {
		configEntry.MaxRetries = data.Get("max_retries").(int)
	}

	// Since this endpoint supports both create operation and update operation,
	// the error checks for access_key and secret_key not being set are not present.
	// This allows calling this endpoint multiple times to provide the values.
	// Hence, the readers of this endpoint should do the validation on
	// the validation of keys before using them.
	entry, err := logical.StorageEntryJSON("config/client", configEntry)
	if err != nil {
		return nil, err
	}

	if changedCreds || changedOtherConfig || req.Operation == logical.CreateOperation {
		if err := req.Storage.Put(ctx, entry); err != nil {
			return nil, err
		}
	}

	if changedCreds {
		b.flushCachedEC2Clients()
		b.flushCachedIAMClients()
		b.defaultAWSAccountID = ""
	}

	return nil, nil
}

type vaultConfig struct {
	Token string `json:"token"`
}

const pathVaultClientHelpSyn = `
`

const pathVaultClientHelpDesc = `
`
