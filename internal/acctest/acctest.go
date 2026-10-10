package acctest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/wfdewith/terraform-provider-kea/internal/provider"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

// ProtoV6ProviderFactories is used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kea": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func NewClient(t *testing.T) *keadhcp4.Client {
	t.Helper()

	client, err := provider.NewDHCP4ClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("KEA_DHCP4_ADDRESS must be set for acceptance tests")
	}

	return client
}

func KeaVersion(t *testing.T) *version.Version {
	t.Helper()

	serverVersion, err := NewClient(t).Version(context.Background())
	if err != nil {
		t.Fatalf("unable to determine Kea server version: %v", err)
	}

	return serverVersion
}

func SkipUnlessKeaVersion(t *testing.T, constraint string) {
	t.Helper()

	constraints, err := version.NewConstraint(constraint)
	if err != nil {
		t.Fatalf("invalid version constraint %q: %v", constraint, err)
	}

	serverVersion := KeaVersion(t)
	if !constraints.Check(serverVersion) {
		t.Skipf("requires Kea %s, server is %s", constraint, serverVersion)
	}
}

// PreCheckMemfile skips the test unless KEA_DHCP4_MEMFILE_ADDRESS points to
// the control socket of the server configured with
// testing/kea-dhcp4-memfile.conf, which has no hosts database.
func PreCheckMemfile(t *testing.T) {
	t.Helper()

	if os.Getenv("KEA_DHCP4_MEMFILE_ADDRESS") == "" {
		t.Skip("KEA_DHCP4_MEMFILE_ADDRESS must be set for acceptance tests without a hosts database")
	}
}

// ProviderConfigMemfile configures the provider for the server of
// PreCheckMemfile.
func ProviderConfigMemfile() string {
	return fmt.Sprintf(`
provider "kea" {
  dhcp4 = {
    address = %q
  }
}
`, os.Getenv("KEA_DHCP4_MEMFILE_ADDRESS"))
}

func ProviderConfig() string {
	return `
provider "kea" {}
`
}

func ProviderConfigTLS(t *testing.T, address string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate test PKI")
	}
	tlsDir := filepath.Join(filepath.Dir(file), "..", "..", "testing", "tls")

	return fmt.Sprintf(`
provider "kea" {
  dhcp4 = {
    address     = %q
    ca_cert     = %q
    client_cert = %q
    client_key  = %q
  }
}
`,
		address,
		readPEM(t, filepath.Join(tlsDir, "ca.pem")),
		readPEM(t, filepath.Join(tlsDir, "client.pem")),
		readPEM(t, filepath.Join(tlsDir, "client-key.pem")),
	)
}

func readPEM(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unable to read %s: %v", path, err)
	}
	return string(contents)
}

// Static reservations defined in testing/kea-dhcp4.conf
const (
	StaticSubnetMAC = "02:ac:10:00:00:01"
	StaticSubnetIP  = "10.67.0.10"
	StaticGlobalMAC = "02:ac:10:00:00:03"
	StaticGlobalIP  = "192.168.67.10"
)

// Static client class defined in testing/kea-dhcp4.conf
const StaticClientClass = "static-class"

// Static subnet defined in testing/kea-dhcp4.conf
const (
	StaticSubnetID     = 1
	StaticSubnetPrefix = "10.67.0.0/24"
)
