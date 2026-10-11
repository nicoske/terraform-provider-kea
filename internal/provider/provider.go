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
				Description: "Connection settings for the Kea DHCPv4 server.",
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
		func() resource.Resource { return &dhcp4.ReservationResource{} },
	}
}

func (p *KeaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &dhcp4.ReservationDataSource{} },
	}
}

func configureDHCP4Client(data *KeaProviderClientModel) (*keadhcp4.Client, error) {
	var address string
	if data != nil && !data.Address.IsNull() && !data.Address.IsUnknown() {
		address = data.Address.ValueString()
	} else if s, ok := os.LookupEnv("KEA_DHCP4_ADDRESS"); ok {
		address = s
	}

	// No address configured - skip DHCP4 client setup
	if address == "" {
		return nil, nil
	}

	var http_username string
	if data != nil && !data.HTTPUsername.IsNull() && !data.HTTPUsername.IsUnknown() {
		http_username = data.HTTPUsername.ValueString()
	} else if s, ok := os.LookupEnv("KEA_DHCP4_HTTP_USERNAME"); ok {
		http_username = s
	}

	var http_password string
	if data != nil && !data.HTTPPassword.IsNull() && !data.HTTPPassword.IsUnknown() {
		http_password = data.HTTPPassword.ValueString()
	} else if s, ok := os.LookupEnv("KEA_DHCP4_HTTP_PASSWORD"); ok {
		http_password = s
	}

	transport, err := newKeaTransport(address, http_username, http_password)
	if err != nil {
		return nil, err
	}

	client := keadhcp4.NewClient(transport)
	return client, nil
}

func NewDHCP4ClientFromEnv() (*keadhcp4.Client, error) {
	return configureDHCP4Client(nil)
}

func newKeaTransport(uri, httpUsername, httpPassword string) (kea.Transport, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "unix":
		return &kea.UnixTransport{SocketPath: u.Path}, nil
	case "http", "https":
		return &kea.HTTPTransport{
			Endpoint: uri,
			Username: httpUsername,
			Password: httpPassword,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}
