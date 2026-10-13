data "kea_dhcp4_config_hash" "current" {}

output "kea_config_hash" {
  value = data.kea_dhcp4_config_hash.current.hash
}
