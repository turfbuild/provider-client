package tf5

import (
	"context"

	"github.com/opentofu/provider-client/tofuprovider/providerops"
)

// InvokeAction implements tofuprovider.GRPCPluginProvider.
//
// Actions require protocol tfplugin6.10 or later, so the tfplugin5 transport
// has no implementation.
func (p *Provider) InvokeAction(ctx context.Context, req *providerops.InvokeActionRequest) (providerops.InvokeActionResponse, error) {
	panic("unimplemented")
}
