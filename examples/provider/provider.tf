# Configure via HTTP endpoint with basic authentication
provider "kea" {
  dhcp4 = {
    address       = "http://192.0.2.1:8000"
    http_username = "admin"
    http_password = "secret"
  }
}

# Configure via UNIX socket
provider "kea" {
  dhcp4 = {
    address = "unix:///run/kea/kea-dhcp4.sock"
  }
}

# Configure via HTTPS with a private CA and mutual TLS
provider "kea" {
  dhcp4 = {
    address     = "https://192.0.2.1:8004"
    ca_cert     = file("${path.module}/certs/ca.pem")
    client_cert = file("${path.module}/certs/client.pem")
    client_key  = file("${path.module}/certs/client-key.pem")
  }
}

# Configure via environment variables
# export KEA_DHCP4_ADDRESS="http://192.0.2.1:8000"
# export KEA_DHCP4_HTTP_USERNAME="admin"
# export KEA_DHCP4_HTTP_PASSWORD="secret"
provider "kea" {
}
