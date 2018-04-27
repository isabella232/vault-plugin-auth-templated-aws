tarmak-vault-auth-plugin
========================

To build:

    $ dep ensure
    $ go build

To run:

Add `plugin_directory = "/etc/vault/plugins"` to vault config.

Copy binary into plugin directory:

    $ mkdir -p /etc/vault/plugins/
    $ cp tarmak-vault-auth-plugin /etc/vault/plugins/tarmak-vault-auth-plugin

Calculate hash of plugin:

    $ export SHA256=$(shasum -a 256 "/etc/vault/plugins/tarmak-vault-auth-plugin" | cut -d' ' -f1)

Register it with vault:

    $ vault write sys/plugins/catalog/tarmak-vault-auth-plugin sha_256="${SHA256}" command="tarmak-vault-auth-plugin"

Enable it as an auth method:

    $ vault auth enable -path="tarmak" -plugin-name="tarmak-vault-auth-plugin" plugin

Check it appears in auth list:

    $ vault auth list

Disable it with:

    $ vault auth disable tarmak


Configuring
-----------

Set the vault token to use for writing new policies:

    vault write auth/tarmak/config/vault token=7459a4df-1e18-6b08-5c0a-f0106badc284

Optionally set the aws credentials for talking to the ec2 api:

    vault write auth/tarmak/config/client secret_key=something access_key=something_else

Create role:

    vault write auth/tarmak/role/vault-test bound_iam_role_arn=arn:aws:iam::228615251467:role/tarmak-vault base_path="/"

Create some templates:

    vault write auth/tarmak/template/vault-test/test-policy template='path "secret/*" { capabilities = ["create"] } path "secret/foo" { capabilities = ["read"] }' type=policy path="sys/policy"
    vault write auth/tarmak/template/vault-test/test-pki template='{"allowed_domains": ["{{ .FQDN }}"], "allow_subdomains": true}' type=generic path="pki/roles"

These templates are processed using go's templating langauge, with the following variables supported:

- `{{ .InstanceHash }}`: the ID of the requesting instance (e.g `i-0f7ebb331c89ed78c`)
- `{{ .FQDN }}`: the private DNS name of the requesting instance (e.g. `ip-172-31-19-213.eu-west-1.compute.internal`)
- `{{ .InternalIPv4 }}`: the private IP address of the requesting instance
- `{{ .BasePath }}`: the `base_path` set on the role used
- `{{ .OutputPath }}`: the `path` set on the template

Get a token:

    vault write auth/tarmak/login pkcs7="$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7)" role=vault-test
