package providerops

import (
	"iter"

	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"

	// For links in documentation comments:
	_ "maps"
)

type GetResourceIdentitySchemasRequest struct {
	// This operation takes no arguments. The struct exists so that future
	// protocol versions can add some without breaking callers.
}

type GetResourceIdentitySchemasResponse interface {
	// Diagnostics are any diagnostics included in the provider's response.
	//
	// If the result's [Diagnostics.HasErrors] method returns true then
	// the results of all other methods are unspecified and meaningless.
	Diagnostics() Diagnostics

	// IdentitySchemas returns the identity schema of each managed resource
	// type that the provider implements resource identity for.
	//
	// The first result of each item is the resource type name. Use
	// [maps.Collect] to produce a map from resource type name to schema.
	//
	// Resource types the provider does not implement identity for are absent
	// from the sequence, so callers must treat a missing entry as "no identity
	// for this type" rather than an error. The sequence is empty for providers
	// that do not implement identity at all, including providers predating the
	// operation — see [tofuprovider.GRPCPluginProvider.GetResourceIdentitySchemas].
	IdentitySchemas() iter.Seq2[string, providerschema.ResourceIdentitySchema]

	common.Sealed
}
