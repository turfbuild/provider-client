package providerops

import (
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

type ValidateManagedResourceConfigRequest struct {
	// ResourceType is the name of the type of resource the given configuration
	// is intended for.
	ResourceType string

	// Config is a dynamic value representation of the object value
	// representing the resource configuration.
	//
	// A value must be provided and its serialization type must be the implied
	// type of the schema given in by this provider's
	// [providerschema.ProviderSchema.ManagedResourceTypeSchemas] method.
	Config providerschema.DynamicValueIn

	// ClientCapabilities allows the caller to declare that it is capable of
	// handling protocol features the provider must otherwise disable to avoid
	// confusing older clients.
	//
	// It matters here and not on the data-source twin because a write-only
	// attribute is a managed-resource idea: a provider whose caller has not
	// declared SupportsWriteOnlyAttributes rejects any configuration that sets
	// one, at validation, before the plan is ever attempted. Protocol 5 has no
	// equivalent field on this request, so this is honoured for protocol 6 only.
	ClientCapabilities *ClientCapabilities
}

type ValidateManagedResourceConfigResponse interface {
	// Diagnostics describe any problems the provider reported with the
	// provided configuration.
	//
	// If this includes any error diagnostics then passing the configuration
	// object is somehow invalid and so passing it to other methods causes
	// unspecified behavior.
	Diagnostics() Diagnostics

	common.Sealed
}
