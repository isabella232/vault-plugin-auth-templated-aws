package awsauth

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func pathTemplate(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "template/" + framework.GenericNameRegex("role") + "/" + framework.GenericNameRegex("templateName"),
		Fields: map[string]*framework.FieldSchema{
			"role": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "",
			},
			"templateName": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "",
			},
			"type": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "Type of this template, one of policy|generic",
			},
			"path": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "Relative path to render template to",
			},
			"template": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "The actual template data",
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

func pathListTemplates(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "template/" + framework.GenericNameRegex("role") + "/?",

		Fields: map[string]*framework.FieldSchema{
			"role": &framework.FieldSchema{
				Type:        framework.TypeString,
				Default:     "",
				Description: "",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ListOperation: b.pathTemplateList,
		},

		HelpSynopsis:    pathTemplateHelpSyn,
		HelpDescription: pathTemplateHelpDesc,
	}
}

// Establishes dichotomy of request operation between CreateOperation and UpdateOperation.
// Returning 'true' forces an UpdateOperation, CreateOperation otherwise.
func (b *backend) pathTemplateExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	roleName := data.Get("role").(string)
	if roleName == "" {
		return false, fmt.Errorf("missing role")
	}

	templateName := data.Get("templateName").(string)
	if roleName == "" {
		return false, fmt.Errorf("missing templateName")
	}

	entry, err := b.lockedTemplateEntry(ctx, req.Storage, roleName, templateName)
	if err != nil {
		return false, err
	}
	return entry != nil, nil
}

func (b *backend) lockedTemplateEntry(ctx context.Context, s logical.Storage, roleName string, templateName string) (*template, error) {
	b.templateMutex.RLock()
	defer b.templateMutex.RUnlock()

	return b.nonLockedTemplateEntry(ctx, s, roleName, templateName)
}

func templatePath(roleName, templateName string) string {
	return fmt.Sprintf("template/%s/%s", roleName, templateName)
}

func (b *backend) nonLockedTemplateEntry(ctx context.Context, s logical.Storage, roleName string, templateName string) (*template, error) {
	if roleName == "" {
		return nil, fmt.Errorf("missing role name")
	}

	if templateName == "" {
		return nil, fmt.Errorf("missing template name")
	}

	entry, err := s.Get(ctx, templatePath(roleName, templateName))
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	var result template
	if err := entry.DecodeJSON(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (b *backend) pathTemplateList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := data.Get("role").(string)
	if roleName == "" {
		return nil, fmt.Errorf("missing role")
	}

	templates, err := req.Storage.List(ctx, "template/"+roleName+"/")
	if err != nil {
		return nil, err
	}
	return logical.ListResponse(templates), nil
}

func (b *backend) pathTemplateRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := data.Get("role").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing role"), nil
	}

	templateName := data.Get("templateName").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing templateName"), nil
	}

	template, err := b.lockedTemplateEntry(ctx, req.Storage, roleName, templateName)
	if err != nil {
		return nil, err
	}

	if template == nil {
		return nil, nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"path":          template.Path,
			"template_name": template.TemplateName,
			"role":          template.Role,
			"type":          template.Type,
			"template":      template.Template,
		},
	}, nil
}

func (b *backend) pathTemplateDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := data.Get("role").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing role"), nil
	}

	templateName := data.Get("templateName").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing templateName"), nil
	}

	b.templateMutex.Lock()
	defer b.templateMutex.Unlock()

	if err := req.Storage.Delete(ctx, templatePath(roleName, templateName)); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathTemplateCreateUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.templateMutex.Lock()
	defer b.templateMutex.Unlock()

	roleName := data.Get("role").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing role"), nil
	}

	templateName := data.Get("templateName").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing templateName"), nil
	}

	t, err := b.nonLockedTemplateEntry(ctx, req.Storage, roleName, templateName)
	if err != nil {
		return nil, err
	}
	if t == nil {
		t = &template{}
	}

	t.Role = roleName
	t.TemplateName = templateName

	template := data.Get("template").(string)
	if template != "" {
		t.Template = template
	}

	templateType := data.Get("type").(string)
	if templateType != "" {
		t.Type = templateType
	}

	path := data.Get("path").(string)
	if path != "" {
		t.Path = path
	}

	entry, err := logical.StorageEntryJSON(templatePath(roleName, templateName), t)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

type template struct {
	Role         string `json:"role"`
	TemplateName string `json:"templateName"`
	Type         string `json:"type"`
	Path         string `json:"path"`
	Template     string `json:"template"`
}

const pathTemplateHelpSyn = `
`

const pathTemplateHelpDesc = `
`
