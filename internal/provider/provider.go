package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/wfdewith/terraform-provider-kea/internal/clients"
	"github.com/wfdewith/terraform-provider-kea/internal/dhcp4"
	"github.com/wfdewith/terraform-provider-kea/kea"
	"github.com/wfdewith/terraform-provider-kea/kea/keadhcp4"
)

type KeaProvider struct {
	version string
}

type KeaProviderModel struct {
	DHCP4 types.Object `tfsdk:"dhcp4"`
}

type KeaProviderClientModel struct {
	Address       types.String `tfsdk:"address"`
	HTTPUsername  types.String `tfsdk:"http_username"`
	HTTPPassword  types.String `tfsdk:"http_password"`
	CACert        types.String `tfsdk:"ca_cert"`
	ClientCert    types.String `tfsdk:"client_cert"`
	ClientKey     types.String `tfsdk:"client_key"`
	SkipTLSVerify types.Bool   `tfsdk:"skip_tls_verify"`

	ConfigWriteFilename types.String `tfsdk:"config_write_filename"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KeaProvider{version}
	}
}

func (p *KeaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kea"
	resp.Version = p.version
}

func (p *KeaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Kea provider enables management of Kea DHCP server resources via the control channel API. " +
			"It connects to Kea servers using UNIX domain sockets or HTTP/HTTPS endpoints.",
		Attributes: map[string]schema.Attribute{
			"dhcp4": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Connection settings for the Kea DHCPv4 server. Every change to the running configuration is followed by `config-write`, so that Kea keeps it across restarts.",
				Attributes: map[string]schema.Attribute{
					"address": schema.StringAttribute{
						Optional: true,
						Description: "Control channel address: `unix:///path/to/socket` or `http(s)://host:port`. " +
							"Falls back to `KEA_DHCP4_ADDRESS` environment variable.",
						Validators: []validator.String{
							IsValidKeaURL(),
						},
					},
					"http_username": schema.StringAttribute{
						Optional:    true,
						Description: "Username for HTTP basic authentication. Falls back to `KEA_DHCP4_HTTP_USERNAME` environment variable.",
					},
					"http_password": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Password for HTTP basic authentication. Falls back to `KEA_DHCP4_HTTP_PASSWORD` environment variable.",
					},
					"ca_cert": schema.StringAttribute{
						Optional: true,
						Description: "PEM-encoded CA certificate bundle used to verify the Kea server's TLS certificate. " +
							"Only applies to `https://` addresses. Falls back to `KEA_DHCP4_CA_CERT` environment variable.",
					},
					"client_cert": schema.StringAttribute{
						Optional: true,
						Description: "PEM-encoded client certificate presented to the Kea server for mutual TLS. " +
							"Requires `client_key` and only applies to `https://` addresses. " +
							"Falls back to `KEA_DHCP4_CLIENT_CERT` environment variable.",
					},
					"client_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "PEM-encoded private key for `client_cert`. Falls back to `KEA_DHCP4_CLIENT_KEY` environment variable.",
					},
					"skip_tls_verify": schema.BoolAttribute{
						Optional: true,
						Description: "Skip verification of the Kea server's TLS certificate. " +
							"Only applies to `https://` addresses. " +
							"Falls back to `KEA_DHCP4_SKIP_TLS_VERIFY` environment variable.",
					},
					"config_write_filename": schema.StringAttribute{
						Optional: true,
						Description: "File that `config-write` writes to, instead of the configuration file Kea was started with. " +
							"Kea only accepts files in the directory of its startup configuration file. " +
							"Falls back to `KEA_DHCP4_CONFIG_WRITE_FILENAME` environment variable.",
					},
				},
			},
		},
	}
}

func (p *KeaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data KeaProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var dhcp4Config *KeaProviderClientModel
	if !data.DHCP4.IsNull() && !data.DHCP4.IsUnknown() {
		dhcp4Config = &KeaProviderClientModel{}
		resp.Diagnostics.Append(data.DHCP4.As(ctx, dhcp4Config, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	dhcp4Client, err := configureDHCP4Client(dhcp4Config)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Kea DHCP4 client: %s", err))
		return
	}

	keaClients := clients.KeaClients{
		DHCP4: dhcp4Client,
	}

	resp.DataSourceData = keaClients
	resp.ResourceData = keaClients
}

func (p *KeaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return &dhcp4.ClientClassResource{} },
		func() resource.Resource { return &dhcp4.ReservationResource{} },
	}
}

func (p *KeaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &dhcp4.ReservationDataSource{} },
	}
}

func configureDHCP4Client(data *KeaProviderClientModel) (*keadhcp4.Client, error) {
	if data == nil {
		data = &KeaProviderClientModel{}
	}

	address := envString(data.Address, "KEA_DHCP4_ADDRESS")

	// No address configured - skip DHCP4 client setup
	if address == "" {
		return nil, nil
	}

	skipTLSVerify, err := envBool(data.SkipTLSVerify, "KEA_DHCP4_SKIP_TLS_VERIFY")
	if err != nil {
		return nil, err
	}

	transport, err := newKeaTransport(
		address,
		envString(data.HTTPUsername, "KEA_DHCP4_HTTP_USERNAME"),
		envString(data.HTTPPassword, "KEA_DHCP4_HTTP_PASSWORD"),
		tlsOptions{
			CACert:        envString(data.CACert, "KEA_DHCP4_CA_CERT"),
			ClientCert:    envString(data.ClientCert, "KEA_DHCP4_CLIENT_CERT"),
			ClientKey:     envString(data.ClientKey, "KEA_DHCP4_CLIENT_KEY"),
			SkipTLSVerify: skipTLSVerify,
		},
	)
	if err != nil {
		return nil, err
	}

	var opts []keadhcp4.ClientOption
	if filename := envString(data.ConfigWriteFilename, "KEA_DHCP4_CONFIG_WRITE_FILENAME"); filename != "" {
		opts = append(opts, keadhcp4.WithConfigWriteFilename(filename))
	}

	return keadhcp4.NewClient(transport, opts...), nil
}

func NewDHCP4ClientFromEnv() (*keadhcp4.Client, error) {
	return configureDHCP4Client(nil)
}

func envString(value types.String, name string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}
	return os.Getenv(name)
}

func envBool(value types.Bool, name string) (bool, error) {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueBool(), nil
	}

	s, ok := os.LookupEnv(name)
	if !ok || s == "" {
		return false, nil
	}

	v, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("invalid value for %s: %w", name, err)
	}
	return v, nil
}

type tlsOptions struct {
	CACert        string
	ClientCert    string
	ClientKey     string
	SkipTLSVerify bool
}

func newKeaTransport(uri, httpUsername, httpPassword string, tlsOpts tlsOptions) (kea.Transport, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "unix":
		return &kea.UnixTransport{SocketPath: u.Path}, nil
	case "http":
		return &kea.HTTPTransport{
			Endpoint: uri,
			Username: httpUsername,
			Password: httpPassword,
		}, nil
	case "https":
		tlsConfig, err := newTLSConfig(tlsOpts)
		if err != nil {
			return nil, err
		}

		httpTransport := http.DefaultTransport.(*http.Transport).Clone()
		httpTransport.TLSClientConfig = tlsConfig

		return &kea.HTTPTransport{
			Endpoint: uri,
			Username: httpUsername,
			Password: httpPassword,
			Client:   &http.Client{Transport: httpTransport},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

func newTLSConfig(opts tlsOptions) (*tls.Config, error) {
	config := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: opts.SkipTLSVerify, //nolint:gosec
	}

	if opts.CACert != "" {
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(opts.CACert)) {
			return nil, fmt.Errorf("failed to parse ca_cert: no PEM-encoded certificates found")
		}
		config.RootCAs = pool
	}

	if opts.ClientCert != "" || opts.ClientKey != "" {
		if opts.ClientCert == "" || opts.ClientKey == "" {
			return nil, fmt.Errorf("client_cert and client_key must be set together")
		}

		cert, err := tls.X509KeyPair([]byte(opts.ClientCert), []byte(opts.ClientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	return config, nil
}
