# Basic subnet with one pool
resource "kea_dhcp4_subnet" "office" {
  id     = 10
  subnet = "192.0.2.0/24"

  pools {
    pool = "192.0.2.100 - 192.0.2.200"
  }

  option_data {
    name = "routers"
    data = "192.0.2.1"
  }

  option_data {
    name = "domain-name-servers"
    data = "192.0.2.53, 192.0.2.54"
  }
}

# Subnet reached through relay agents, with lease timers
resource "kea_dhcp4_subnet" "branch" {
  id     = 20
  subnet = "198.51.100.0/24"
  relay  = ["198.51.100.1"]

  valid_lifetime = 3600
  renew_timer    = 900
  rebind_timer   = 1800

  pools {
    pool = "198.51.100.10 - 198.51.100.99"
  }

  # Pool restricted to a client class, with its own options
  pools {
    pool           = "198.51.100.128/26"
    client_classes = ["voip-phones"]

    option_data {
      name = "tftp-server-name"
      data = "198.51.100.5"
    }
  }

  option_data {
    name = "routers"
    data = "198.51.100.1"
  }
}

# Directly attached subnet with PXE boot and DNS updates
resource "kea_dhcp4_subnet" "lab" {
  id        = 30
  subnet    = "203.0.113.0/24"
  interface = "eth1"

  next_server    = "203.0.113.5"
  boot_file_name = "pxelinux.0"

  reservations_global    = false
  reservations_in_subnet = true

  ddns_send_updates      = true
  ddns_qualifying_suffix = "lab.example.org."

  pools {
    pool = "203.0.113.100 - 203.0.113.200"
  }

  user_context = jsonencode({
    site  = "lab"
    owner = "network-team"
  })
}
