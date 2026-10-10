package dhcp4_test

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

// An address where nothing listens.
const testAccUnreachableAddress = "http://127.0.0.1:1"

func TestAccReservation_servers(t *testing.T) {
	mac := "02:5e:41:00:00:01"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)
	servers := testAccServers()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckServers(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			for _, server := range servers {
				reservation, err := testAccKeaClientAt(server).GetReservation(context.Background(), kea.OperationTargetMemory, query)
				if err != nil {
					return err
				}
				if reservation != nil {
					return fmt.Errorf("reservation still exists on %s", server)
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config:        testAccReservationConfig_servers(false, mac, "10.67.0.150", servers...),
				PostApplyFunc: testAccCheckReservationOnServers(t, query, "10.67.0.150", servers...),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config:        testAccReservationConfig_servers(false, mac, "10.67.0.151", servers...),
				PostApplyFunc: testAccCheckReservationOnServers(t, query, "10.67.0.151", servers...),
			},
		},
	})
}

func TestAccReservation_serversPartialWrite(t *testing.T) {
	mac := "02:5e:41:00:00:02"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)
	first := os.Getenv("KEA_DHCP4_ADDRESS")
	t.Cleanup(func() {
		_ = testAccKeaClientAt(first).DeleteReservation(context.Background(), kea.OperationTargetMemory, query)
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckServers(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_servers(false, mac, "10.67.0.152", first, testAccUnreachableAddress),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`reservation-add\s+succeeded\s+on\s+%s\s+and\s+failed\s+on\s+%s`,
					regexp.QuoteMeta(first), regexp.QuoteMeta(testAccUnreachableAddress))),
			},
		},
	})
}

func TestAccReservation_serversStrictSync(t *testing.T) {
	mac := "02:5e:41:00:00:03"
	ip := "10.67.0.153"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)
	servers := testAccServers()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckServers(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccReservationConfig_servers(true, mac, ip, servers...),
				PostApplyFunc: testAccCheckReservationOnServers(t, query, ip, servers...),
			},
			{
				PreConfig: func() {
					other := netip.MustParseAddr("10.67.0.154")
					reservation := keadhcp4.Reservation{
						SubnetID:  1,
						HWAddress: kea.HexID{0x02, 0x5e, 0x41, 0x00, 0x00, 0x03},
						IPAddress: &other,
					}
					err := testAccKeaClientAt(servers[1]).UpdateReservation(context.Background(), kea.OperationTargetMemory, reservation)
					if err != nil {
						t.Fatalf("failed to update reservation on %s: %v", servers[1], err)
					}
				},
				Config: testAccReservationConfig_servers(true, mac, ip, servers...),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`servers\s+out\s+of\s+sync:\s+%s\s+answers\s+differently\s+to\s+reservation-get`,
					regexp.QuoteMeta(servers[1]))),
			},
			{
				// Without strict_sync the first server's answer is used
				Config: testAccReservationConfig_servers(false, mac, ip, servers...),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccReservation_serversUnavailable(t *testing.T) {
	mac := "02:5e:41:00:00:05"
	ip := "10.67.0.156"
	first := os.Getenv("KEA_DHCP4_ADDRESS")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckServers(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_servers(false, mac, ip, first),
			},
			{
				Config: testAccReservationConfig_servers(false, mac, ip, testAccUnreachableAddress, first),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccReservationConfig_servers(true, mac, ip, first, testAccUnreachableAddress),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`servers\s+out\s+of\s+sync:\s+%s\s+did\s+not\s+answer\s+reservation-get`,
					regexp.QuoteMeta(testAccUnreachableAddress))),
			},
			{
				Config: testAccReservationConfig_servers(false, mac, ip, first),
			},
		},
	})
}

func TestAccClientClass_servers(t *testing.T) {
	name := "servers-class"
	servers := testAccServers()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckServers(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			for _, server := range servers {
				class, err := testAccKeaClientAt(server).GetClientClass(context.Background(), name)
				if err != nil {
					return err
				}
				if class != nil {
					return fmt.Errorf("client class still exists on %s", server)
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfigServers(false, servers...) + fmt.Sprintf(`
resource "kea_dhcp4_client_class" "test" {
  name = %q
  test = "option[60].text == 'servers'"
}
`, name),
				PostApplyFunc: func() {
					for _, server := range servers {
						class, err := testAccKeaClientAt(server).GetClientClass(context.Background(), name)
						if err != nil {
							t.Fatalf("failed to get client class from %s: %v", server, err)
						}
						if class == nil {
							t.Fatalf("client class does not exist on %s", server)
						}
					}
				},
			},
		},
	})
}

func testAccServers() []string {
	return []string{os.Getenv("KEA_DHCP4_ADDRESS"), os.Getenv("KEA_DHCP4_ADDRESS_2")}
}

// testAccKeaClientAt uses the credentials of KEA_DHCP4_ADDRESS, which both
// test servers share.
func testAccKeaClientAt(address string) *keadhcp4.Client {
	return keadhcp4.NewClient(&kea.HTTPTransport{
		Endpoint: address,
		Username: os.Getenv("KEA_DHCP4_HTTP_USERNAME"),
		Password: os.Getenv("KEA_DHCP4_HTTP_PASSWORD"),
	})
}

func testAccCheckReservationOnServers(t *testing.T, query keaquery.ReservationQuery, ip string, servers ...string) func() {
	return func() {
		for _, server := range servers {
			reservation, err := testAccKeaClientAt(server).GetReservation(context.Background(), kea.OperationTargetMemory, query)
			if err != nil {
				t.Fatalf("failed to get reservation from %s: %v", server, err)
			}
			if reservation == nil {
				t.Fatalf("reservation does not exist on %s", server)
			}
			if reservation.IPAddress == nil || reservation.IPAddress.String() != ip {
				t.Fatalf("reservation on %s has ip-address %v, expected %s", server, reservation.IPAddress, ip)
			}
		}
	}
}

func testAccReservationConfig_servers(strict bool, mac, ip string, servers ...string) string {
	return acctest.ProviderConfigServers(strict, servers...) + fmt.Sprintf(`
resource "kea_dhcp4_reservation" "test" {
  subnet_id        = 1
  hw_address       = %q
  ip_address       = %q
  operation_target = "memory"
}
`, mac, ip)
}
