package providerops

import (
	"iter"

	"github.com/opentofu/provider-client/tofuprovider/internal/common"
)

// AttributePath represents a path to a specific attribute within a resource's
// configuration or state. It is used to identify which attributes require
// replacement when a resource cannot be updated in-place.
type AttributePath interface {
	// Steps returns an iterable sequence of the path steps from root to leaf.
	Steps() iter.Seq[AttributePathStep]

	common.Sealed
}

// AttributePathStep represents a single step in an attribute path.
type AttributePathStep interface {
	// AttributeName returns the attribute name if this step selects an
	// attribute by name, along with true. Returns ("", false) otherwise.
	AttributeName() (string, bool)

	// ElementKeyString returns the string key if this step selects a map
	// element by string key, along with true. Returns ("", false) otherwise.
	ElementKeyString() (string, bool)

	// ElementKeyInt returns the integer index if this step selects a list
	// or tuple element by index, along with true. Returns (0, false) otherwise.
	ElementKeyInt() (int64, bool)

	common.Sealed
}
