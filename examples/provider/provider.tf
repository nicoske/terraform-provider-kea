# Configure via HTTP endpoint with basic authentication
provider "kea" {
  dhcp4 = {
    address       = "http://192.0.2.1:8000"
    http_username = "admin"
    http_password = "secret"
  }
}

# Configure via HTTPS endpoint with mutual TLS
provider "kea" {
  dhcp4 = {
    address              = "https://kea.example.com:8000"
    tls_ca_file          = "/etc/terraform/kea/ca.pem"
    tls_client_cert_file = "/etc/terraform/kea/client.pem"
    tls_client_key_file  = "/etc/terraform/kea/client-key.pem"
  }
}

# Configure via UNIX socket
provider "kea" {
  dhcp4 = {
    address = "unix:///run/kea/kea-dhcp4.sock"
  }
}

# Configure via environment variables
# export KEA_DHCP4_ADDRESS="http://192.0.2.1:8000"
# export KEA_DHCP4_HTTP_USERNAME="admin"
# export KEA_DHCP4_HTTP_PASSWORD="secret"
# export KEA_DHCP4_TLS_CA_FILE="/etc/terraform/kea/ca.pem"
# export KEA_DHCP4_TLS_CLIENT_CERT_FILE="/etc/terraform/kea/client.pem"
# export KEA_DHCP4_TLS_CLIENT_KEY_FILE="/etc/terraform/kea/client-key.pem"
provider "kea" {
}
