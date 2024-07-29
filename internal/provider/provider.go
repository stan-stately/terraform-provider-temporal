package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"go.temporal.io/sdk/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &temporalProvider{}
)

// temporalProviderModel maps provider schema data to a Go type.
type temporalProviderModel struct {
	Address   types.String `tfsdk:"address"`
	Namespace types.String `tfsdk:"namespace"`
}

type providerConfig struct {
	client    client.Client
	namespace string
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &temporalProvider{
			version: version,
		}
	}
}

// temporalProvider is the provider implementation.
type temporalProvider struct {
	version string
}

// Metadata returns the provider type name.
func (p *temporalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "temporal"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *temporalProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Temporal Terraform Provider allows you to manage [Temporal](https://temporal.io/) resources using [Terraform](https://www.terraform.io/).",
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				Optional:    true,
				MarkdownDescription: "Address of the Temporal server. Of the form `host:port`.",
			},
			"namespace": schema.StringAttribute{
				Optional:    true,
				Description: "Namespace to operate in.",
			},
		},
	}
}

// Configure prepares a Temporal API client for data sources and resources.
func (p *temporalProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config temporalProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	address := os.Getenv("TEMPORAL_ADDRESS")
	if !config.Address.IsNull() {
		address = config.Address.ValueString()
	}

	namespace := "default"
	if !config.Namespace.IsNull() {
		namespace = config.Namespace.ValueString()
	} else if os.Getenv("TEMPORAL_NAMESPACE") != "" {
		namespace = os.Getenv("TEMPORAL_NAMESPACE")
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if address == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("address"),
			"Missing Temporal API Address",
			"The provider cannot create the Temporal API client as there is a missing or empty value for the Temporal API host. "+
				"Set the host value in the configuration or use the TEMPORAL_ADDRESS environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		return
	}

	temporalClient, err := client.Dial(client.Options{
		HostPort:  address,
		Namespace: namespace,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to establish a connection with the Temporal server on "+address, err.Error())
		return
	}

	cfg := &providerConfig{
		client:    temporalClient,
		namespace: namespace,
	}

	resp.DataSourceData = cfg
	resp.ResourceData = cfg
}

// DataSources defines the data sources implemented in the provider.
func (p *temporalProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Resources defines the resources implemented in the provider.
func (p *temporalProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNamespaceResource,
		NewScheduleResource,
	}
}
