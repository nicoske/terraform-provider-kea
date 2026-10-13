package provider_test

import (
	"context"
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
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

func TestProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig(),
			},
		},
	})
}

func TestProvider_tls(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file          = %q
    tls_client_cert_file = %q
    tls_client_key_file  = %q
`, acctest.TLSFile("ca.pem"), acctest.TLSFile("client.pem"), acctest.TLSFile("client-key.pem")),
				Check: resource.TestCheckResourceAttr("data.kea_dhcp4_reservation.test", "ip_address", acctest.StaticSubnetIP),
			},
		},
	})
}

func TestProvider_tlsServerName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file          = %q
    tls_client_cert_file = %q
    tls_client_key_file  = %q
    tls_server_name      = "kea-dhcp4"
`, acctest.TLSFile("ca.pem"), acctest.TLSFile("client.pem"), acctest.TLSFile("client-key.pem")),
				Check: resource.TestCheckResourceAttr("data.kea_dhcp4_reservation.test", "ip_address", acctest.StaticSubnetIP),
			},
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file          = %q
    tls_client_cert_file = %q
    tls_client_key_file  = %q
    tls_server_name      = "other.example"
`, acctest.TLSFile("ca.pem"), acctest.TLSFile("client.pem"), acctest.TLSFile("client-key.pem")),
				ExpectError: regexp.MustCompile(`not\s+other\.example`),
			},
		},
	})
}

func TestProvider_tlsEnvironment(t *testing.T) {
	acctest.PreCheckTLS(t)

	t.Setenv("KEA_DHCP4_ADDRESS", os.Getenv("KEA_DHCP4_TLS_ADDRESS"))
	t.Setenv("KEA_DHCP4_TLS_CA_FILE", acctest.TLSFile("ca.pem"))
	t.Setenv("KEA_DHCP4_TLS_CLIENT_CERT_FILE", acctest.TLSFile("client.pem"))
	t.Setenv("KEA_DHCP4_TLS_CLIENT_KEY_FILE", acctest.TLSFile("client-key.pem"))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig() + testAccProviderConfig_staticReservation(),
				Check:  resource.TestCheckResourceAttr("data.kea_dhcp4_reservation.test", "ip_address", acctest.StaticSubnetIP),
			},
		},
	})
}

func TestProvider_tlsWithoutClientCert(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file = %q
`, acctest.TLSFile("ca.pem")),
				ExpectError: regexp.MustCompile(`certificate\s+required`),
			},
		},
	})
}

func TestProvider_tlsUnknownCA(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_client_cert_file = %q
    tls_client_key_file  = %q
`, acctest.TLSFile("client.pem"), acctest.TLSFile("client-key.pem")),
				ExpectError: regexp.MustCompile(`signed\s+by\s+unknown\s+authority`),
			},
		},
	})
}

func TestProvider_tlsCertWithoutKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckTLS(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_tls(`
    tls_ca_file          = %q
    tls_client_cert_file = %q
`, acctest.TLSFile("ca.pem"), acctest.TLSFile("client.pem")),
				ExpectError: regexp.MustCompile(`tls_client_cert_file\s+is\s+set\s+without\s+tls_client_key_file`),
			},
		},
	})
}

func testAccProviderConfig_tls(tlsAttributes string, args ...any) string {
	return fmt.Sprintf(`
provider "kea" {
  dhcp4 = {
    address = %q
%s  }
}
`, os.Getenv("KEA_DHCP4_TLS_ADDRESS"), fmt.Sprintf(tlsAttributes, args...)) + testAccProviderConfig_staticReservation()
}

func testAccProviderConfig_staticReservation() string {
	return fmt.Sprintf(`
data "kea_dhcp4_reservation" "test" {
  subnet_id  = 1
  hw_address = %q
}
`, acctest.StaticSubnetMAC)
}

func TestAccProvider_configWrite(t *testing.T) {
	configFile := testAccConfigFile(t)
	name := "config-write"
	lastWrite := testAccModTime(t, configFile)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			return testAccCheckConfigWritten(configFile, &lastWrite)
		},
		Steps: []resource.TestStep{
			{
				Config:        testAccProviderConfig_configWrite("", name, "option[host-name].text == 'a'"),
				PostApplyFunc: testAccCheckConfigWrittenFunc(t, configFile, &lastWrite),
			},
			{
				Config:        testAccProviderConfig_configWrite("", name, "option[host-name].text == 'b'"),
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
	configFile := filepath.Join(filepath.Dir(testAccConfigFile(t)), "config-write-filename.json")
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove %s: %v", configFile, err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_configWrite(filepath.Base(configFile), "config-write-filename", "option[host-name].text == 'a'"),
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
	name := "config-write-outside"
	t.Cleanup(func() {
		client := keadhcp4.NewClient(&kea.HTTPTransport{
			Endpoint: os.Getenv("KEA_DHCP4_ADDRESS"),
			Username: os.Getenv("KEA_DHCP4_HTTP_USERNAME"),
			Password: os.Getenv("KEA_DHCP4_HTTP_PASSWORD"),
		})
		_ = client.DeleteClientClass(context.Background(), name)
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderConfig_configWrite("/tmp/kea-dhcp4.conf", name, "option[host-name].text == 'a'"),
				ExpectError: regexp.MustCompile(`client class added but config-write failed(.|\s)+same\s+directory\s+as\s+the\s+config\s+file`),
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

func testAccConfigFile(t *testing.T) string {
	t.Helper()

	configFile := os.Getenv("KEA_DHCP4_CONFIG_FILE")
	if configFile == "" {
		t.Skip("KEA_DHCP4_CONFIG_FILE must be set to the host path of the Kea startup configuration file")
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
	if _, ok := config.Dhcp4["hosts-databases"]; !ok {
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

func testAccProviderConfig_configWrite(filename, name, test string) string {
	filenameAttr := ""
	if filename != "" {
		filenameAttr = fmt.Sprintf("config_write_filename = %q", filename)
	}

	return fmt.Sprintf(`
provider "kea" {
  dhcp4 = {
    %s
  }
}

resource "kea_dhcp4_client_class" "test" {
  name = %q
  test = %q
}
`, filenameAttr, name, test)
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
