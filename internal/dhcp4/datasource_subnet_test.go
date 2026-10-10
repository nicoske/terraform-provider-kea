package dhcp4_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
)

func TestAccSubnetDataSource_byID(t *testing.T) {
	resourceName := "data.kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetDataSourceConfig_byID(111, "10.67.19.0/25"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "111"),
					resource.TestCheckResourceAttr(resourceName, "subnet", "10.67.19.0/25"),
					resource.TestCheckResourceAttr(resourceName, "valid_lifetime", "3600"),
					resource.TestCheckResourceAttr(resourceName, "relay.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "pools.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "pools.0.pool", "10.67.19.10-10.67.19.20"),
					resource.TestCheckResourceAttr(resourceName, "pools.0.option_data.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "option_data.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "option_data.*", map[string]string{
						"name": "routers",
						"data": "10.67.19.1",
					}),
				),
			},
		},
	})
}

func TestAccSubnetDataSource_bySubnet(t *testing.T) {
	resourceName := "data.kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetDataSourceConfig_bySubnet(112, "10.67.19.128/25"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "112"),
					resource.TestCheckResourceAttr(resourceName, "subnet", "10.67.19.128/25"),
					resource.TestCheckResourceAttr(resourceName, "client_classes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "boot_file_name", "pxelinux.0"),
				),
			},
		},
	})
}

func TestAccSubnetDataSource_static(t *testing.T) {
	resourceName := "data.kea_dhcp4_subnet.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSubnetDataSourceConfig_static(acctest.StaticSubnetPrefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", fmt.Sprint(acctest.StaticSubnetID)),
					resource.TestCheckResourceAttr(resourceName, "interface", "dummy0"),
					resource.TestCheckResourceAttr(resourceName, "pools.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "pools.0.pool", "10.67.0.100-10.67.0.200"),
					resource.TestCheckNoResourceAttr(resourceName, "relay"),
					resource.TestCheckNoResourceAttr(resourceName, "next_server"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "option_data.*", map[string]string{
						"name": "routers",
						"data": "10.67.0.1",
					}),
				),
			},
		},
	})
}

func TestAccSubnetDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig() + `
data "kea_dhcp4_subnet" "test" {
  id = 4000000000
}
`,
				ExpectError: regexp.MustCompile("Subnet Not Found"),
			},
		},
	})
}

func TestAccSubnetDataSource_idAndSubnet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig() + fmt.Sprintf(`
data "kea_dhcp4_subnet" "test" {
  id     = %d
  subnet = %q
}
`, acctest.StaticSubnetID, acctest.StaticSubnetPrefix),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}

func testAccSubnetDataSourceConfig_byID(id uint32, subnet string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_subnet" "test" {
  id             = %d
  subnet         = %q
  valid_lifetime = 3600
  relay          = ["10.1.19.1"]

  option_data {
    name = "routers"
    data = "10.67.19.1"
  }

  pools {
    pool = "10.67.19.10 - 10.67.19.20"

    option_data {
      name = "domain-name"
      data = "pool.example"
    }
  }
}

data "kea_dhcp4_subnet" "test" {
  id = kea_dhcp4_subnet.test.id
}
`, acctest.ProviderConfig(), id, subnet)
}

func testAccSubnetDataSourceConfig_bySubnet(id uint32, subnet string) string {
	return fmt.Sprintf(`
%s

resource "kea_dhcp4_subnet" "test" {
  id             = %d
  subnet         = %q
  client_classes = ["acc-data-source"]
  boot_file_name = "pxelinux.0"
}

data "kea_dhcp4_subnet" "test" {
  subnet = kea_dhcp4_subnet.test.subnet
}
`, acctest.ProviderConfig(), id, subnet)
}

func testAccSubnetDataSourceConfig_static(subnet string) string {
	return fmt.Sprintf(`
%s

data "kea_dhcp4_subnet" "test" {
  subnet = %q
}
`, acctest.ProviderConfig(), subnet)
}
