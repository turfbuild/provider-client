package tf6

import (
	"context"
	"fmt"
	"time"

	"github.com/opentofu/provider-client/tofuprovider/grpc/tfplugin6"
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerops"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OpenEphemeralResource implements tofuprovider.GRPCPluginProvider.
func (p *Provider) OpenEphemeralResource(ctx context.Context, req *providerops.OpenEphemeralResourceRequest) (providerops.OpenEphemeralResourceResponse, error) {
	configVal, err := makeDynamicValueMsgpack(req.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid Config value: %w", err)
	}
	protoReq := &tfplugin6.OpenEphemeralResource_Request{
		TypeName:           req.ResourceType,
		Config:             configVal,
		ClientCapabilities: prepareClientCapabilities(req.ClientCapabilities),
	}

	protoResp, err := p.client.OpenEphemeralResource(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return openEphemeralResourceResponse{proto: protoResp}, nil
}

type openEphemeralResourceResponse struct {
	proto *tfplugin6.OpenEphemeralResource_Response
	common.SealedImpl
}

// Diagnostics implements providerops.OpenEphemeralResourceResponse.
func (o openEphemeralResourceResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: o.proto.Diagnostics}
}

// Result implements providerops.OpenEphemeralResourceResponse.
func (o openEphemeralResourceResponse) Result() providerschema.DynamicValueOut {
	if o.proto.Result == nil {
		return nil
	}
	return dynamicValue{proto: o.proto.Result}
}

// RenewTime implements providerops.OpenEphemeralResourceResponse.
func (o openEphemeralResourceResponse) RenewTime() (time.Time, bool) {
	return renewTime(o.proto.RenewAt)
}

// ProviderInternal implements providerops.OpenEphemeralResourceResponse.
func (o openEphemeralResourceResponse) ProviderInternal() []byte {
	return o.proto.Private
}

// Deferred implements providerops.OpenEphemeralResourceResponse.
func (o openEphemeralResourceResponse) Deferred() providerops.Deferred {
	if o.proto.Deferred == nil {
		return nil
	}
	return deferred{proto: o.proto.Deferred}
}

// RenewEphemeralResource implements tofuprovider.GRPCPluginProvider.
func (p *Provider) RenewEphemeralResource(ctx context.Context, req *providerops.RenewEphemeralResourceRequest) (providerops.RenewEphemeralResourceResponse, error) {
	protoReq := &tfplugin6.RenewEphemeralResource_Request{
		TypeName: req.ResourceType,
		Private:  req.ProviderInternal,
	}

	protoResp, err := p.client.RenewEphemeralResource(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return renewEphemeralResourceResponse{proto: protoResp}, nil
}

type renewEphemeralResourceResponse struct {
	proto *tfplugin6.RenewEphemeralResource_Response
	common.SealedImpl
}

// Diagnostics implements providerops.RenewEphemeralResourceResponse.
func (r renewEphemeralResourceResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: r.proto.Diagnostics}
}

// RenewTime implements providerops.RenewEphemeralResourceResponse.
func (r renewEphemeralResourceResponse) RenewTime() (time.Time, bool) {
	return renewTime(r.proto.RenewAt)
}

// ProviderInternal implements providerops.RenewEphemeralResourceResponse.
func (r renewEphemeralResourceResponse) ProviderInternal() []byte {
	return r.proto.Private
}

// CloseEphemeralResource implements tofuprovider.GRPCPluginProvider.
func (p *Provider) CloseEphemeralResource(ctx context.Context, req *providerops.CloseEphemeralResourceRequest) (providerops.CloseEphemeralResourceResponse, error) {
	protoReq := &tfplugin6.CloseEphemeralResource_Request{
		TypeName: req.ResourceType,
		Private:  req.ProviderInternal,
	}

	protoResp, err := p.client.CloseEphemeralResource(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return closeEphemeralResourceResponse{proto: protoResp}, nil
}

type closeEphemeralResourceResponse struct {
	proto *tfplugin6.CloseEphemeralResource_Response
	common.SealedImpl
}

// Diagnostics implements providerops.CloseEphemeralResourceResponse.
func (c closeEphemeralResourceResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: c.proto.Diagnostics}
}

// ValidateEphemeralResourceConfig implements tofuprovider.GRPCPluginProvider.
func (p *Provider) ValidateEphemeralResourceConfig(ctx context.Context, req *providerops.ValidateEphemeralResourceConfigRequest) (providerops.ValidateEphemeralResourceConfigResponse, error) {
	configVal, err := makeDynamicValueMsgpack(req.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid Config value: %w", err)
	}
	protoReq := &tfplugin6.ValidateEphemeralResourceConfig_Request{
		TypeName: req.ResourceType,
		Config:   configVal,
	}

	protoResp, err := p.client.ValidateEphemeralResourceConfig(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return validateEphemeralResourceConfigResponse{proto: protoResp}, nil
}

type validateEphemeralResourceConfigResponse struct {
	proto *tfplugin6.ValidateEphemeralResourceConfig_Response
	common.SealedImpl
}

// Diagnostics implements providerops.ValidateEphemeralResourceConfigResponse.
func (v validateEphemeralResourceConfigResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: v.proto.Diagnostics}
}

// renewTime converts the protocol's optional renewal deadline. An absent
// timestamp means the provider wants no renewal at all, which the (time, bool)
// pair reports as false — the caller must not substitute a deadline of its own.
func renewTime(ts *timestamppb.Timestamp) (time.Time, bool) {
	if ts == nil {
		return time.Time{}, false
	}
	return ts.AsTime(), true
}
