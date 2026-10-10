data "kea_dhcp4_config_hash" "current" {}

output "kea_config_hashes" {
  value = data.kea_dhcp4_config_hash.current.hashes
}
