package dhcp4_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
)

func TestAccClientClassDataSource_basic(t *testing.T) {
	name := "acc-class-data-source"
	resourceName := "data.kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassDataSourceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", name),
					resource.TestCheckResourceAttr(resourceName, "test", "option[60].text == 'data-source'"),
					resource.TestCheckResourceAttr(resourceName, "only_in_additional_list", "false"),
					resource.TestCheckResourceAttr(resourceName, "valid_lifetime", "600"),
					resource.TestCheckResourceAttr(resourceName, "option_def.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.name", "acc-ds-string"),
					resource.TestCheckResourceAttr(resourceName, "option_def.0.type", "string"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "option_data.*", map[string]string{
						"name": "acc-ds-string",
						"data": "hello",
					}),
				),
			},
		},
	})
}

func TestAccClientClassDataSource_static(t *testing.T) {
	resourceName := "data.kea_dhcp4_client_class.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientClassDataSourceConfig_static(acctest.StaticClientClass),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", acctest.StaticClientClass),
					resource.TestCheckResourceAttr(resourceName, "test", "option[60].text == 'static'"),
					resource.TestCheckResourceAttr(resourceName, "next_server", "10.67.0.5"),
					resource.TestCheckNoResourceAttr(resourceName, "boot_file_name"),
					resource.TestCheckNoResourceAttr(resourceName, "server_hostname"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "option_data.*", map[string]string{
						"name": "domain-name",
						"data": "static.example",
					}),
				),
			},
		},
	})
}

func TestAccClientClassDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccClientClassDataSourceConfig_static("acc-class-missing"),
				ExpectError: regexp.MustCompile("Client Class Not Found"),
			},
		},
	})
}

func testAccClientClassDataSourceConfig_basic(name string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_client_class" "test" {
  name           = %q
  test           = "option[60].text == 'data-source'"
  valid_lifetime = 600

  option_def {
    name = "acc-ds-string"
    code = 233
    type = "string"
  }

  option_data {
    name = "acc-ds-string"
    data = "hello"
  }
}

data "kea_dhcp4_client_class" "test" {
  name = kea_dhcp4_client_class.test.name
}
`, acctest.ProviderConfig(), name)
}

func testAccClientClassDataSourceConfig_static(name string) string {
	return fmt.Sprintf(`
%s

data "kea_dhcp4_client_class" "test" {
  name = %q
}
`, acctest.ProviderConfig(), name)
}
