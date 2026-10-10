package dhcp4_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
	"github.com/wfdewith/terraform-provider-kea/kea/keaquery"
)

func TestAccSubnet_basic(t *testing.T) {
	resourceName := "kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfig_basic(100, "10.67.10.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "100"),
					resource.TestCheckResourceAttr(resourceName, "subnet", "10.67.10.0/24"),
					resource.TestCheckResourceAttrSet(resourceName, "valid_lifetime"),
					resource.TestCheckNoResourceAttr(resourceName, "pools.#"),
					resource.TestCheckNoResourceAttr(resourceName, "relay"),
					resource.TestCheckNoResourceAttr(resourceName, "shared_network_name"),
				),
				PostApplyFunc: testAccCheckSubnetExists(t, 100),
			},
		},
	})
}

func TestAccSubnet_pools(t *testing.T) {
	resourceName := "kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfig_withBlocks(101, "10.67.11.0/24", `
  pools {
    pool                        = "10.67.11.10 - 10.67.11.20"
    client_classes              = ["acc-pool-a"]
    evaluate_additional_classes = ["acc-pool-b"]
  }

  pools {
    pool = "10.67.11.64/26"
  }

  pools {
    pool = "10.67.11.128 - 10.67.11.143"
  }

  pools {
    pool = "10.67.11.200 - 10.67.11.200"
  }
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pools.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "pools.0.pool", "10.67.11.10 - 10.67.11.20"),
					resource.TestCheckResourceAttr(resourceName, "pools.0.client_classes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "pools.0.evaluate_additional_classes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "pools.1.pool", "10.67.11.64/26"),
					resource.TestCheckResourceAttr(resourceName, "pools.2.pool", "10.67.11.128 - 10.67.11.143"),
					resource.TestCheckResourceAttr(resourceName, "pools.3.pool", "10.67.11.200 - 10.67.11.200"),
				),
				PostApplyFunc: testAccCheckSubnetExists(t, 101),
			},
		},
	})
}

func TestAccSubnet_withOptionData(t *testing.T) {
	resourceName := "kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfig_withBlocks(102, "10.67.12.0/24", `
  option_data {
    name = "routers"
    data = "10.67.12.1"
  }

  option_data {
    name       = "domain-name-servers"
    data       = "0a:43:0c:02"
    csv_format = false
  }

  pools {
    pool = "10.67.12.100 - 10.67.12.200"

    option_data {
      name = "domain-name"
      data = "pool.example"
    }

    option_data {
      code       = 224
      data       = "'hello'"
      csv_format = false
    }
  }
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "option_data.*", map[string]string{
						"name": "domain-name-servers",
						"data": "0a:43:0c:02",
					}),
					resource.TestCheckResourceAttr(resourceName, "pools.0.option_data.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "pools.0.option_data.*", map[string]string{
						"code": "224",
						"data": "'hello'",
					}),
				),
				PostApplyFunc: testAccCheckSubnetExists(t, 102),
			},
		},
	})
}

func TestAccSubnet_full(t *testing.T) {
	resourceName := "kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfig_full(103, "10.67.13.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "interface", "dummy0"),
					resource.TestCheckResourceAttr(resourceName, "relay.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "client_classes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "evaluate_additional_classes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "next_server", "10.67.13.5"),
					resource.TestCheckResourceAttr(resourceName, "server_hostname", "tftp.example"),
					resource.TestCheckResourceAttr(resourceName, "boot_file_name", "pxelinux.0"),
					resource.TestCheckResourceAttr(resourceName, "valid_lifetime", "3600"),
					resource.TestCheckResourceAttr(resourceName, "min_valid_lifetime", "1800"),
					resource.TestCheckResourceAttr(resourceName, "max_valid_lifetime", "7200"),
					resource.TestCheckResourceAttr(resourceName, "renew_timer", "900"),
					resource.TestCheckResourceAttr(resourceName, "rebind_timer", "1800"),
					resource.TestCheckResourceAttr(resourceName, "reservations_global", "false"),
					resource.TestCheckResourceAttr(resourceName, "reservations_in_subnet", "true"),
					resource.TestCheckResourceAttr(resourceName, "reservations_out_of_pool", "true"),
					resource.TestCheckResourceAttr(resourceName, "ddns_send_updates", "true"),
					resource.TestCheckResourceAttr(resourceName, "ddns_override_no_update", "false"),
					resource.TestCheckResourceAttr(resourceName, "ddns_override_client_update", "true"),
					resource.TestCheckResourceAttr(resourceName, "ddns_replace_client_name", "when-not-present"),
					resource.TestCheckResourceAttr(resourceName, "ddns_generated_prefix", "host"),
					resource.TestCheckResourceAttr(resourceName, "ddns_qualifying_suffix", "example.org."),
					resource.TestCheckResourceAttr(resourceName, "ddns_update_on_renew", "true"),
					resource.TestCheckResourceAttr(resourceName, "ddns_ttl", "300"),
					resource.TestCheckResourceAttrSet(resourceName, "user_context"),
				),
				PostApplyFunc: testAccCheckSubnetExists(t, 103),
			},
		},
	})
}

func TestAccSubnet_update(t *testing.T) {
	resourceName := "kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfig_withBlocks(104, "10.67.14.0/24", `
  valid_lifetime = 3600
  renew_timer    = 900

  option_data {
    name = "routers"
    data = "10.67.14.1"
  }

  pools {
    pool = "10.67.14.10 - 10.67.14.20"
  }
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "valid_lifetime", "3600"),
					resource.TestCheckResourceAttr(resourceName, "min_valid_lifetime", "3600"),
					resource.TestCheckResourceAttr(resourceName, "renew_timer", "900"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "pools.#", "1"),
				),
				PostApplyFunc: testAccCheckSubnetExists(t, 104),
			},
			{
				Config: testAccSubnetConfig_withBlocks(104, "10.67.14.0/24", `
  valid_lifetime = 1800
  renew_timer    = 600
  rebind_timer   = 1200

  option_data {
    name = "routers"
    data = "10.67.14.2"
  }

  option_data {
    name = "domain-name-servers"
    data = "10.67.14.53"
  }

  pools {
    pool = "10.67.14.100 - 10.67.14.200"
  }

  pools {
    pool = "10.67.14.10 - 10.67.14.20"

    option_data {
      name = "domain-name"
      data = "pool.example"
    }
  }
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(resourceName, tfjsonpath.New("min_valid_lifetime")),
						plancheck.ExpectUnknownValue(resourceName, tfjsonpath.New("max_valid_lifetime")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "valid_lifetime", "1800"),
					resource.TestCheckResourceAttr(resourceName, "min_valid_lifetime", "1800"),
					resource.TestCheckResourceAttr(resourceName, "max_valid_lifetime", "1800"),
					resource.TestCheckResourceAttr(resourceName, "renew_timer", "600"),
					resource.TestCheckResourceAttr(resourceName, "rebind_timer", "1200"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "option_data.*", map[string]string{
						"name": "routers",
						"data": "10.67.14.2",
					}),
					resource.TestCheckResourceAttr(resourceName, "pools.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "pools.0.pool", "10.67.14.100 - 10.67.14.200"),
					resource.TestCheckResourceAttr(resourceName, "pools.1.option_data.#", "1"),
				),
				PostApplyFunc: testAccCheckSubnetExists(t, 104),
			},
			{
				Config: testAccSubnetConfig_withBlocks(104, "10.67.14.0/24", `
  boot_file_name = "pxelinux.0"

  pools {
    pool = "10.67.14.100 - 10.67.14.200"
  }
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue(resourceName, tfjsonpath.New("valid_lifetime")),
						plancheck.ExpectUnknownValue(resourceName, tfjsonpath.New("renew_timer")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "valid_lifetime", "7200"),
					resource.TestCheckNoResourceAttr(resourceName, "renew_timer"),
					resource.TestCheckNoResourceAttr(resourceName, "rebind_timer"),
					resource.TestCheckResourceAttr(resourceName, "boot_file_name", "pxelinux.0"),
					resource.TestCheckNoResourceAttr(resourceName, "option_data.#"),
					resource.TestCheckResourceAttr(resourceName, "pools.#", "1"),
				),
				PostApplyFunc: testAccCheckSubnetExists(t, 104),
			},
			{
				Config: testAccSubnetConfig_withBlocks(104, "10.67.14.0/24", `
  boot_file_name = "pxelinux.1"

  pools {
    pool = "10.67.14.100 - 10.67.14.200"
  }
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						expectKnownTimers(resourceName),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New("valid_lifetime"), knownvalue.Int64Exact(7200)),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New("renew_timer"), knownvalue.Null()),
					},
				},
			},
			{
				Config: testAccSubnetConfig_withBlocks(104, "10.67.14.0/24", `
  boot_file_name = "pxelinux.1"
  valid_lifetime = 7200

  pools {
    pool = "10.67.14.100 - 10.67.14.200"
  }
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: testAccSubnetConfig_withBlocks(104, "10.67.14.0/24", `
  boot_file_name = "pxelinux.2"
  valid_lifetime = 7200

  pools {
    pool = "10.67.14.100 - 10.67.14.200"
  }
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						expectKnownTimers(resourceName),
					},
				},
			},
		},
	})
}

// expectKnownTimers expects every subnet timer to be known in the plan.
func expectKnownTimers(resourceName string) plancheck.PlanCheck {
	return knownTimersCheck{resourceName}
}

type knownTimersCheck struct {
	resourceName string
}

func (c knownTimersCheck) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, rc := range req.Plan.ResourceChanges {
		if rc.Address != c.resourceName {
			continue
		}
		unknown, _ := rc.Change.AfterUnknown.(map[string]any)
		for _, name := range []string{"valid_lifetime", "min_valid_lifetime", "max_valid_lifetime", "renew_timer", "rebind_timer"} {
			if unknown[name] == true {
				resp.Error = fmt.Errorf("%s.%s is unknown", c.resourceName, name)
				return
			}
		}
		return
	}
	resp.Error = fmt.Errorf("%s not found in plan", c.resourceName)
}

func TestAccSubnet_destroy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccSubnetConfig_basic(105, "10.67.15.0/25"),
				PostApplyFunc: testAccCheckSubnetExists(t, 105),
			},
			{
				Config:        acctest.ProviderConfig(),
				PostApplyFunc: testAccCheckSubnetDestroyed(t, 105),
			},
		},
	})
}

func TestAccSubnet_disappears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccSubnetConfig_basic(106, "10.67.15.128/25"),
				PostApplyFunc: testAccDeleteSubnet(t, 106),
			},
			{
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccSubnet_reorderSetsNoUpdate(t *testing.T) {
	resourceName := "kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfig_withBlocks(107, "10.67.16.0/24", `
  client_classes = ["acc-a", "acc-b", "acc-c"]
  relay          = ["10.1.16.1", "10.1.16.2", "10.1.16.3"]

  option_data {
    name = "routers"
    data = "10.67.16.1"
  }

  option_data {
    name = "domain-name"
    data = "example.com"
  }

  option_data {
    name       = "domain-name-servers"
    data       = "0a000001"
    csv_format = false
  }

  pools {
    pool           = "10.67.16.10 - 10.67.16.20"
    client_classes = ["acc-x", "acc-y"]
  }
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "client_classes.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "relay.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "3"),
				),
				PostApplyFunc: testAccCheckSubnetExists(t, 107),
			},
			{
				Config: testAccSubnetConfig_withBlocks(107, "10.67.16.0/24", `
  client_classes = ["acc-c", "acc-a", "acc-b"]
  relay          = ["10.1.16.3", "10.1.16.1", "10.1.16.2"]

  option_data {
    name       = "domain-name-servers"
    data       = "0a000001"
    csv_format = false
  }

  option_data {
    name = "routers"
    data = "10.67.16.1"
  }

  option_data {
    name = "domain-name"
    data = "example.com"
  }

  pools {
    pool           = "10.67.16.10 - 10.67.16.20"
    client_classes = ["acc-y", "acc-x"]
  }
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccSubnet_import(t *testing.T) {
	resourceName := "kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetConfig_withBlocks(108, "10.67.17.0/24", `
  valid_lifetime = 3600
  relay          = ["10.1.17.1"]
  user_context   = jsonencode({ site = "acc" })

  option_data {
    name = "routers"
    data = "10.67.17.1"
  }

  pools {
    pool           = "10.67.17.64/27"
    client_classes = ["acc-import"]
  }

  pools {
    pool = "10.67.17.100-10.67.17.200"

    option_data {
      name = "domain-name"
      data = "pool.example"
    }
  }
`),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "108",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSubnet_importInvalidID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccSubnetConfig_basic(113, "10.67.18.0/28"),
				ResourceName:  "kea_dhcp4_subnet.test",
				ImportState:   true,
				ImportStateId: "not-a-number",
				ExpectError:   regexp.MustCompile("Invalid Import ID"),
			},
		},
	})
}

func TestAccSubnet_changeIdentity(t *testing.T) {
	testCases := map[string]struct {
		idBefore, idAfter         uint32
		subnetBefore, subnetAfter string
	}{
		"subnet": {
			idBefore:     109,
			idAfter:      109,
			subnetBefore: "10.67.18.64/27",
			subnetAfter:  "10.67.18.96/27",
		},
		"id": {
			idBefore:     110,
			idAfter:      114,
			subnetBefore: "10.67.18.128/27",
			subnetAfter:  "10.67.18.128/27",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { acctest.PreCheck(t) },
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:        testAccSubnetConfig_basic(tc.idBefore, tc.subnetBefore),
						PostApplyFunc: testAccCheckSubnetExists(t, tc.idBefore),
					},
					{
						Config: testAccSubnetConfig_basic(tc.idAfter, tc.subnetAfter),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("kea_dhcp4_subnet.test", plancheck.ResourceActionDestroyBeforeCreate),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("kea_dhcp4_subnet.test", "id", fmt.Sprint(tc.idAfter)),
							resource.TestCheckResourceAttr("kea_dhcp4_subnet.test", "subnet", tc.subnetAfter),
						),
						PostApplyFunc: testAccCheckSubnetExists(t, tc.idAfter),
					},
				},
			})
		})
	}
}

func TestAccSubnet_invalidPoolAtPlan(t *testing.T) {
	testCases := map[string]string{
		"host_bits": "10.67.18.5/28",
		"reversed":  "10.67.18.20 - 10.67.18.10",
		"single":    "10.67.18.10",
		"ipv6":      "2001:db8::1 - 2001:db8::2",
	}

	for name, pool := range testCases {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { acctest.PreCheck(t) },
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: testAccSubnetConfig_withBlocks(115, "10.67.18.0/28", fmt.Sprintf(`
  pools {
    pool = %q
  }
`, pool)),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile("Invalid Pool"),
				}},
			})
		})
	}
}

func TestAccSubnet_poolOutsideSubnetFails(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testAccSubnetConfig_withBlocks(116, "10.67.18.16/28", `
  pools {
    pool = "10.67.18.200 - 10.67.18.210"
  }
`),
			ExpectError: regexp.MustCompile("Error Adding Subnet"),
		}},
	})
}

func testAccCheckSubnetExists(t *testing.T, id uint32) func() {
	return func() {
		client := testAccKeaClient()

		subnet, err := client.GetSubnet(context.Background(), keaquery.SubnetByID(id))
		if err != nil {
			t.Fatalf("failed to get subnet: %v", err)
		}

		if subnet == nil {
			t.Fatal("subnet does not exist in Kea")
		}
	}
}

func testAccCheckSubnetDestroyed(t *testing.T, id uint32) func() {
	return func() {
		client := testAccKeaClient()

		subnet, err := client.GetSubnet(context.Background(), keaquery.SubnetByID(id))
		if err != nil {
			t.Fatalf("failed to check subnet: %v", err)
		}

		if subnet != nil {
			t.Fatal("subnet still exists")
		}
	}
}

func testAccDeleteSubnet(t *testing.T, id uint32) func() {
	return func() {
		client := testAccKeaClient()
		if err := client.DeleteSubnet(context.Background(), id); err != nil {
			t.Fatalf("failed to delete subnet: %v", err)
		}
	}
}

func testAccSubnetConfig_basic(id uint32, subnet string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_subnet" "test" {
  id     = %d
  subnet = %q
}
`, acctest.ProviderConfig(), id, subnet)
}

func testAccSubnetConfig_withBlocks(id uint32, subnet, blocks string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_subnet" "test" {
  id     = %d
  subnet = %q
%s
}
`, acctest.ProviderConfig(), id, subnet, blocks)
}

func testAccSubnetConfig_full(id uint32, subnet string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_subnet" "test" {
  id     = %d
  subnet = %q

  interface                   = "dummy0"
  relay                       = ["10.1.13.1", "10.1.13.2"]
  client_classes              = ["acc-full"]
  evaluate_additional_classes = ["acc-full-additional"]

  next_server     = "10.67.13.5"
  server_hostname = "tftp.example"
  boot_file_name  = "pxelinux.0"

  valid_lifetime     = 3600
  min_valid_lifetime = 1800
  max_valid_lifetime = 7200
  renew_timer        = 900
  rebind_timer       = 1800

  reservations_global      = false
  reservations_in_subnet   = true
  reservations_out_of_pool = true

  ddns_send_updates           = true
  ddns_override_no_update     = false
  ddns_override_client_update = true
  ddns_replace_client_name    = "when-not-present"
  ddns_generated_prefix       = "host"
  ddns_qualifying_suffix      = "example.org."
  ddns_update_on_renew        = true
  ddns_ttl                    = 300

  user_context = jsonencode({
    owner = "network-team"
    tags  = ["acc", "full"]
  })

  pools {
    pool = "10.67.13.100 - 10.67.13.200"
  }
}
`, acctest.ProviderConfig(), id, subnet)
}
