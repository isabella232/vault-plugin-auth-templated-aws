Vault config:

```
storage "file" {
  path = "/home/centos/vault-data"
}

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = "true"
  tls_cert_file = "/etc/vault/tls/tls.pem"
  tls_key_file = "/etc/vault/tls/tls-key.pem"
}

default_lease_ttl = "168h"
max_lease_ttl = "720h"
disable_mlock = false

cluster_name = "vault-louis"

api_addr = "http://localhost:8200"
plugin_directory = "/etc/vault/plugins"
```

```
export VAULT_ADDR='http://127.0.0.1:8200'
```

```
sudo vault server -log-level=trace -config=vault.hcl
```
