vault-plugin-auth-templated-aws
===============================

vault-plugin-auth-templated-aws is a vault plugin to dynamically create roles and policies based on the identity of the EC2 instance requesting a vault token.
It is implemented as a fork of vault's awsauth backend, with a templating system added.

To build:

    $ dep ensure
    $ go build

To run:

Add `plugin_directory = "/etc/vault/plugins"` to vault config.

Copy binary into plugin directory:

    $ mkdir -p /etc/vault/plugins/
    $ cp vault-plugin-auth-templated-aws /etc/vault/plugins/vault-plugin-auth-templated-aws

Calculate hash of plugin:

    $ export SHA256=$(shasum -a 256 "/etc/vault/plugins/vault-plugin-auth-templated-aws" | cut -d' ' -f1)

Register it with vault:

    $ vault write sys/plugins/catalog/vault-plugin-auth-templated-aws sha_256="${SHA256}" command="vault-plugin-auth-templated-aws"

Enable it as an auth method:

    $ vault auth enable -path="tarmak" -plugin-name="vault-plugin-auth-templated-aws" plugin

Check it appears in auth list:

    $ vault auth list

Disable it with:

    $ vault auth disable tarmak


Configuring
-----------

Set the vault token and address to use for writing new policies:

    vault write auth/tarmak/config/vault token=7459a4df-1e18-6b08-5c0a-f0106badc284 address=http://127.0.0.1:8200

Optionally set the aws credentials for talking to the ec2 api:

    vault write auth/tarmak/config/client secret_key=something access_key=something_else

Create role:

    vault write auth/tarmak/role/vault-test bound_iam_role_arn=arn:aws:iam::228615251467:role/tarmak-vault base_path="/"

Create some templates (see the section below for more information):

    vault write auth/tarmak/template/vault-test/test-policy template='path "secret/*" { capabilities = ["create"] } path "secret/foo" { capabilities = ["read"] }' type=policy path="sys/policy"
    vault write auth/tarmak/template/vault-test/test-pki template='{"allowed_domains": ["{{ .FQDN }}"], "allow_subdomains": true}' type=generic path="pki/roles"

Get a token:

    vault write auth/tarmak/login pkcs7="$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7)" role=vault-test

Templates
---------

Templates are processed using go's templating langauge, with the following variables supported:

- `{{ .InstanceHash }}`: the ID of the requesting instance (e.g `i-0f7ebb331c89ed78c`)
- `{{ .FQDN }}`: the private DNS name of the requesting instance (e.g. `ip-172-31-19-213.eu-west-1.compute.internal`)
- `{{ .InternalIPv4 }}`: the private IP address of the requesting instance
- `{{ .BasePath }}`: the `base_path` set on the role used
- `{{ .OutputPath }}`: the `path` set on the template
- `{{ .TemplateName }}`: the name of the template

These templates will be rendered to `{{.BasePath}}/{{.OutputPath}}/{{.TemplateName}}-{{.InstanceHash}}` in vault.

### policy

Templates with `type=policy` are parsed and processed in HCL. See [this page](https://www.vaultproject.io/docs/concepts/policies.html#policy-syntax) for details.

### generic

Templates with `type=generic` are specified in JSON format, and are processed as generic vault secrets.
Although being intended to configure PKI roles, they could be used for other purposes.
