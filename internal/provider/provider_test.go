package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
)

func TestProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig(),
			},
		},
	})
}

func TestProvider_tls(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file          = %q
    tls_client_cert_file = %q
    tls_client_key_file  = %q
`, acctest.TLSFile("ca.pem"), acctest.TLSFile("client.pem"), acctest.TLSFile("client-key.pem")),
				Check: resource.TestCheckResourceAttr("data.kea_dhcp4_reservation.test", "ip_address", acctest.StaticSubnetIP),
			},
		},
	})
}

func TestProvider_tlsServerName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file          = %q
    tls_client_cert_file = %q
    tls_client_key_file  = %q
    tls_server_name      = "kea-dhcp4"
`, acctest.TLSFile("ca.pem"), acctest.TLSFile("client.pem"), acctest.TLSFile("client-key.pem")),
				Check: resource.TestCheckResourceAttr("data.kea_dhcp4_reservation.test", "ip_address", acctest.StaticSubnetIP),
			},
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file          = %q
    tls_client_cert_file = %q
    tls_client_key_file  = %q
    tls_server_name      = "other.example"
`, acctest.TLSFile("ca.pem"), acctest.TLSFile("client.pem"), acctest.TLSFile("client-key.pem")),
				ExpectError: regexp.MustCompile(`not\s+other\.example`),
			},
		},
	})
}

func TestProvider_tlsEnvironment(t *testing.T) {
	acctest.PreCheckTLS(t)

	t.Setenv("KEA_DHCP4_ADDRESS", os.Getenv("KEA_DHCP4_TLS_ADDRESS"))
	t.Setenv("KEA_DHCP4_TLS_CA_FILE", acctest.TLSFile("ca.pem"))
	t.Setenv("KEA_DHCP4_TLS_CLIENT_CERT_FILE", acctest.TLSFile("client.pem"))
	t.Setenv("KEA_DHCP4_TLS_CLIENT_KEY_FILE", acctest.TLSFile("client-key.pem"))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig() + testAccProviderConfig_staticReservation(),
				Check:  resource.TestCheckResourceAttr("data.kea_dhcp4_reservation.test", "ip_address", acctest.StaticSubnetIP),
			},
		},
	})
}

func TestProvider_tlsWithoutClientCert(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file = %q
`, acctest.TLSFile("ca.pem")),
				ExpectError: regexp.MustCompile(`certificate\s+required`),
			},
		},
	})
}

func TestProvider_tlsUnknownCA(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_client_cert_file = %q
    tls_client_key_file  = %q
`, acctest.TLSFile("client.pem"), acctest.TLSFile("client-key.pem")),
				ExpectError: regexp.MustCompile(`signed\s+by\s+unknown\s+authority`),
			},
		},
	})
}

func TestProvider_tlsCertWithoutKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file          = %q
    tls_client_cert_file = %q
`, acctest.TLSFile("ca.pem"), acctest.TLSFile("client.pem")),
				ExpectError: regexp.MustCompile(`tls_client_cert_file\s+is\s+set\s+without\s+tls_client_key_file`),
			},
		},
	})
}

func testAccProviderConfig_tls(tlsAttributes string, args ...any) string {
	return fmt.Sprintf(`
provider "kea" {
  dhcp4 = {
    address = %q
%s  }
}
`, os.Getenv("KEA_DHCP4_TLS_ADDRESS"), fmt.Sprintf(tlsAttributes, args...)) + testAccProviderConfig_staticReservation()
}

func testAccProviderConfig_staticReservation() string {
	return fmt.Sprintf(`
data "kea_dhcp4_reservation" "test" {
  subnet_id  = 1
  hw_address = %q
}
`, acctest.StaticSubnetMAC)
}
