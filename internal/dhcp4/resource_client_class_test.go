package dhcp4_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
)

func TestAccClientClass_basic(t *testing.T) {
	name := "acc-class-basic"
	test := "option[60].text == 'basic'"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_basic(name, test),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", name),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "test", test),
					resource.TestCheckResourceAttr(resourceName, "only_in_additional_list", "false"),
					resource.TestCheckResourceAttr(resourceName, "next_server", "0.0.0.0"),
					resource.TestCheckResourceAttr(resourceName, "server_hostname", ""),
					resource.TestCheckResourceAttr(resourceName, "boot_file_name", ""),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
			{
				Config: testAccClientClassConfig_basic(name, test),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccClientClass_full(t *testing.T) {
	name := "acc-class-full"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_full(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "only_in_additional_list", "true"),
					resource.TestCheckResourceAttr(resourceName, "next_server", "10.67.0.5"),
					resource.TestCheckResourceAttr(resourceName, "server_hostname", "tftp.example"),
					resource.TestCheckResourceAttr(resourceName, "boot_file_name", "pxelinux.0"),
					resource.TestCheckResourceAttr(resourceName, "valid_lifetime", "3600"),
					resource.TestCheckResourceAttr(resourceName, "min_valid_lifetime", "1800"),
					resource.TestCheckResourceAttr(resourceName, "max_valid_lifetime", "7200"),
					resource.TestCheckResourceAttrSet(resourceName, "user_context"),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
			{
				Config: testAccClientClassConfig_full(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccClientClass_nextServerZero(t *testing.T) {
	name := "acc-class-next-server-zero"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_withNextServer(name, "0.0.0.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "next_server", "0.0.0.0"),
				),
			},
		},
	})
}

func TestAccClientClass_explicitEmptyStrings(t *testing.T) {
	name := "acc-class-empty-strings"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_emptyStrings(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "server_hostname", ""),
					resource.TestCheckResourceAttr(resourceName, "boot_file_name", ""),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.record_types", ""),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.encapsulate", ""),
				),
			},
		},
	})
}

func TestAccClientClass_withOptionData(t *testing.T) {
	name := "acc-class-option-data"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_withOptionData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "option_data.*", map[string]string{
						"name":       "domain-name-servers",
						"data":       "10.67.0.53, 10.67.0.54",
						"csv_format": "true",
						"space":      "dhcp4",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "option_data.*", map[string]string{
						"name":       "domain-name",
						"data":       "'class.example'",
						"csv_format": "false",
					}),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
			{
				Config: testAccClientClassConfig_withOptionData(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccClientClass_withOptionDef(t *testing.T) {
	name := "acc-class-option-def"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_withOptionDef(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "option_def.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.name", "acc-record"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.code", "231"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.type", "record"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.record_types", "uint8, string"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.array", "false"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.space", "dhcp4"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.encapsulate", ""),
					resource.TestCheckResourceAttr(resourceName, "option_def.1.record_types", ""),
					resource.TestCheckResourceAttr(resourceName, "option_def.1.name", "acc-array"),
					resource.TestCheckResourceAttr(resourceName, "option_def.1.array", "true"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "2"),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
			{
				Config: testAccClientClassConfig_withOptionDef(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccClientClass_update(t *testing.T) {
	name := "acc-class-update"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_basic(name, "option[60].text == 'before'"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "test", "option[60].text == 'before'"),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
			{
				Config: testAccClientClassConfig_withOptionDef(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "option_def.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "2"),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
			{
				Config: testAccClientClassConfig_basic(name, "option[60].text == 'after'"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "test", "option[60].text == 'after'"),
					resource.TestCheckNoResourceAttr(resourceName, "option_def.#"),
					resource.TestCheckNoResourceAttr(resourceName, "option_data.#"),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
		},
	})
}

func TestAccClientClass_rename(t *testing.T) {
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccClientClassConfig_basic("acc-class-rename-1", "option[60].text == 'rename'"),
				PostApplyFunc: testAccCheckClientClassExists(t, "acc-class-rename-1"),
			},
			{
				Config: testAccClientClassConfig_basic("acc-class-rename-2", "option[60].text == 'rename'"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				PostApplyFunc: func() {
					testAccCheckClientClassDestroyed(t, "acc-class-rename-1")()
					testAccCheckClientClassExists(t, "acc-class-rename-2")()
				},
			},
		},
	})
}

func TestAccClientClass_templateTest(t *testing.T) {
	name := "acc-class-template"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_basic(name, "option[60].text == 'template'"),
			},
			{
				Config: testAccClientClassConfig_templateTest(name, "substring(option[61].hex, 0, 3)"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "template_test", "substring(option[61].hex, 0, 3)"),
					resource.TestCheckNoResourceAttr(resourceName, "test"),
				),
			},
			{
				Config: testAccClientClassConfig_templateTest(name, "substring(option[61].hex, 0, 4)"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: testAccClientClassConfig_basic(name, "option[60].text == 'template'"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "test", "option[60].text == 'template'"),
					resource.TestCheckNoResourceAttr(resourceName, "template_test"),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
		},
	})
}

func TestAccClientClass_destroy(t *testing.T) {
	name := "acc-class-destroy"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccClientClassConfig_basic(name, "option[60].text == 'destroy'"),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
			{
				Config:        acctest.ProviderConfig(),
				PostApplyFunc: testAccCheckClientClassDestroyed(t, name),
			},
		},
	})
}

func TestAccClientClass_disappears(t *testing.T) {
	name := "acc-class-disappears"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccClientClassConfig_basic(name, "option[60].text == 'disappears'"),
				PostApplyFunc: testAccDeleteClientClass(t, name),
			},
			{
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccClientClass_reorderSetsNoUpdate(t *testing.T) {
	name := "acc-class-reorder"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_withOptions(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "3"),
				),
				PostApplyFunc: testAccCheckClientClassExists(t, name),
			},
			{
				Config: testAccClientClassConfig_withOptionsReordered(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccClientClass_import(t *testing.T) {
	name := "acc-class-import"
	resourceName := "kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassConfig_withOptionDef(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckClientClassExists(t *testing.T, name string) func() {
	return func() {
		client := acctest.NewClient(t)

		class, err := client.GetClientClass(context.Background(), name)
		if err != nil {
			t.Fatalf("failed to get client class: %v", err)
		}

		if class == nil {
			t.Fatal("client class does not exist in Kea")
		}
	}
}

func testAccCheckClientClassDestroyed(t *testing.T, name string) func() {
	return func() {
		client := acctest.NewClient(t)

		class, err := client.GetClientClass(context.Background(), name)
		if err != nil {
			t.Fatalf("failed to check client class: %v", err)
		}

		if class != nil {
			t.Fatal("client class still exists")
		}
	}
}

func testAccDeleteClientClass(t *testing.T, name string) func() {
	return func() {
		client := acctest.NewClient(t)
		if err := client.DeleteClientClass(context.Background(), name); err != nil {
			t.Fatalf("failed to delete client class: %v", err)
		}
	}
}

func testAccClientClassConfig_basic(name, test string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name = %q
  test = %q
}
`, acctest.ProviderConfig(), name, test)
}

func testAccClientClassConfig_withNextServer(name, nextServer string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name        = %q
  next_server = %q
}
`, acctest.ProviderConfig(), name, nextServer)
}

func testAccClientClassConfig_emptyStrings(name string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name            = %q
  server_hostname = ""
  boot_file_name  = ""

  option_def {
    name         = "acc-empty-strings"
    code         = 236
    type         = "uint8"
    record_types = ""
    encapsulate  = ""
  }
}
`, acctest.ProviderConfig(), name)
}

func testAccClientClassConfig_templateTest(name, templateTest string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name          = %q
  template_test = %q
}
`, acctest.ProviderConfig(), name, templateTest)
}

func testAccClientClassConfig_full(name string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name                    = %q
  test                    = "option[93].hex == 0x0007"
  only_in_additional_list = true
  next_server             = "10.67.0.5"
  server_hostname         = "tftp.example"
  boot_file_name          = "pxelinux.0"
  valid_lifetime          = 3600
  min_valid_lifetime      = 1800
  max_valid_lifetime      = 7200
  user_context = jsonencode({
    owner = "network-team"
  })
}
`, acctest.ProviderConfig(), name)
}

func testAccClientClassConfig_withOptionData(name string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name = %q
  test = "option[60].text == 'options'"

  option_data {
    name = "domain-name-servers"
    data = "10.67.0.53, 10.67.0.54"
  }

  option_data {
    name       = "domain-name"
    data       = "'class.example'"
    csv_format = false
  }
}
`, acctest.ProviderConfig(), name)
}

func testAccClientClassConfig_withOptionDef(name string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name = %q
  test = "option[60].text == 'definitions'"

  option_def {
    name         = "acc-record"
    code         = 231
    type         = "record"
    record_types = "uint8, string"
  }

  option_def {
    name  = "acc-array"
    code  = 232
    type  = "uint16"
    array = true
  }

  option_data {
    name = "acc-record"
    data = "1, text"
  }

  option_data {
    code = 232
    data = "1, 2, 3"
  }
}
`, acctest.ProviderConfig(), name)
}

func testAccClientClassConfig_withOptions(name string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name = %q
  test = "option[60].text == 'reorder'"

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
`, acctest.ProviderConfig(), name)
}

func testAccClientClassConfig_withOptionsReordered(name string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name = %q
  test = "option[60].text == 'reorder'"

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
`, acctest.ProviderConfig(), name)
}
