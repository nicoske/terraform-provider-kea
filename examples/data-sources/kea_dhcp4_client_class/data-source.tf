# Look up a client class by name
data "kea_dhcp4_client_class" "voip" {
  name = "voip-phones"
}

output "voip_test" {
  value = data.kea_dhcp4_client_class.voip.test
}

output "voip_options" {
  value = data.kea_dhcp4_client_class.voip.option_data
}
