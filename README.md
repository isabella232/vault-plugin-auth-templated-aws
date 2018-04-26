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

    vault write auth/tarmak/config/vault token=235ef135-0a9a-d1c4-a6bc-3e23d81ec63e

Optionally set the aws credentials for talking to the ec2 api:

    vault write auth/tarmak/config/client secret_key=something access_key=something_else

Create role:

    vault write auth/tarmak/role/vault-test auth_type=ec2 bound_iam_role_arn=arn:aws:iam::228615251467:role/tarmak-vault base_path=test-cluster

Create a template:

    vault write auth/tarmak/template/vault-test/example template='path "secret/*" { capabilities = ["create"] } path "secret/foo" { capabilities = ["read"] }' type=policy base_path=testing-plugin

Attempt to get a token:

    vault write auth/tarmak/login pkcs7="$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7)" role=vault-test
