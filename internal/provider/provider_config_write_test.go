package provider_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/wfdewith/terraform-provider-kea/internal/acctest"
)

func TestAccProvider_configWrite(t *testing.T) {
	configFile := testAccMemfileConfigFile(t)
	mac := "02:11:22:33:99:01"
	lastWrite := testAccModTime(t, configFile)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckMemfile(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			return testAccCheckConfigWritten(configFile, &lastWrite)
		},
		Steps: []resource.TestStep{
			{
				Config:        testAccProviderConfig_configWrite("", mac, "10.67.0.240"),
				PostApplyFunc: testAccCheckConfigWrittenFunc(t, configFile, &lastWrite),
			},
			{
				Config:        testAccProviderConfig_configWrite("", mac, "10.67.0.241"),
				PostApplyFunc: testAccCheckConfigWrittenFunc(t, configFile, &lastWrite),
			},
		},
	})
}

func TestAccProvider_configWriteWithoutHostsDatabase(t *testing.T) {
	configFile := testAccMemfileConfigFile(t)
	mac := "02:11:22:33:99:04"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckMemfile(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			found, err := testAccConfigFileHasReservation(configFile, mac)
			if err != nil {
				return err
			}
			if found {
				return fmt.Errorf("reservation %s still in %s", mac, configFile)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_withoutHostsDatabase(mac, "10.67.0.244"),
				PostApplyFunc: func() {
					found, err := testAccConfigFileHasReservation(configFile, mac)
					if err != nil {
						t.Fatal(err)
					}
					if !found {
						t.Fatalf("reservation %s not in %s", mac, configFile)
					}
				},
			},
		},
	})
}

func TestAccProvider_configWriteFilename(t *testing.T) {
	configFile := filepath.Join(filepath.Dir(testAccMemfileConfigFile(t)), "config-write-filename.json")
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove %s: %v", configFile, err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckMemfile(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_configWrite(filepath.Base(configFile), "02:11:22:33:99:02", "10.67.0.242"),
				PostApplyFunc: func() {
					// Kea creates new files with mode 0640 as root, only their presence can be checked.
					if _, err := os.Stat(configFile); err != nil {
						t.Fatalf("config-write did not create %s: %v", configFile, err)
					}
				},
			},
		},
	})
}

func TestAccProvider_configWriteOutsideConfigDir(t *testing.T) {
	mac := "02:11:22:33:99:03"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckMemfile(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderConfig_configWrite("/tmp/kea-dhcp4.conf", mac, "10.67.0.243"),
				ExpectError: regexp.MustCompile(`reservation added but config-write failed(.|\s)+same\s+directory\s+as\s+the\s+config\s+file`),
			},
		},
	})
}

func testAccMemfileConfigFile(t *testing.T) string {
	t.Helper()

	configFile := os.Getenv("KEA_DHCP4_MEMFILE_CONFIG_FILE")
	if configFile == "" {
		t.Skip("KEA_DHCP4_MEMFILE_CONFIG_FILE must be set to the host path of the configuration file of the server without a hosts database")
	}
	return configFile
}

func testAccModTime(t *testing.T, path string) time.Time {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", path, err)
	}
	return info.ModTime()
}

func testAccCheckConfigWritten(path string, lastWrite *time.Time) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}
	if !info.ModTime().After(*lastWrite) {
		return fmt.Errorf("%s not written since %s", path, lastWrite.Format(time.RFC3339Nano))
	}
	*lastWrite = info.ModTime()

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	var config struct {
		Dhcp4 map[string]json.RawMessage `json:"Dhcp4"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	if _, ok := config.Dhcp4["lease-database"]; !ok {
		return fmt.Errorf("%s does not hold the running configuration", path)
	}
	return nil
}

func testAccConfigFileHasReservation(path, mac string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var config struct {
		Dhcp4 struct {
			Subnet4 []struct {
				Reservations []struct {
					HWAddress string `json:"hw-address"`
				} `json:"reservations"`
			} `json:"subnet4"`
		} `json:"Dhcp4"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return false, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	for _, subnet := range config.Dhcp4.Subnet4 {
		for _, reservation := range subnet.Reservations {
			if strings.EqualFold(reservation.HWAddress, mac) {
				return true, nil
			}
		}
	}
	return false, nil
}

func testAccCheckConfigWrittenFunc(t *testing.T, path string, lastWrite *time.Time) func() {
	return func() {
		if err := testAccCheckConfigWritten(path, lastWrite); err != nil {
			t.Fatal(err)
		}
	}
}

func testAccProviderConfig_configWrite(filename, mac, ip string) string {
	filenameAttr := ""
	if filename != "" {
		filenameAttr = fmt.Sprintf("config_write_filename = %q", filename)
	}

	return fmt.Sprintf(`
provider "kea" {
  dhcp4 = {
    address = %q
    %s
  }
}

resource "kea_dhcp4_reservation" "test" {
  subnet_id  = 1
  hw_address = %q
  ip_address = %q
}
`, os.Getenv("KEA_DHCP4_MEMFILE_ADDRESS"), filenameAttr, mac, ip)
}

func testAccProviderConfig_withoutHostsDatabase(mac, ip string) string {
	return acctest.ProviderConfigMemfile() + fmt.Sprintf(`
resource "kea_dhcp4_reservation" "test" {
  subnet_id  = 1
  hw_address = %q
  ip_address = %q
}
`, mac, ip)
}
