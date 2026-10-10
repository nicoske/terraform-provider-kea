package dhcp4_test

import (
	"context"
	"fmt"
	"net/netip"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

func TestAccConfigHashDataSource_basic(t *testing.T) {
	servers := testAccServers()
	dataSourceName := "data.kea_dhcp4_config_hash.test"
	var hashBefore string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckServers(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfigHashDataSourceConfig(servers[1]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "hashes.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "in_sync", "true"),
					testAccCheckConfigHash(dataSourceName, servers[1], &hashBefore),
				),
			},
			{
				Config: testAccConfigHashDataSourceConfig(servers...),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "hashes.%", "2"),
					resource.TestCheckResourceAttrSet(dataSourceName, "hashes."+servers[0]),
					// The test servers have different lease and hosts databases
					resource.TestCheckResourceAttr(dataSourceName, "in_sync", "false"),
					resource.TestCheckResourceAttrPtr(dataSourceName, "hashes."+servers[1], &hashBefore),
				),
			},
			{
				PreConfig: func() {
					testAccAddReservationOutOfBand(t, servers[1], "02:5e:41:00:00:04", "10.67.0.155")
				},
				Config: testAccConfigHashDataSourceConfig(servers...),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "in_sync", "false"),
					testAccCheckConfigHashChanged(dataSourceName, servers[1], hashBefore),
				),
			},
		},
	})
}

func testAccCheckConfigHash(name, server string, hash *string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(name, "hashes."+server, func(value string) error {
		want, err := testAccKeaClientAt(server).ConfigHash(context.Background())
		if err != nil {
			return err
		}
		if value != want {
			return fmt.Errorf("hash of %s is %s, expected %s", server, value, want)
		}
		*hash = value
		return nil
	})
}

func testAccCheckConfigHashChanged(name, server, before string) resource.TestCheckFunc {
	var after string
	return resource.ComposeAggregateTestCheckFunc(
		testAccCheckConfigHash(name, server, &after),
		func(*terraform.State) error {
			if after == before {
				return fmt.Errorf("hash of %s did not change", server)
			}
			return nil
		},
	)
}

func testAccAddReservationOutOfBand(t *testing.T, server, mac, ip string) {
	t.Helper()

	hwAddress, err := kea.ParseHexID(mac)
	if err != nil {
		t.Fatal(err)
	}
	addr := netip.MustParseAddr(ip)
	client := testAccKeaClientAt(server)
	err = client.AddReservation(context.Background(), kea.OperationTargetMemory, keadhcp4.Reservation{
		SubnetID:  1,
		HWAddress: hwAddress,
		IPAddress: &addr,
	})
	if err != nil {
		t.Fatalf("failed to add reservation on %s: %v", server, err)
	}
	t.Cleanup(func() {
		_ = client.DeleteReservation(context.Background(), kea.OperationTargetMemory,
			keaquery.ReservationByIdentifier(1, "hw-address", mac))
	})
}

func testAccConfigHashDataSourceConfig(servers ...string) string {
	return acctest.ProviderConfigServers(false, servers...) + `
data "kea_dhcp4_config_hash" "test" {}
`
}
