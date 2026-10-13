package dhcp4_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
)

func TestAccReservationDataSource_basic(t *testing.T) {
	mac := "02:3c:f1:86:a4:2e"
	ip := "10.67.0.19"
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_basic(1, mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccReservationDataSource_withClientID(t *testing.T) {
	clientID := "01:aa:bb:dd"
	ip := "10.67.0.77"
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_withClientID(1, clientID, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "client_id", clientID),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccReservationDataSource_withCircuitID(t *testing.T) {
	circuitID := "05:06:07:08"
	ip := "10.67.0.78"
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_withCircuitID(1, circuitID, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "circuit_id", circuitID),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccReservationDataSource_withDUID(t *testing.T) {
	duid := "00:03:00:01:ca:fe:ba:be:00:01"
	ip := "10.67.0.79"
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_withDUID(1, duid, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "duid", duid),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccReservationDataSource_withFlexID(t *testing.T) {
	flexID := "07:08:09:0a:0b:0c"
	ip := "10.67.0.80"
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_withFlexID(1, flexID, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "flex_id", flexID),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccReservationDataSource_withHostname(t *testing.T) {
	mac := "02:b9:47:d2:6f:91"
	ip := "10.67.0.128"
	hostname := fmt.Sprintf("test-host-%s", mac[12:])
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_withHostname(1, mac, ip, hostname),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname),
				),
			},
		},
	})
}

func TestAccReservationDataSource_global(t *testing.T) {
	mac := "02:7e:93:15:cb:a8"
	ip := "192.168.67.88"
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_global(mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subnet_id", "0"),
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
				),
			},
		},
	})
}

func TestAccReservationDataSource_byIP(t *testing.T) {
	mac := "02:4a:8c:f3:b2:d7"
	ip := "10.67.0.45"
	hostname := "test-host-by-ip"
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_byIP(1, mac, ip, hostname),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subnet_id", "1"),
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccReservationDataSource_staticByHWAddress(t *testing.T) {
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_staticByHWAddress(1, acctest.StaticSubnetMAC),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", acctest.StaticSubnetMAC),
					resource.TestCheckResourceAttr(resourceName, "ip_address", acctest.StaticSubnetIP),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccReservationDataSource_staticByIP(t *testing.T) {
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_staticByIP(1, acctest.StaticSubnetIP),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", acctest.StaticSubnetMAC),
					resource.TestCheckResourceAttr(resourceName, "ip_address", acctest.StaticSubnetIP),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccReservationDataSource_staticGlobal(t *testing.T) {
	resourceName := "data.kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationDataSourceConfig_staticByHWAddress(0, acctest.StaticGlobalMAC),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subnet_id", "0"),
					resource.TestCheckResourceAttr(resourceName, "hw_address", acctest.StaticGlobalMAC),
					resource.TestCheckResourceAttr(resourceName, "ip_address", acctest.StaticGlobalIP),
				),
			},
		},
	})
}

func testAccReservationDataSourceConfig_basic(subnetID uint32, mac, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  hw_address = %q
  ip_address = %q
}

data "kea_dhcp4_reservation" "test" {
  subnet_id  = kea_dhcp4_reservation.test.subnet_id
  hw_address = kea_dhcp4_reservation.test.hw_address
}
`, acctest.ProviderConfig(), subnetID, mac, ip)
}

func testAccReservationDataSourceConfig_withClientID(subnetID uint32, clientID, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  client_id  = %q
  ip_address = %q
}

data "kea_dhcp4_reservation" "test" {
  subnet_id = kea_dhcp4_reservation.test.subnet_id
  client_id = kea_dhcp4_reservation.test.client_id
}
`, acctest.ProviderConfig(), subnetID, clientID, ip)
}

func testAccReservationDataSourceConfig_withCircuitID(subnetID uint32, circuitID, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  circuit_id = %q
  ip_address = %q
}

data "kea_dhcp4_reservation" "test" {
  subnet_id  = kea_dhcp4_reservation.test.subnet_id
  circuit_id = kea_dhcp4_reservation.test.circuit_id
}
`, acctest.ProviderConfig(), subnetID, circuitID, ip)
}

func testAccReservationDataSourceConfig_withDUID(subnetID uint32, duid, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  duid       = %q
  ip_address = %q
}

data "kea_dhcp4_reservation" "test" {
  subnet_id = kea_dhcp4_reservation.test.subnet_id
  duid      = kea_dhcp4_reservation.test.duid
}
`, acctest.ProviderConfig(), subnetID, duid, ip)
}

func testAccReservationDataSourceConfig_withFlexID(subnetID uint32, flexID, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  flex_id    = %q
  ip_address = %q
}

data "kea_dhcp4_reservation" "test" {
  subnet_id = kea_dhcp4_reservation.test.subnet_id
  flex_id   = kea_dhcp4_reservation.test.flex_id
}
`, acctest.ProviderConfig(), subnetID, flexID, ip)
}

func testAccReservationDataSourceConfig_withHostname(subnetID uint32, mac, ip, hostname string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  hw_address = %q
  ip_address = %q
  hostname   = %q
}

data "kea_dhcp4_reservation" "test" {
  subnet_id  = kea_dhcp4_reservation.test.subnet_id
  hw_address = kea_dhcp4_reservation.test.hw_address
}
`, acctest.ProviderConfig(), subnetID, mac, ip, hostname)
}

func testAccReservationDataSourceConfig_global(mac, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = 0
  hw_address = %q
  ip_address = %q
}

data "kea_dhcp4_reservation" "test" {
  subnet_id  = kea_dhcp4_reservation.test.subnet_id
  hw_address = kea_dhcp4_reservation.test.hw_address
}
`, acctest.ProviderConfig(), mac, ip)
}

func testAccReservationDataSourceConfig_byIP(subnetID uint32, mac, ip, hostname string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  hw_address = %q
  ip_address = %q
  hostname   = %q
}

data "kea_dhcp4_reservation" "test" {
  subnet_id  = kea_dhcp4_reservation.test.subnet_id
  ip_address = kea_dhcp4_reservation.test.ip_address
}
`, acctest.ProviderConfig(), subnetID, mac, ip, hostname)
}

func testAccReservationDataSourceConfig_staticByHWAddress(subnetID uint32, mac string) string {
	return fmt.Sprintf(`
%s

data "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  hw_address = %q
}
`, acctest.ProviderConfig(), subnetID, mac)
}

func testAccReservationDataSourceConfig_staticByIP(subnetID uint32, ip string) string {
	return fmt.Sprintf(`
%s

data "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  ip_address = %q
}
`, acctest.ProviderConfig(), subnetID, ip)
}
