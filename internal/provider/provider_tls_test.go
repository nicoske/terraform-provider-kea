package provider_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
)

func TestAccTLSConnectivity(t *testing.T) {
	address := os.Getenv("KEA_DHCP4_TLS_ADDRESS")
	if address == "" {
		t.Skip("KEA_DHCP4_TLS_ADDRESS must be set for the TLS connectivity check")
	}

	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

data "kea_dhcp4_reservation" "test" {
  subnet_id  = 1
  hw_address = %q
}
`, acctest.ProviderConfigTLS(t, address), acctest.StaticSubnetMAC),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "ip_address", acctest.StaticSubnetIP),
				),
			},
		},
	})
}
