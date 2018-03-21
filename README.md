tarmak-vault-auth-plugin
========================

To build:

    $ dep ensure
    $ go build

To run:

Add `plugin_directory = "/etc/vault/plugins"` to vault config.

Copy binary into plugin directory:

    $ cp tarmak-vault-auth-plugin /etc/vault/plugins/tarmak-vault-auth-plugin

Calculate hash of plugin:

    $ export SHA256=$(shasum -a 256 "/etc/vault/plugins/tarmak-vault-auth-plugin" | cut -d' ' -f1)

Register it with vault:

    $ vault write sys/plugins/catalog/tarmak-vault-auth-plugin sha_256="${SHA256}" command="tarmak-vault-auth-plugin"

Enable it as an auth method:

    $ vault auth enable -path="tarmak" -plugin-name="tarmak-vault-auth-plugin" plugin

Check it appears in auth list:

    $ vault auth list
