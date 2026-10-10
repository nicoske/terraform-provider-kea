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
- DHCPv4 subnets with pools (resource and data source)
- Running configuration hash per server (data source)

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
- **HTTPS control socket** - For remote connections over TLS, optionally with
  client certificates (`cert-required`)

**Important:** This provider does **not** work with the Kea Control Agent
(`kea-ctrl-agent`). The Control Agent is deprecated in favor of direct HTTP
control sockets on the DHCP services themselves. Configure the `http` hook
library directly on your `kea-dhcp4` (or `kea-dhcp6`) service instead.

Each resource needs a hook library on the Kea server: `host_cmds` for host
reservations, `class_cmds` for client classes, `subnet_cmds` for subnets.

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

Or via HTTPS with a private CA and a client certificate for mutual TLS:

```terraform
provider "kea" {
  dhcp4 = {
    address              = "https://kea-server:8000"
    tls_ca_file          = "/etc/terraform/kea/ca.pem"
    tls_client_cert_file = "/etc/terraform/kea/client.pem"
    tls_client_key_file  = "/etc/terraform/kea/client-key.pem"
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

Changes made through the control channel live in memory only. Set
`config_write = true` (or `KEA_DHCP4_CONFIG_WRITE=true`) to send `config-write`
after every change so that Kea saves its configuration to disk:

```terraform
provider "kea" {
  dhcp4 = {
    address      = "http://kea-server:8000"
    config_write = true
  }
}
```

### Several servers

Kea high availability peers do not replicate configuration changes, so each
peer must receive every change. List the peers in `servers` to send every
change to each of them in order:

```terraform
provider "kea" {
  dhcp4 = {
    servers = [
      { address = "http://kea-1:8000" },
      { address = "http://kea-2:8000" },
    ]
  }
}
```

A change stops at the first server that fails, and the error names the
servers that already applied it. Reads ask every server, use the answer of
the first server that answers and warn when another one answers differently
or does not answer; `strict_sync = true` makes that an error.

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
