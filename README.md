# Terraform Provider for Kea DHCP

A Terraform provider for managing [Kea DHCP](https://www.isc.org/kea/) server
resources.

## Overview

This provider enables Infrastructure as Code management of Kea DHCP servers
through their control channel API. It allows you to manage DHCP reservations
programmatically without requiring server restarts.

Currently supported:
- DHCPv4 host reservations (resource and data source)
- DHCPv4 client classes (resource and data source)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building from source)
- A Kea DHCP server with a control channel configured

### Kea Server Configuration

This provider communicates directly with Kea DHCP servers via their control
channel. You must configure either:

- **UNIX domain socket** - For local connections to the server
- **HTTP control socket** - For remote connections (requires the `http` hook
  library)

**Important:** This provider does **not** work with the Kea Control Agent
(`kea-ctrl-agent`). The Control Agent is deprecated in favor of direct HTTP
control sockets on the DHCP services themselves. Configure the `http` hook
library directly on your `kea-dhcp4` (or `kea-dhcp6`) service instead.

Each resource needs a hook library on the Kea server: `host_cmds` for host
reservations, `class_cmds` for client classes. Host reservations go to the
hosts database when the server has one, otherwise to its configuration file.

See [`testing/kea-dhcp4.conf`](testing/kea-dhcp4.conf) for a working example of
control socket and hook library configuration.

## Installation

### Terraform Registry

```terraform
terraform {
  required_providers {
    kea = {
      source = "wfdewith/kea"
    }
  }
}
```

### Building from Source

```sh
git clone https://github.com/wfdewith/terraform-provider-kea.git
cd terraform-provider-kea
make build
```

To use a locally built provider, add a dev override to your `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "wfdewith/kea" = "/path/to/terraform-provider-kea"
  }
  direct {}
}
```

## Quick Start

Configure the provider to connect to your Kea server via HTTP:

```terraform
provider "kea" {
  dhcp4 = {
    address = "http://kea-server:8000"
  }
}
```

Or via UNIX socket for local connections:

```terraform
provider "kea" {
  dhcp4 = {
    address = "unix:///run/kea/kea-dhcp4.sock"
  }
}
```

Or via environment variables:

```sh
export KEA_DHCP4_ADDRESS="http://kea-server:8000"
export KEA_DHCP4_HTTP_USERNAME="admin"  # optional
export KEA_DHCP4_HTTP_PASSWORD="secret" # optional
```

```terraform
provider "kea" {}
```

Every change to the running configuration is followed by `config-write`, so
that Kea saves it to the file it was started with and keeps it across
restarts. Set `config_write_filename` to write to another file in the same
directory.

Create a DHCP reservation:

```terraform
resource "kea_dhcp4_reservation" "webserver" {
  subnet_id  = 1
  hw_address = "aa:bb:cc:dd:ee:01"
  ip_address = "192.0.2.10"
  hostname   = "webserver"
}
```

See the [`examples/`](examples/) directory for more usage examples.

## Documentation

Full documentation is available in the [`docs/`](docs/) directory.

## License

Mozilla Public License 2.0 - see [LICENSE](LICENSE) for details.
