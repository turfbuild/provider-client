package tf5

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/opentofu/provider-client/tofuprovider/grpc/tfplugin5"
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerops"
)

// ValidateActionConfig implements tofuprovider.GRPCPluginProvider.
func (p *Provider) ValidateActionConfig(ctx context.Context, req *providerops.ValidateActionConfigRequest) (providerops.ValidateActionConfigResponse, error) {
	configVal, err := makeDynamicValueMsgpack(req.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid Config value: %w", err)
	}
	protoReq := &tfplugin5.ValidateActionConfig_Request{
		ActionType: req.ActionType,
		Config:     configVal,
	}
	protoResp, err := p.client.ValidateActionConfig(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return validateActionConfigResponse{proto: protoResp}, nil
}

type validateActionConfigResponse struct {
	proto *tfplugin5.ValidateActionConfig_Response
	common.SealedImpl
}

// Diagnostics implements providerops.ValidateActionConfigResponse.
func (r validateActionConfigResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: r.proto.Diagnostics}
}

// PlanAction implements tofuprovider.GRPCPluginProvider.
func (p *Provider) PlanAction(ctx context.Context, req *providerops.PlanActionRequest) (providerops.PlanActionResponse, error) {
	configVal, err := makeDynamicValueMsgpack(req.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid Config value: %w", err)
	}
	protoReq := &tfplugin5.PlanAction_Request{
		ActionType:         req.ActionType,
		Config:             configVal,
		ClientCapabilities: prepareClientCapabilities(req.ClientCapabilities),
	}
	protoResp, err := p.client.PlanAction(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return planActionResponse{proto: protoResp}, nil
}

type planActionResponse struct {
	proto *tfplugin5.PlanAction_Response
	common.SealedImpl
}

// Diagnostics implements providerops.PlanActionResponse.
func (r planActionResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: r.proto.Diagnostics}
}

// Deferred implements providerops.PlanActionResponse.
func (r planActionResponse) Deferred() providerops.Deferred {
	if r.proto.Deferred == nil {
		return nil
	}
	return deferred{proto: r.proto.Deferred}
}

// InvokeAction implements tofuprovider.GRPCPluginProvider.
func (p *Provider) InvokeAction(ctx context.Context, req *providerops.InvokeActionRequest) (providerops.InvokeActionResponse, error) {
	configVal, err := makeDynamicValueMsgpack(req.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid Config value: %w", err)
	}
	protoReq := &tfplugin5.InvokeAction_Request{
		ActionType:         req.ActionType,
		Config:             configVal,
		ClientCapabilities: prepareClientCapabilities(req.ClientCapabilities),
	}
	stream, err := p.client.InvokeAction(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return invokeActionResponse{stream: stream}, nil
}

type invokeActionResponse struct {
	stream grpc.ServerStreamingClient[tfplugin5.InvokeAction_Event]
	common.SealedImpl
}

// Recv implements providerops.InvokeActionResponse. It returns io.EOF (from the
// underlying gRPC stream) once the action's events are fully drained.
func (r invokeActionResponse) Recv() (providerops.InvokeActionEvent, error) {
	proto, err := r.stream.Recv()
	if err != nil {
		return nil, err
	}
	return invokeActionEvent{proto: proto}, nil
}

type invokeActionEvent struct {
	proto *tfplugin5.InvokeAction_Event
	common.SealedImpl
}

// Progress implements providerops.InvokeActionEvent.
func (e invokeActionEvent) Progress() (string, bool) {
	if progress := e.proto.GetProgress(); progress != nil {
		return progress.GetMessage(), true
	}
	return "", false
}

// Completed implements providerops.InvokeActionEvent.
func (e invokeActionEvent) Completed() (providerops.Diagnostics, bool) {
	if completed := e.proto.GetCompleted(); completed != nil {
		return diagnostics{proto: completed.GetDiagnostics()}, true
	}
	return nil, false
}
