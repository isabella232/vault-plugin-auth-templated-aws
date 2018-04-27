package awsauth

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/api"
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
			"address": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "Address of vault API",
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
			"token":   vaultConfig.Token,
			"address": vaultConfig.Address,
		},
	}, nil
}

func (b *backend) pathVaultClientDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	if err := req.Storage.Delete(ctx, "config/vault"); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathVaultClientCreateUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	configEntry, err := b.nonLockedVaultConfigEntry(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if configEntry == nil {
		configEntry = &vaultConfig{}
	}

	token, ok := data.GetOk("token")
	if ok {
		configEntry.Token = token.(string)
	}

	address, ok := data.GetOk("address")
	if ok {
		configEntry.Address = address.(string)
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

func (b *backend) GetVaultClient(ctx context.Context, s logical.Storage) (*api.Client, error) {
	config := api.DefaultConfig()

	vaultClient, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	storedConf, err := b.nonLockedVaultConfigEntry(ctx, s)
	if err != nil {
		return nil, err
	}
	if storedConf.Token == "" {
		return nil, fmt.Errorf("error: missing vault token")
	}
	vaultClient.SetToken(storedConf.Token)

	if storedConf.Address == "" {
		return nil, fmt.Errorf("error: missing vault API address")
	}
	vaultClient.SetAddress(storedConf.Address)

	return vaultClient, nil
}

type vaultConfig struct {
	Token   string `json:"token"`
	Address string `json:"address"`
}

const pathVaultClientHelpSyn = `
`

const pathVaultClientHelpDesc = `
`
