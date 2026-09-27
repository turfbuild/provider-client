package providerops

import (
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// PlanActionRequest describes a request for a configured provider to consider
// an action's configuration ahead of its invocation.
//
// An action has no state and produces no planned value, so this is not the
// action's counterpart of PlanManagedResourceChange: there is nothing to merge
// and nothing to predict. It exists so that a provider can report plan-time
// problems only a configured provider can see, and — when the caller permits
// deferral — say that the action cannot be invoked yet because something it
// depends on is not known.
type PlanActionRequest struct {
	// ActionType is the provider-defined action type to plan, as reported by
	// the action_schemas in the provider's GetProviderSchema response.
	ActionType string

	// Config is a dynamic value representation of the object value
	// representing the action's configuration. It may carry unknown values.
	//
	// A value must be provided and its serialization type must be the implied
	// type of the schema given by this provider's
	// [providerschema.ProviderSchema.ActionSchemas] method.
	Config providerschema.DynamicValueIn

	// ClientCapabilities allows the caller to declare that it is capable of
	// handling certain response data that was added to the protocol after
	// it was initially defined, and thus which the provider must disable
	// by default to avoid confusing older clients.
	//
	// In particular, a provider may answer with a non-nil Deferred only when
	// SupportsDeferral is set.
	ClientCapabilities *ClientCapabilities
}

// PlanActionResponse is the result of planning an action.
type PlanActionResponse interface {
	// Diagnostics are any diagnostics included in the provider's response.
	//
	// If the result's [Diagnostics.HasErrors] method returns true then the
	// action must not be invoked with this configuration.
	Diagnostics() Diagnostics

	// Deferred returns a non-nil value if the provider does not have enough
	// information to invoke this action yet, such as if the provider
	// configuration is not yet known enough to know which API endpoint to
	// connect to, or if the action's own configuration is not known enough.
	//
	// If this returns nil then the provider expects to be able to invoke the
	// action with this configuration once any unknown values in it are
	// decided.
	Deferred() Deferred

	common.Sealed
}
