package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"

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
	Address      types.String `tfsdk:"address"`
	HTTPUsername types.String `tfsdk:"http_username"`
	HTTPPassword types.String `tfsdk:"http_password"`

	TLSCAFile         types.String `tfsdk:"tls_ca_file"`
	TLSClientCertFile types.String `tfsdk:"tls_client_cert_file"`
	TLSClientKeyFile  types.String `tfsdk:"tls_client_key_file"`
	TLSServerName     types.String `tfsdk:"tls_server_name"`

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
				Description: "Connection settings for the Kea DHCPv4 server. Every change is followed by `config-write`, so that Kea keeps it across restarts.",
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
					"tls_ca_file": schema.StringAttribute{
						Optional: true,
						Description: "Path to a PEM CA bundle used to verify the server certificate of an `https` address instead of the system roots. " +
							"Falls back to `KEA_DHCP4_TLS_CA_FILE` environment variable.",
					},
					"tls_client_cert_file": schema.StringAttribute{
						Optional: true,
						Description: "Path to a PEM client certificate for mutual TLS. Requires `tls_client_key_file`. " +
							"Falls back to `KEA_DHCP4_TLS_CLIENT_CERT_FILE` environment variable.",
					},
					"tls_client_key_file": schema.StringAttribute{
						Optional: true,
						Description: "Path to the PEM private key of `tls_client_cert_file`. " +
							"Falls back to `KEA_DHCP4_TLS_CLIENT_KEY_FILE` environment variable.",
					},
					"tls_server_name": schema.StringAttribute{
						Optional: true,
						Description: "Server name used to verify the server certificate, when it differs from the host in `address`. " +
							"Falls back to `KEA_DHCP4_TLS_SERVER_NAME` environment variable.",
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
		func() resource.Resource { return &dhcp4.SubnetResource{} },
	}
}

func (p *KeaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &dhcp4.ClientClassDataSource{} },
		func() datasource.DataSource { return &dhcp4.ConfigHashDataSource{} },
		func() datasource.DataSource { return &dhcp4.ReservationDataSource{} },
		func() datasource.DataSource { return &dhcp4.SubnetDataSource{} },
	}
}

func configureDHCP4Client(data *KeaProviderClientModel) (*keadhcp4.Client, error) {
	address := configValue(data, func(m *KeaProviderClientModel) types.String { return m.Address }, "KEA_DHCP4_ADDRESS")

	// No address configured - skip DHCP4 client setup
	if address == "" {
		return nil, nil
	}

	transport, err := newKeaTransport(address, transportConfig{
		httpUsername:      configValue(data, func(m *KeaProviderClientModel) types.String { return m.HTTPUsername }, "KEA_DHCP4_HTTP_USERNAME"),
		httpPassword:      configValue(data, func(m *KeaProviderClientModel) types.String { return m.HTTPPassword }, "KEA_DHCP4_HTTP_PASSWORD"),
		tlsCAFile:         configValue(data, func(m *KeaProviderClientModel) types.String { return m.TLSCAFile }, "KEA_DHCP4_TLS_CA_FILE"),
		tlsClientCertFile: configValue(data, func(m *KeaProviderClientModel) types.String { return m.TLSClientCertFile }, "KEA_DHCP4_TLS_CLIENT_CERT_FILE"),
		tlsClientKeyFile:  configValue(data, func(m *KeaProviderClientModel) types.String { return m.TLSClientKeyFile }, "KEA_DHCP4_TLS_CLIENT_KEY_FILE"),
		tlsServerName:     configValue(data, func(m *KeaProviderClientModel) types.String { return m.TLSServerName }, "KEA_DHCP4_TLS_SERVER_NAME"),
	})
	if err != nil {
		return nil, err
	}

	var opts []keadhcp4.ClientOption
	if filename := configValue(data, func(m *KeaProviderClientModel) types.String { return m.ConfigWriteFilename }, "KEA_DHCP4_CONFIG_WRITE_FILENAME"); filename != "" {
		opts = append(opts, keadhcp4.WithConfigWriteFilename(filename))
	}

	return keadhcp4.NewClient(transport, opts...), nil
}

func configValue(data *KeaProviderClientModel, attr func(*KeaProviderClientModel) types.String, env string) string {
	if data != nil {
		if v := attr(data); !v.IsNull() && !v.IsUnknown() {
			return v.ValueString()
		}
	}
	return os.Getenv(env)
}

type transportConfig struct {
	httpUsername      string
	httpPassword      string
	tlsCAFile         string
	tlsClientCertFile string
	tlsClientKeyFile  string
	tlsServerName     string
}

func newKeaTransport(uri string, cfg transportConfig) (kea.Transport, error) {
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
			Username: cfg.httpUsername,
			Password: cfg.httpPassword,
		}, nil
	case "https":
		client, err := newTLSHTTPClient(cfg.tlsCAFile, cfg.tlsClientCertFile, cfg.tlsClientKeyFile, cfg.tlsServerName)
		if err != nil {
			return nil, err
		}
		return &kea.HTTPTransport{
			Endpoint: uri,
			Client:   client,
			Username: cfg.httpUsername,
			Password: cfg.httpPassword,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}
