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

Create role:

    vault write auth/tarmak/role/ami-1b6c3b62 auth_type=ec2 bound_ami_id=ami-1b6c3b62 policies=prod,dev

Attempt to get a token:

    vault write auth/tarmak/login pkcs7="$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/pkcs7)"
