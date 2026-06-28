package providerops

import (
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// InvokeActionRequest describes a request to invoke a provider-defined action.
//
// Actions are pure side effects: they do not read or write managed resource
// state. The provider streams progress back as the action runs and finishes
// with a terminal completion event carrying any diagnostics.
type InvokeActionRequest struct {
	// ActionType is the provider-defined action type to invoke
	// (e.g. "aws_lambda_invoke"), as reported by the action_schemas in the
	// provider's GetProviderSchema response.
	ActionType string

	// Config is the action's configuration object, encoded using the
	// serialization type from the action's schema.
	Config providerschema.DynamicValueIn
}

// InvokeActionResponse is the server-streaming result of invoking an action.
//
// Call Recv repeatedly to consume events until it returns io.EOF, which marks
// the end of the stream after the provider has sent its terminal completion
// event. Any other error from Recv is a transport-level failure.
type InvokeActionResponse interface {
	// Recv returns the next event emitted by the provider while invoking the
	// action. It returns an event with Completed() set when the action
	// finishes, and io.EOF once the stream is fully drained.
	Recv() (InvokeActionEvent, error)

	common.Sealed
}

// InvokeActionEvent is a single event in an action invocation stream: either an
// interim progress message or the terminal completion (carrying diagnostics).
type InvokeActionEvent interface {
	// Progress returns the human-readable progress message and true if this is
	// a progress event, or ("", false) otherwise.
	Progress() (string, bool)

	// Completed returns the terminal diagnostics and true if this is the
	// completion event, or (nil, false) otherwise. When this returns true the
	// action has finished; the next Recv call returns io.EOF.
	Completed() (Diagnostics, bool)

	common.Sealed
}
