package providerops

import (
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

type UpgradeResourceIdentityRequest struct {
	// ResourceType is the name of the type of resource the given identity data
	// was created by.
	ResourceType string

	// IdentitySchemaVersion is the identity version that was current when the
	// provided raw identity data was created. This is separate from the
	// resource type's own schema version.
	//
	// Callers must save that version number alongside the identity data in
	// their state snapshot and provide it here, since the provider may need to
	// vary its behavior based on the starting version.
	IdentitySchemaVersion int64

	// PrevIdentityRaw is the raw representation of the previously-saved
	// identity.
	//
	// As with UpgradeManagedResourceState, clients are not expected to have
	// access to schema information for older versions of a provider, so the
	// client skips trying to decode the data itself and assumes the provider
	// knows how to decode data created by earlier versions of itself.
	PrevIdentityRaw providerschema.RawState
}

type UpgradeResourceIdentityResponse interface {
	// Diagnostics are any diagnostics included in the provider's response.
	//
	// If the result's [Diagnostics.HasErrors] method returns true then
	// the results of all other methods are unspecified and meaningless.
	Diagnostics() Diagnostics

	// UpgradedIdentity is the upgraded identity data.
	//
	// This must be decoded using the type implied by the current identity
	// schema of the resource type, and re-saved under that schema's version.
	UpgradedIdentity() providerschema.DynamicValueOut

	common.Sealed
}
