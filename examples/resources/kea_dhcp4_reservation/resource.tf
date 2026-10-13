# Basic DHCP reservation using MAC address
resource "kea_dhcp4_reservation" "server01" {
  subnet_id  = 1
  hw_address = "aa:bb:cc:dd:ee:01"
  ip_address = "192.0.2.10"
}

# Reservation using client ID
resource "kea_dhcp4_reservation" "server02" {
  subnet_id  = 1
  client_id  = "01:aa:bb:cc:dd:ee:02"
  ip_address = "192.0.2.11"
}

# Global reservation (not bound to a specific subnet)
resource "kea_dhcp4_reservation" "global_device" {
  subnet_id  = 0
  hw_address = "aa:bb:cc:dd:ee:ff"
  ip_address = "192.0.2.100"
}

# Advanced reservation with DHCP options
resource "kea_dhcp4_reservation" "web_server" {
  subnet_id  = 2
  hw_address = "aa:bb:cc:dd:ee:10"
  ip_address = "198.51.100.10"
  hostname   = "web-server"

  option_data {
    name = "domain-name-servers"
    data = "198.51.100.1, 198.51.100.2"
  }

  option_data {
    name = "domain-name"
    data = "example.com"
  }

  option_data {
    name = "interface-mtu"
    data = "9000"
  }
}

# PXE boot configuration for network booting
resource "kea_dhcp4_reservation" "pxe_client" {
  subnet_id      = 2
  hw_address     = "aa:bb:cc:dd:ee:20"
  ip_address     = "198.51.100.20"
  hostname       = "pxe-client"
  next_server    = "198.51.100.5"
  boot_file_name = "pxelinux.0"

  option_data {
    name = "tftp-server-name"
    data = "198.51.100.5"
  }
}

# Reservation with client classes
resource "kea_dhcp4_reservation" "classified_device" {
  subnet_id      = 3
  hw_address     = "aa:bb:cc:dd:ee:30"
  ip_address     = "203.0.113.10"
  hostname       = "iot-device"
  client_classes = ["IoT", "limited-bandwidth"]

  option_data {
    name = "dhcp-lease-time"
    data = "3600"
  }
}

# Reservation with hexadecimal option data
resource "kea_dhcp4_reservation" "vendor_device" {
  subnet_id  = 3
  hw_address = "aa:bb:cc:dd:ee:40"
  ip_address = "203.0.113.20"

  # Vendor-specific option (code 43) in hex format
  option_data {
    code       = 43
    data       = "01:04:c0:a8:01:01"
    csv_format = false
  }

  # Always send this option even if not requested
  option_data {
    name        = "domain-name-servers"
    data        = "203.0.113.1"
    always_send = true
  }
}

# Reservation with user context for metadata tracking
resource "kea_dhcp4_reservation" "monitored_device" {
  subnet_id  = 3
  hw_address = "aa:bb:cc:dd:ee:50"
  ip_address = "203.0.113.30"
  hostname   = "monitored-server"

  # Arbitrary JSON metadata stored with the reservation
  user_context = jsonencode({
    department  = "engineering"
    owner       = "network-team"
    environment = "production"
    tags        = ["critical", "monitored", "high-priority"]
    metadata = {
      location    = "datacenter-east-1"
      rack        = "A-42"
      last_update = "2026-01-22"
    }
  })
}
