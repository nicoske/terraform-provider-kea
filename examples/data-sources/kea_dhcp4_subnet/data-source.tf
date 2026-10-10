# Look up a subnet by ID
data "kea_dhcp4_subnet" "office" {
  id = 10
}

# Look up a subnet by prefix
data "kea_dhcp4_subnet" "branch" {
  subnet = "198.51.100.0/24"
}

output "office_pools" {
  value = [for p in data.kea_dhcp4_subnet.office.pools : p.pool]
}

# Reserve an address in a subnet found by prefix
resource "kea_dhcp4_reservation" "printer" {
  subnet_id  = data.kea_dhcp4_subnet.branch.id
  hw_address = "aa:bb:cc:dd:ee:70"
  ip_address = "198.51.100.70"
}
