package providerschema

import (
	"iter"

	"github.com/opentofu/provider-client/tofuprovider/internal/common"

	// For links in documentation comments:
	_ "maps"
)

// ResourceIdentitySchema describes the structure and types of the data a
// provider uses to identify instances of a managed resource type.
//
// Resource identity is a versioned object, separate from the resource's own
// schema, whose value is considered permanent for a given identity version and
// therefore suitable for long-term storage. Providers use it to compare
// resources — whether already managed or newly discovered — without relying on
// the shape of the resource's attributes.
//
// Identity is opt-in per resource type: a provider may implement it for some
// types and not others, and older providers do not implement it at all.
// Callers must therefore tolerate its absence everywhere.
type ResourceIdentitySchema interface {
	// Version is the identity version, separate from the resource's
	// [Schema.SchemaVersion].
	//
	// It changes whenever the structure or format of the identity attributes
	// changes. Versioning starts at 0, and identity data recorded under
	// differing versions must always be treated as unequal — compare versions
	// before comparing values, and use the UpgradeResourceIdentity operation
	// to migrate data recorded under an older version.
	Version() int64

	// Attributes returns an iterable sequence of the attributes making up this
	// identity.
	//
	// The first result of each item is the unique attribute name. Use
	// [maps.Collect] to produce a map from attribute name to definition.
	Attributes() iter.Seq2[string, ResourceIdentityAttribute]

	// This interface cannot be implemented outside of this module, because
	// future versions might extend the interface to include new protocol
	// features.
	common.Sealed
}

// ResourceIdentityAttribute describes a single attribute within a
// [ResourceIdentitySchema]. Every identity attribute participates in identity
// comparisons.
type ResourceIdentityAttribute interface {
	// Name is the identity attribute name.
	Name() string

	// Type returns the type constraint that any value assigned to this
	// attribute must conform to.
	Type() TypeConstraint

	// RequiredForImport returns true when this attribute must be supplied for
	// an identity-keyed ImportManagedResourceState call to succeed.
	RequiredForImport() bool

	// OptionalForImport returns true when this attribute need not be supplied
	// for an identity-keyed import, because the provider can supply it itself.
	// It may still be supplied.
	//
	// A correct provider sets exactly one of RequiredForImport and
	// OptionalForImport on each attribute.
	OptionalForImport() bool

	// DocDescription returns the provider's human-readable description of the
	// attribute. The second result describes the intended format for the
	// description string.
	DocDescription() (string, DocStringFormat)

	// This interface cannot be implemented outside of this module, because
	// future versions might extend the interface to include new protocol
	// features.
	common.Sealed
}
