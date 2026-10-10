# Class matching a vendor class identifier
resource "kea_dhcp4_client_class" "voip" {
  name = "voip-phones"
  test = "substring(option[60].hex, 0, 6) == 'Aastra'"

  option_data {
    name = "tftp-server-name"
    data = "192.0.2.5"
  }
}

# PXE boot for UEFI clients
resource "kea_dhcp4_client_class" "pxe_uefi" {
  name           = "pxe-uefi"
  test           = "option[93].hex == 0x0007"
  next_server    = "192.0.2.5"
  boot_file_name = "grubx64.efi"
  valid_lifetime = 600
}

# Class with a custom option definition
resource "kea_dhcp4_client_class" "provisioning" {
  name = "provisioning"
  test = "option[60].text == 'provisioning'"

  option_def {
    name = "provisioning-url"
    code = 224
    type = "string"
  }

  option_data {
    name = "provisioning-url"
    data = "https://provisioning.example.com/"
  }
}

# Class only evaluated when a subnet or pool lists it in
# evaluate-additional-classes. Classes are evaluated in definition order,
# so depends_on keeps it after the classes it refers to.
resource "kea_dhcp4_client_class" "late" {
  name                    = "late-voip"
  test                    = "member('voip-phones')"
  only_in_additional_list = true

  option_data {
    name = "domain-name"
    data = "voip.example.com"
  }

  depends_on = [kea_dhcp4_client_class.voip]
}

# Template class spawning one subclass per client identifier prefix
resource "kea_dhcp4_client_class" "per_vendor" {
  name          = "per-vendor"
  template_test = "substring(option[61].hex, 0, 3)"
}
