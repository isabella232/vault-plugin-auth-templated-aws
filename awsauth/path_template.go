package awsauth

import (
	"context"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathTemplate(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "template/" + framework.GenericNameRegex("role") + "/" + framework.GenericNameRegex("templateName"),
		Fields: map[string]*framework.FieldSchema{
			"template": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "",
			},
			"templateName": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "",
			},
		},

		ExistenceCheck: b.pathTemplateExistenceCheck,

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.CreateOperation: b.pathTemplateCreateUpdate,
			logical.UpdateOperation: b.pathTemplateCreateUpdate,
			logical.DeleteOperation: b.pathTemplateDelete,
			logical.ReadOperation:   b.pathTemplateRead,
		},

		HelpSynopsis:    pathTemplateHelpSyn,
		HelpDescription: pathTemplateHelpDesc,
	}
}

// Establishes dichotomy of request operation between CreateOperation and UpdateOperation.
// Returning 'true' forces an UpdateOperation, CreateOperation otherwise.
func (b *backend) pathTemplateExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	entry, err := b.lockedTemplateEntry(ctx, req.Storage)
	if err != nil {
		return false, err
	}
	return entry != nil, nil
}

func (b *backend) lockedTemplateEntry(ctx context.Context, s logical.Storage) (*vaultConfig, error) {
	b.configMutex.RLock()
	defer b.configMutex.RUnlock()

	return b.nonLockedTemplateEntry(ctx, s)
}

func (b *backend) nonLockedTemplateEntry(ctx context.Context, s logical.Storage, roleName string, templateName string) (*vaultConfig, error) {
	entry, err := s.Get(ctx, "template")
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

func (b *backend) pathTemplateRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	vaultConfig, err := b.lockedTemplateEntry(ctx, req.Storage)
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

func (b *backend) pathTemplateDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	if err := req.Storage.Delete(ctx, "template"); err != nil {
		return nil, err
	}

	b.vaultClient.SetToken("")

	return nil, nil
}

func (b *backend) pathTemplateCreateUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	configEntry, err := b.nonLockedTemplateEntry(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if configEntry == nil {
		configEntry = &vaultConfig{}
	}

	token, ok := data.GetOk("token")
	if ok {
		b.vaultClient.SetToken(token.(string))
	}

	entry, err := logical.StorageEntryJSON("config/vault", configEntry)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

type template struct {
	Token string `json:"token"`
}

const pathTemplateHelpSyn = `
`

const pathTemplateHelpDesc = `
`
