ui           = false
cluster_name = "integin-pilot-openbao"
api_addr     = "http://127.0.0.1:18200"

storage "file" {
  path = "/openbao/file"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = true
}

default_lease_ttl = "1h"
max_lease_ttl     = "4h"

audit "file" "integin-pilot-audit" {
  description = "Pilot rehearsal audit trail"
  options {
    file_path = "/openbao/logs/audit.log"
  }
}