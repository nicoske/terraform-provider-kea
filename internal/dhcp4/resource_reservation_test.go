package dhcp4_test

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

func TestAccReservation_basic(t *testing.T) {
	mac := "02:A3:7B:4E:91:22"
	ip := "10.67.0.42"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_basic(1, mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_withClientID(t *testing.T) {
	clientID := "01:aa:bb:cc"
	ip := "10.67.0.74"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "client-id", clientID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withClientID(1, clientID, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "client_id", clientID),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_withCircuitID(t *testing.T) {
	circuitID := "01:02:03:04"
	ip := "10.67.0.73"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "circuit-id", circuitID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withCircuitID(1, circuitID, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "circuit_id", circuitID),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_withQuotedCircuitID(t *testing.T) {
	circuitID := "'charter950'"
	ip := "10.67.0.72"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "circuit-id", "63686172746572393530")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withCircuitID(1, circuitID, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "circuit_id", circuitID),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_withDUID(t *testing.T) {
	duid := "00:03:00:01:de:ad:be:ef:ca:fe"
	ip := "10.67.0.75"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "duid", duid)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withDUID(1, duid, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "duid", duid),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_withFlexID(t *testing.T) {
	flexID := "01:02:03:04:05:06"
	ip := "10.67.0.76"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "flex-id", flexID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withFlexID(1, flexID, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "flex_id", flexID),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_withHostname(t *testing.T) {
	mac := "02:f8:c2:5d:19:a6"
	ip := "10.67.0.142"
	hostname := fmt.Sprintf("test-host-%d", rand.Intn(10000))
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withHostname(1, mac, ip, hostname),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_withOptionData(t *testing.T) {
	mac := "02:6b:d9:31:84:cf"
	ip := "10.67.0.27"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withOptionData(1, mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "option_data.0.name", "domain-name-servers"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_withUserContext(t *testing.T) {
	mac := "02:5a:b6:4d:e9:72"
	ip := "10.67.0.64"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withUserContext(1, mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "user_context"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_global(t *testing.T) {
	mac := "02:95:4a:62:b8:e1"
	ip := "192.168.67.143"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(0, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_global(mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subnet_id", "0"),
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_destroy(t *testing.T) {
	mac := "02:9a:3f:6c:d1:84"
	ip := "10.67.0.201"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_basic(1, mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kea_dhcp4_reservation.test", "hw_address", mac),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
			{
				Config:        acctest.ProviderConfig(),
				PostApplyFunc: testAccCheckReservationDestroyed(t, query),
			},
		},
	})
}

func TestAccReservation_disappears(t *testing.T) {
	mac := "02:1c:8e:a7:f2:3d"
	ip := "10.67.0.177"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_basic(1, mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
				),
				PostApplyFunc: testAccDeleteReservation(t, query),
			},
			{
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccReservation_update(t *testing.T) {
	mac := "02:d4:71:39:ac:56"
	ip := "10.67.0.91"
	hostname1 := fmt.Sprintf("test-host-%d", rand.Intn(10000))
	hostname2 := fmt.Sprintf("test-host-%d", rand.Intn(10000))
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withHostname(1, mac, ip, hostname1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname1),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
			{
				Config: testAccReservationConfig_withHostname(1, mac, ip, hostname2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hostname", hostname2),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
		},
	})
}

func TestAccReservation_changeIdentity(t *testing.T) {
	testCases := map[string]struct {
		configBefore string
		configAfter  string
		queryBefore  keaquery.ReservationQuery
		queryAfter   keaquery.ReservationQuery
	}{
		"hw_address": {
			configBefore: testAccReservationConfig_basic(1, "02:11:22:33:44:01", "10.67.0.211"),
			configAfter:  testAccReservationConfig_basic(1, "02:11:22:33:44:02", "10.67.0.211"),
			queryBefore:  keaquery.ReservationByIdentifier(1, "hw-address", "02:11:22:33:44:01"),
			queryAfter:   keaquery.ReservationByIdentifier(1, "hw-address", "02:11:22:33:44:02"),
		},
		"client_id": {
			configBefore: testAccReservationConfig_withClientID(1, "01:11:22:33:44:03", "10.67.0.212"),
			configAfter:  testAccReservationConfig_withClientID(1, "01:11:22:33:44:04", "10.67.0.212"),
			queryBefore:  keaquery.ReservationByIdentifier(1, "client-id", "01:11:22:33:44:03"),
			queryAfter:   keaquery.ReservationByIdentifier(1, "client-id", "01:11:22:33:44:04"),
		},
		"circuit_id": {
			configBefore: testAccReservationConfig_withCircuitID(1, "02:11:22:33:44:05", "10.67.0.213"),
			configAfter:  testAccReservationConfig_withCircuitID(1, "02:11:22:33:44:06", "10.67.0.213"),
			queryBefore:  keaquery.ReservationByIdentifier(1, "circuit-id", "02:11:22:33:44:05"),
			queryAfter:   keaquery.ReservationByIdentifier(1, "circuit-id", "02:11:22:33:44:06"),
		},
		"duid": {
			configBefore: testAccReservationConfig_withDUID(1, "00:03:00:01:11:22:33:44:55:11", "10.67.0.214"),
			configAfter:  testAccReservationConfig_withDUID(1, "00:03:00:01:11:22:33:44:55:12", "10.67.0.214"),
			queryBefore:  keaquery.ReservationByIdentifier(1, "duid", "00:03:00:01:11:22:33:44:55:11"),
			queryAfter:   keaquery.ReservationByIdentifier(1, "duid", "00:03:00:01:11:22:33:44:55:12"),
		},
		"flex_id": {
			configBefore: testAccReservationConfig_withFlexID(1, "02:11:22:33:44:07", "10.67.0.215"),
			configAfter:  testAccReservationConfig_withFlexID(1, "02:11:22:33:44:08", "10.67.0.215"),
			queryBefore:  keaquery.ReservationByIdentifier(1, "flex-id", "02:11:22:33:44:07"),
			queryAfter:   keaquery.ReservationByIdentifier(1, "flex-id", "02:11:22:33:44:08"),
		},
		"subnet_id": {
			configBefore: testAccReservationConfig_basic(1, "02:11:22:33:44:09", "10.67.0.216"),
			configAfter:  testAccReservationConfig_basic(2, "02:11:22:33:44:09", "10.67.1.216"),
			queryBefore:  keaquery.ReservationByIdentifier(1, "hw-address", "02:11:22:33:44:09"),
			queryAfter:   keaquery.ReservationByIdentifier(2, "hw-address", "02:11:22:33:44:09"),
		},
		"identifier_type": {
			configBefore: testAccReservationConfig_basic(1, "02:11:22:33:44:0a", "10.67.0.217"),
			configAfter:  testAccReservationConfig_withClientID(1, "01:11:22:33:44:0b", "10.67.0.217"),
			queryBefore:  keaquery.ReservationByIdentifier(1, "hw-address", "02:11:22:33:44:0a"),
			queryAfter:   keaquery.ReservationByIdentifier(1, "client-id", "01:11:22:33:44:0b"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:        tc.configBefore,
						PostApplyFunc: testAccCheckReservationExists(t, tc.queryBefore),
					},
					{
						Config:        tc.configAfter,
						PostApplyFunc: testAccCheckReservationReplaced(t, tc.queryBefore, tc.queryAfter),
					},
				},
			})
		})
	}
}

func TestAccReservation_reorderSetsNoUpdate(t *testing.T) {
	mac := "02:e7:2f:58:c3:9b"
	ip := "10.67.0.165"
	resourceName := "kea_dhcp4_reservation.test"
	query := keaquery.ReservationByIdentifier(1, "hw-address", mac)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withClientClassesAndOptions(1, mac, ip),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hw_address", mac),
					resource.TestCheckResourceAttr(resourceName, "ip_address", ip),
					resource.TestCheckResourceAttr(resourceName, "client_classes.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "3"),
				),
				PostApplyFunc: testAccCheckReservationExists(t, query),
			},
			{
				Config: testAccReservationConfig_withClientClassesAndOptionsReordered(1, mac, ip),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccReservation_optionDataNormalization(t *testing.T) {
	testCases := map[string]struct {
		mac        string
		ip         string
		blocks     []string
		minVersion string
	}{
		"non_canonical_hex": {
			mac: "02:11:22:33:88:01",
			ip:  "10.67.0.220",
			blocks: []string{`option_data {
    name       = "domain-name-servers"
    data       = "0a000001"
    csv_format = false
  }`},
		},
		"quoted_string": {
			mac: "02:11:22:33:88:02",
			ip:  "10.67.0.221",
			blocks: []string{`option_data {
    name       = "domain-name"
    data       = "'example.com'"
    csv_format = false
  }`},
		},
		"escaped_comma": {
			mac: "02:11:22:33:88:0e",
			ip:  "10.67.0.240",
			blocks: []string{`option_data {
    name       = "domain-name"
    data       = "example\\,com"
    csv_format = true
  }`},
		},
		"duplicate_client_classes": {
			mac:        "02:11:22:33:88:03",
			ip:         "10.67.0.222",
			minVersion: "3.1.0",
			blocks: []string{`option_data {
    name           = "domain-name-servers"
    data           = "0a000001"
    csv_format     = false
    client_classes = ["a"]
  }
  option_data {
    name           = "domain-name-servers"
    data           = "0a:00:00:02"
    csv_format     = false
    client_classes = ["b"]
  }`},
		},
		"undefined_code": {
			mac: "02:11:22:33:88:04",
			ip:  "10.67.0.223",
			blocks: []string{`option_data {
    code       = 224
    data       = "'hello'"
    csv_format = false
  }
  option_data {
    code = 6
    data = "10.0.0.1"
  }`},
		},
		"unresolved_undefined_code": {
			mac: "02:11:22:33:88:09",
			ip:  "10.67.0.228",
			blocks: []string{
				`option_data {
    code = 224
    data = "'hello'"
  }
  option_data {
    name = "domain-name-servers"
    data = "10.0.0.1, 10.0.0.2"
  }`,
				`option_data {
    code = 224
    data = "68:65:6c:6c:6f"
  }
  option_data {
    name = "domain-name-servers"
    data = "10.0.0.1, 10.0.0.2"
  }`,
			},
		},
		"mix_and_update": {
			mac: "02:11:22:33:88:08",
			ip:  "10.67.0.227",
			blocks: []string{
				`option_data {
    name       = "domain-name-servers"
    data       = "0a000001"
    csv_format = false
  }`,
				`option_data {
    name = "domain-name-servers"
    data = "10.0.0.1"
  }`,
				`option_data {
    name       = "domain-name-servers"
    data       = "0a 00 00 03"
    csv_format = false
  }`,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() {
					if tc.minVersion != "" {
						acctest.SkipUnlessKeaVersion(t, ">= "+tc.minVersion)
					}
				},
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Steps:                    optionDataSteps(1, tc.mac, tc.ip, tc.blocks...),
			})
		})
	}
}

func TestAccReservation_optionDataSameBytes(t *testing.T) {
	testCases := map[string]struct {
		mac    string
		ip     string
		blocks string
	}{
		"two": {
			mac: "02:11:22:33:88:05",
			ip:  "10.67.0.224",
			blocks: `option_data {
    name       = "domain-name-servers"
    data       = "0a000001"
    csv_format = false
  }
  option_data {
    name       = "routers"
    data       = "0A:00:00:01"
    csv_format = false
  }`,
		},
		"three": {
			mac: "02:11:22:33:88:06",
			ip:  "10.67.0.225",
			blocks: `option_data {
    name       = "domain-name-servers"
    data       = "0a 00 00 01"
    csv_format = false
  }
  option_data {
    name       = "routers"
    data       = "0a000001"
    csv_format = false
  }
  option_data {
    name       = "time-servers"
    data       = "0x0A000001"
    csv_format = false
  }`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Steps:                    optionDataSteps(1, tc.mac, tc.ip, tc.blocks),
			})
		})
	}
}

func TestAccReservation_optionDataSameBytesMany(t *testing.T) {
	names := []string{"domain-name-servers", "routers", "time-servers", "name-servers", "log-servers", "cookie-servers", "lpr-servers", "impress-servers"}
	spellings := []string{"0a000001", "0A:00:00:01", "0x0a000001", "0a 00 00 01", "a000001", "0A000001", "0a:0:0:1", "0xA000001"}
	macs := []string{"02:11:22:33:88:11", "02:11:22:33:88:12", "02:11:22:33:88:13", "02:11:22:33:88:14"}
	ips := []string{"10.67.0.236", "10.67.0.237", "10.67.0.238", "10.67.0.239"}

	for rot := 0; rot < 4; rot++ {
		var blocks string
		for i, n := range names {
			blocks += fmt.Sprintf("  option_data {\n    name       = %q\n    data       = %q\n    csv_format = false\n  }\n", n, spellings[(i+rot*3)%len(spellings)])
		}

		t.Run(fmt.Sprint(rot), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Steps:                    optionDataSteps(1, macs[rot], ips[rot], blocks),
			})
		})
	}
}

func TestAccReservation_optionDataRespell(t *testing.T) {
	mac := "02:11:22:33:88:0a"
	ip := "10.67.0.229"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccReservationConfig_withOptionDataBlocks(1, mac, ip, `option_data {
    name       = "domain-name-servers"
    data       = "0a000001"
    csv_format = false
  }
  option_data {
    name       = "routers"
    data       = "0a000001"
    csv_format = false
  }`)},
			{
				Config: testAccReservationConfig_withOptionDataBlocks(1, mac, ip, `option_data {
    name       = "domain-name-servers"
    data       = "0A:00:00:01"
    csv_format = false
  }
  option_data {
    name       = "routers"
    data       = "0x0a000001"
    csv_format = false
  }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectNonEmptyPlan()},
				},
			},
			{
				Config: testAccReservationConfig_withOptionDataBlocks(1, mac, ip, `option_data {
    name       = "domain-name-servers"
    data       = "0A:00:00:01"
    csv_format = false
    always_send = true
  }
  option_data {
    name       = "routers"
    data       = "0x0a000001"
    csv_format = false
  }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectNonEmptyPlan()},
				},
			},
		},
	})
}

func TestAccReservation_optionDataUnresolvedRespell(t *testing.T) {
	mac := "02:11:22:33:88:0b"
	ip := "10.67.0.230"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccReservationConfig_withOptionDataBlocks(1, mac, ip, `option_data {
    code = 224
    data = "'hello'"
  }`)},
			{
				Config: testAccReservationConfig_withOptionDataBlocks(1, mac, ip, `option_data {
    code = 224
    data = "68656C6C6F"
  }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectNonEmptyPlan()},
				},
			},
			{
				Config: testAccReservationConfig_withOptionDataBlocks(1, mac, ip, `option_data {
    code       = 224
    data       = "68656C6C6F"
    csv_format = false
  }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccReservation_optionDataInvalidAtPlan(t *testing.T) {
	mac := "02:11:22:33:88:0c"
	ip := "10.67.0.231"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testAccReservationConfig_withOptionDataBlocks(1, mac, ip, `option_data {
    name       = "domain-name-servers"
    data       = "zz"
    csv_format = false
  }`),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile("Invalid Option Data"),
		}},
	})
}

func TestAccReservation_optionDataCSVForUndefinedFails(t *testing.T) {
	mac := "02:11:22:33:88:0d"
	ip := "10.67.0.232"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testAccReservationConfig_withOptionDataBlocks(1, mac, ip, `option_data {
    code       = 224
    data       = "hello"
    csv_format = true
  }`),
			ExpectError: regexp.MustCompile("Error Adding Reservation"),
		}},
	})
}

func TestAccReservation_optionDataClientClassesUnsupported(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.SkipUnlessKeaVersion(t, "< 3.1.0")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testAccReservationConfig_withOptionDataBlocks(1, "02:11:22:33:88:0f", "10.67.0.241", `option_data {
    name           = "domain-name-servers"
    data           = "0a000001"
    csv_format     = false
    client_classes = ["a"]
  }`),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile("Unsupported Kea Version"),
		}},
	})
}

func TestAccReservation_import(t *testing.T) {
	mac := "02:fa:c9:88:fa:69"
	ip := "10.67.0.77"
	resourceName := "kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withClientClassesAndOptions(1, mac, ip),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccReservation_importInvalid(t *testing.T) {
	mac := "02:ce:7e:2a:fc:ba"
	ip := "10.67.0.57"
	resourceName := "kea_dhcp4_reservation.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccReservationConfig_withClientClassesAndOptions(1, mac, ip),
			},
			{
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "foo",
				ExpectError:   regexp.MustCompile(`Error Parsing Reservation ID`),
			},
			{
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "foo/bar/baz",
				ExpectError:   regexp.MustCompile(`Error Parsing Reservation ID`),
			},
			{
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "1/bar/baz",
				ExpectError:   regexp.MustCompile(`Error Parsing Reservation ID`),
			},
			{
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "1/hw-address/baz",
				ExpectError:   regexp.MustCompile(`Error Parsing Reservation ID`),
			},
			{
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: "1/ip-address/10.67.0.57",
				ExpectError:   regexp.MustCompile(`Error Parsing Reservation ID`),
			},
		},
	})
}

func testAccCheckReservationExists(t *testing.T, query keaquery.ReservationQuery) func() {
	return func() {
		client := acctest.NewClient(t)

		reservation, err := client.GetReservation(context.Background(), kea.OperationTargetDatabase, query)
		if err != nil {
			t.Fatalf("failed to get reservation: %v", err)
		}

		if reservation == nil {
			t.Fatal("reservation does not exist in Kea")
		}
	}
}

func testAccCheckReservationDestroyed(t *testing.T, query keaquery.ReservationQuery) func() {
	return func() {
		client := acctest.NewClient(t)

		reservation, err := client.GetReservation(context.Background(), kea.OperationTargetDatabase, query)
		if err != nil {
			t.Fatalf("failed to check reservation: %v", err)
		}

		if reservation != nil {
			t.Fatal("reservation still exists")
		}
	}
}

func testAccCheckReservationReplaced(t *testing.T, before, after keaquery.ReservationQuery) func() {
	return func() {
		testAccCheckReservationDestroyed(t, before)()
		testAccCheckReservationExists(t, after)()
	}
}

func testAccDeleteReservation(t *testing.T, query keaquery.ReservationQuery) func() {
	return func() {
		client := acctest.NewClient(t)
		if err := client.DeleteReservation(context.Background(), kea.OperationTargetDatabase, query); err != nil {
			t.Fatalf("failed to delete reservation: %v", err)
		}
	}
}

func testAccReservationConfig_basic(subnetID uint32, mac, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  hw_address = %q
  ip_address = %q
}
`, acctest.ProviderConfig(), subnetID, mac, ip)
}

func testAccReservationConfig_withClientID(subnetID uint32, clientID, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  client_id  = %q
  ip_address = %q
}
`, acctest.ProviderConfig(), subnetID, clientID, ip)
}

func testAccReservationConfig_withCircuitID(subnetID uint32, circuitID, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  circuit_id = %q
  ip_address = %q
}
`, acctest.ProviderConfig(), subnetID, circuitID, ip)
}

func testAccReservationConfig_withDUID(subnetID uint32, duid, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  duid       = %q
  ip_address = %q
}
`, acctest.ProviderConfig(), subnetID, duid, ip)
}

func testAccReservationConfig_withFlexID(subnetID uint32, flexID, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  flex_id    = %q
  ip_address = %q
}
`, acctest.ProviderConfig(), subnetID, flexID, ip)
}

func testAccReservationConfig_withHostname(subnetID uint32, mac, ip, hostname string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  hw_address = %q
  ip_address = %q
  hostname   = %q
}
`, acctest.ProviderConfig(), subnetID, mac, ip, hostname)
}

func testAccReservationConfig_withOptionData(subnetID uint32, mac, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  hw_address = %q
  ip_address = %q

  option_data {
    name       = "domain-name-servers"
    data       = "8.8.8.8,8.8.4.4"
    csv_format = true
  }
}
`, acctest.ProviderConfig(), subnetID, mac, ip)
}

func testAccReservationConfig_withOptionDataBlocks(subnetID uint32, mac, ip, blocks string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = %d
  hw_address = %q
  ip_address = %q

%s
}
`, acctest.ProviderConfig(), subnetID, mac, ip, blocks)
}

// optionDataSteps applies each config, then re-applies it and asserts an empty
// plan, guarding against option data representation churn.
func optionDataSteps(subnetID uint32, mac, ip string, blocks ...string) []resource.TestStep {
	var steps []resource.TestStep
	for _, b := range blocks {
		cfg := testAccReservationConfig_withOptionDataBlocks(subnetID, mac, ip, b)
		steps = append(steps,
			resource.TestStep{Config: cfg},
			resource.TestStep{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		)
	}
	return steps
}

func testAccReservationConfig_global(mac, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = 0
  hw_address = %q
  ip_address = %q
}
`, acctest.ProviderConfig(), mac, ip)
}

func testAccReservationConfig_withClientClassesAndOptions(subnetID uint32, mac, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id      = %d
  hw_address     = %q
  ip_address     = %q
  client_classes = ["web-servers", "production", "monitoring"]

  option_data {
    name = "domain-name-servers"
    data = "8.8.8.8,8.8.4.4"
  }

  option_data {
    name = "domain-name"
    data = "example.com"
  }

  option_data {
    name = "routers"
    data = "192.0.2.1"
  }
}
`, acctest.ProviderConfig(), subnetID, mac, ip)
}

func testAccReservationConfig_withClientClassesAndOptionsReordered(subnetID uint32, mac, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id      = %d
  hw_address     = %q
  ip_address     = %q
  client_classes = ["monitoring", "production", "web-servers"]

  option_data {
    name = "routers"
    data = "192.0.2.1"
  }

  option_data {
    name = "domain-name-servers"
    data = "8.8.8.8,8.8.4.4"
  }

  option_data {
    name = "domain-name"
    data = "example.com"
  }
}
`, acctest.ProviderConfig(), subnetID, mac, ip)
}

func testAccReservationConfig_withUserContext(subnetID uint32, mac, ip string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_reservation" "test" {
  subnet_id    = %d
  hw_address   = %q
  ip_address   = %q
  user_context = jsonencode({
    department = "engineering"
    owner      = "network-team"
    tags       = ["production", "critical"]
  })
}
`, acctest.ProviderConfig(), subnetID, mac, ip)
}
