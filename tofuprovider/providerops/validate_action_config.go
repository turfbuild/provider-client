package providerops

import (
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// ValidateActionConfigRequest describes a request to validate the
// configuration of a provider-defined action per the provider's
// internally-implemented validation rules.
//
// Like the validation of a managed resource configuration, this is a
// stateless check against an unconfigured provider: the configuration may
// carry unknown values, and the provider reports only what it can tell
// without its own configuration applied.
type ValidateActionConfigRequest struct {
	// ActionType is the provider-defined action type the configuration is
	// intended for, as reported by the action_schemas in the provider's
	// GetProviderSchema response.
	ActionType string

	// Config is a dynamic value representation of the object value
	// representing the action's configuration.
	//
	// A value must be provided and its serialization type must be the implied
	// type of the schema given by this provider's
	// [providerschema.ProviderSchema.ActionSchemas] method.
	Config providerschema.DynamicValueIn
}

// ValidateActionConfigResponse is the result of validating an action
// configuration.
type ValidateActionConfigResponse interface {
	// Diagnostics describe any problems the provider reported with the
	// provided configuration.
	//
	// If this includes any error diagnostics then the configuration object is
	// somehow invalid, and passing it to PlanAction or InvokeAction causes
	// unspecified behavior.
	Diagnostics() Diagnostics

	common.Sealed
}
