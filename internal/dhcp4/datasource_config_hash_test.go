package dhcp4_test

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

func TestAccConfigHashDataSource_basic(t *testing.T) {
	dataSourceName := "data.kea_dhcp4_config_hash.test"
	var hashBefore string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckMemfile(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigHashDataSourceConfig(),
				Check:  testAccCheckConfigHash(dataSourceName, &hashBefore),
			},
			{
				PreConfig: func() {
					testAccAddReservationOutOfBand(t, "02:5e:41:00:00:04", "10.67.0.155")
				},
				Config: testAccConfigHashDataSourceConfig(),
				Check:  testAccCheckConfigHashChanged(dataSourceName, hashBefore),
			},
		},
	})
}

// testAccMemfileClient queries the server of acctest.PreCheckMemfile, which
// shares the credentials of KEA_DHCP4_ADDRESS.
func testAccMemfileClient() *keadhcp4.Client {
	return keadhcp4.NewClient(&kea.HTTPTransport{
		Endpoint: os.Getenv("KEA_DHCP4_MEMFILE_ADDRESS"),
		Username: os.Getenv("KEA_DHCP4_HTTP_USERNAME"),
		Password: os.Getenv("KEA_DHCP4_HTTP_PASSWORD"),
	})
}

func testAccCheckConfigHash(name string, hash *string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(name, "hash", func(value string) error {
		want, err := testAccMemfileClient().ConfigHash(context.Background())
		if err != nil {
			return err
		}
		if value != want {
			return fmt.Errorf("hash is %s, expected %s", value, want)
		}
		*hash = value
		return nil
	})
}

func testAccCheckConfigHashChanged(name, before string) resource.TestCheckFunc {
	var after string
	return resource.ComposeAggregateTestCheckFunc(
		testAccCheckConfigHash(name, &after),
		func(*terraform.State) error {
			if after == before {
				return fmt.Errorf("hash did not change")
			}
			return nil
		},
	)
}

func testAccAddReservationOutOfBand(t *testing.T, mac, ip string) {
	t.Helper()

	hwAddress, err := kea.ParseHexID(mac)
	if err != nil {
		t.Fatal(err)
	}
	addr := netip.MustParseAddr(ip)
	client := testAccMemfileClient()
	err = client.AddReservation(context.Background(), kea.OperationTargetMemory, keadhcp4.Reservation{
		SubnetID:  1,
		HWAddress: hwAddress,
		IPAddress: &addr,
	})
	if err != nil {
		t.Fatalf("failed to add reservation: %v", err)
	}
	t.Cleanup(func() {
		_ = client.DeleteReservation(context.Background(), kea.OperationTargetMemory,
			keaquery.ReservationByIdentifier(1, "hw-address", mac))
	})
}

func testAccConfigHashDataSourceConfig() string {
	return acctest.ProviderConfigMemfile() + `
data "kea_dhcp4_config_hash" "test" {}
`
}
