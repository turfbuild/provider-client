package tf6

import (
	"context"
	"fmt"
	"iter"
	"slices"

	"github.com/opentofu/provider-client/tofuprovider/grpc/tfplugin6"
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerops"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// ApplyManagedResourceChange implements tofuprovider.GRPCPluginProvider.
func (p *Provider) ApplyManagedResourceChange(ctx context.Context, req *providerops.ApplyManagedResourceChangeRequest) (providerops.ApplyManagedResourceChangeResponse, error) {
	priorState, err := makeDynamicValueMsgpack(req.PriorState)
	if err != nil {
		return nil, fmt.Errorf("invalid PriorState value: %w", err)
	}
	plannedNewState, err := makeDynamicValueMsgpack(req.PlannedNewState)
	if err != nil {
		return nil, fmt.Errorf("invalid PlannedNewState value: %w", err)
	}
	config, err := makeDynamicValueMsgpack(req.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid Config value: %w", err)
	}

	var providerMeta *tfplugin6.DynamicValue
	if req.ProviderMeta != providerschema.NoDynamicValue {
		providerMeta, err = makeDynamicValueMsgpack(req.ProviderMeta)
		if err != nil {
			return nil, fmt.Errorf("invalid ProviderMeta value: %w", err)
		}
	}

	plannedIdentity, err := makeResourceIdentityData(req.PlannedNewIdentity)
	if err != nil {
		return nil, fmt.Errorf("invalid PlannedNewIdentity value: %w", err)
	}

	protoReq := &tfplugin6.ApplyResourceChange_Request{
		TypeName:        req.ResourceType,
		PriorState:      priorState,
		PlannedState:    plannedNewState,
		Config:          config,
		PlannedPrivate:  req.PlannedProviderInternal,
		ProviderMeta:    providerMeta,
		PlannedIdentity: plannedIdentity,
	}

	protoResp, err := p.client.ApplyResourceChange(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return applyManagedResourceChangeResponse{proto: protoResp}, nil
}

// ImportManagedResourceState implements tofuprovider.GRPCPluginProvider.
func (p *Provider) ImportManagedResourceState(ctx context.Context, req *providerops.ImportManagedResourceStateRequest) (providerops.ImportManagedResourceStateResponse, error) {
	identity, err := makeResourceIdentityData(req.Identity)
	if err != nil {
		return nil, fmt.Errorf("invalid Identity value: %w", err)
	}

	protoReq := &tfplugin6.ImportResourceState_Request{
		TypeName:           req.ResourceType,
		Id:                 req.ID,
		ClientCapabilities: prepareClientCapabilities(req.ClientCapabilities),
		Identity:           identity,
	}

	protoResp, err := p.client.ImportResourceState(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return importManagedResourceStateResponse{proto: protoResp}, nil
}

// MoveManagedResourceState implements tofuprovider.GRPCPluginProvider.
func (p *Provider) MoveManagedResourceState(ctx context.Context, req *providerops.MoveManagedResourceStateRequest) (providerops.MoveManagedResourceStateResponse, error) {
	panic("unimplemented")
}

// PlanManagedResourceChange implements tofuprovider.GRPCPluginProvider.
func (p *Provider) PlanManagedResourceChange(ctx context.Context, req *providerops.PlanManagedResourceChangeRequest) (providerops.PlanManagedResourceChangeResponse, error) {
	priorState, err := makeDynamicValueMsgpack(req.PriorState)
	if err != nil {
		return nil, fmt.Errorf("invalid PriorState value: %w", err)
	}
	proposedNewState, err := makeDynamicValueMsgpack(req.ProposedNewState)
	if err != nil {
		return nil, fmt.Errorf("invalid ProposedNewState value: %w", err)
	}
	config, err := makeDynamicValueMsgpack(req.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid Config value: %w", err)
	}

	var providerMeta *tfplugin6.DynamicValue
	if req.ProviderMeta != providerschema.NoDynamicValue {
		providerMeta, err = makeDynamicValueMsgpack(req.ProviderMeta)
		if err != nil {
			return nil, fmt.Errorf("invalid ProviderMeta value: %w", err)
		}
	}

	priorIdentity, err := makeResourceIdentityData(req.PriorIdentity)
	if err != nil {
		return nil, fmt.Errorf("invalid PriorIdentity value: %w", err)
	}

	protoReq := &tfplugin6.PlanResourceChange_Request{
		TypeName:           req.ResourceType,
		PriorState:         priorState,
		ProposedNewState:   proposedNewState,
		Config:             config,
		PriorPrivate:       req.PriorProviderInternal,
		ProviderMeta:       providerMeta,
		ClientCapabilities: prepareClientCapabilities(req.ClientCapabilities),
		PriorIdentity:      priorIdentity,
	}

	protoResp, err := p.client.PlanResourceChange(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return planManagedResourceChangeResponse{proto: protoResp}, nil
}

// ReadManagedResource implements tofuprovider.GRPCPluginProvider.
func (p *Provider) ReadManagedResource(ctx context.Context, req *providerops.ReadManagedResourceRequest) (providerops.ReadManagedResourceResponse, error) {
	currentState, err := makeDynamicValueMsgpack(req.CurrentState)
	if err != nil {
		return nil, fmt.Errorf("invalid CurrentState value: %w", err)
	}

	var providerMeta *tfplugin6.DynamicValue
	if req.ProviderMeta != providerschema.NoDynamicValue {
		providerMeta, err = makeDynamicValueMsgpack(req.ProviderMeta)
		if err != nil {
			return nil, fmt.Errorf("invalid ProviderMeta value: %w", err)
		}
	}

	currentIdentity, err := makeResourceIdentityData(req.CurrentIdentity)
	if err != nil {
		return nil, fmt.Errorf("invalid CurrentIdentity value: %w", err)
	}

	protoReq := &tfplugin6.ReadResource_Request{
		TypeName:           req.ResourceType,
		CurrentState:       currentState,
		Private:            req.ProviderInternal,
		ProviderMeta:       providerMeta,
		ClientCapabilities: prepareClientCapabilities(req.ClientCapabilities),
		CurrentIdentity:    currentIdentity,
	}

	protoResp, err := p.client.ReadResource(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return readManagedResourceResponse{proto: protoResp}, nil
}

// UpgradeManagedResourceState implements tofuprovider.GRPCPluginProvider.
func (p *Provider) UpgradeManagedResourceState(ctx context.Context, req *providerops.UpgradeManagedResourceStateRequest) (providerops.UpgradeManagedResourceStateResponse, error) {
	panic("unimplemented")
}

// ValidateManagedResourceConfig implements tofuprovider.GRPCPluginProvider.
func (p *Provider) ValidateManagedResourceConfig(ctx context.Context, req *providerops.ValidateManagedResourceConfigRequest) (providerops.ValidateManagedResourceConfigResponse, error) {
	configVal, err := makeDynamicValueMsgpack(req.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid Config value: %w", err)
	}
	protoReq := &tfplugin6.ValidateResourceConfig_Request{
		TypeName:           req.ResourceType,
		Config:             configVal,
		ClientCapabilities: prepareClientCapabilities(req.ClientCapabilities),
	}

	protoResp, err := p.client.ValidateResourceConfig(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return validateManagedResourceConfigResponse{proto: protoResp}, nil
}

type validateManagedResourceConfigResponse struct {
	proto *tfplugin6.ValidateResourceConfig_Response
	common.SealedImpl
}

// Diagnostics implements providerops.ValidateEphemeralResourceConfigResponse.
func (v validateManagedResourceConfigResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: v.proto.Diagnostics}
}

type planManagedResourceChangeResponse struct {
	proto *tfplugin6.PlanResourceChange_Response
	common.SealedImpl
}

// Diagnostics implements providerops.PlanManagedResourceChangeResponse.
func (p planManagedResourceChangeResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: p.proto.Diagnostics}
}

// PlannedNewState implements providerops.PlanManagedResourceChangeResponse.
func (p planManagedResourceChangeResponse) PlannedNewState() providerschema.DynamicValueOut {
	return dynamicValue{proto: p.proto.PlannedState}
}

// PlannedProviderInternal implements providerops.PlanManagedResourceChangeResponse.
func (p planManagedResourceChangeResponse) PlannedProviderInternal() []byte {
	return p.proto.PlannedPrivate
}

// LegacyTypeSystem implements providerops.PlanManagedResourceChangeResponse.
func (p planManagedResourceChangeResponse) LegacyTypeSystem() bool {
	return p.proto.LegacyTypeSystem
}

// Deferred implements providerops.PlanManagedResourceChangeResponse.
func (p planManagedResourceChangeResponse) Deferred() providerops.Deferred {
	if p.proto.Deferred == nil {
		return nil
	}
	return deferred{proto: p.proto.Deferred}
}

// RequiresReplace implements providerops.PlanManagedResourceChangeResponse.
func (p planManagedResourceChangeResponse) RequiresReplace() iter.Seq[providerops.AttributePath] {
	return common.MapSeq(slices.Values(p.proto.RequiresReplace), func(protoPath *tfplugin6.AttributePath) providerops.AttributePath {
		return attributePath{proto: protoPath}
	})
}

// PlannedNewIdentity implements providerops.PlanManagedResourceChangeResponse.
func (p planManagedResourceChangeResponse) PlannedNewIdentity() providerschema.DynamicValueOut {
	return resourceIdentityOut(p.proto.PlannedIdentity)
}

type attributePath struct {
	proto *tfplugin6.AttributePath
	common.SealedImpl
}

// Steps implements providerops.AttributePath.
func (a attributePath) Steps() iter.Seq[providerops.AttributePathStep] {
	return common.MapSeq(slices.Values(a.proto.Steps), func(protoStep *tfplugin6.AttributePath_Step) providerops.AttributePathStep {
		return attributePathStep{proto: protoStep}
	})
}

type attributePathStep struct {
	proto *tfplugin6.AttributePath_Step
	common.SealedImpl
}

// AttributeName implements providerops.AttributePathStep.
func (s attributePathStep) AttributeName() (string, bool) {
	if step, ok := s.proto.Selector.(*tfplugin6.AttributePath_Step_AttributeName); ok {
		return step.AttributeName, true
	}
	return "", false
}

// ElementKeyString implements providerops.AttributePathStep.
func (s attributePathStep) ElementKeyString() (string, bool) {
	if step, ok := s.proto.Selector.(*tfplugin6.AttributePath_Step_ElementKeyString); ok {
		return step.ElementKeyString, true
	}
	return "", false
}

// ElementKeyInt implements providerops.AttributePathStep.
func (s attributePathStep) ElementKeyInt() (int64, bool) {
	if step, ok := s.proto.Selector.(*tfplugin6.AttributePath_Step_ElementKeyInt); ok {
		return step.ElementKeyInt, true
	}
	return 0, false
}

type applyManagedResourceChangeResponse struct {
	proto *tfplugin6.ApplyResourceChange_Response
	common.SealedImpl
}

// Diagnostics implements providerops.ApplyManagedResourceChangeResponse.
func (a applyManagedResourceChangeResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: a.proto.Diagnostics}
}

// PlannedNewState implements providerops.ApplyManagedResourceChangeResponse.
func (a applyManagedResourceChangeResponse) PlannedNewState() providerschema.DynamicValueOut {
	if a.proto.NewState == nil {
		return nil
	}
	return dynamicValue{proto: a.proto.NewState}
}

// ProviderInternal implements providerops.ApplyManagedResourceChangeResponse.
func (a applyManagedResourceChangeResponse) ProviderInternal() []byte {
	return a.proto.Private
}

// LegacyTypeSystem implements providerops.ApplyManagedResourceChangeResponse.
func (a applyManagedResourceChangeResponse) LegacyTypeSystem() bool {
	return a.proto.LegacyTypeSystem
}

// NewIdentity implements providerops.ApplyManagedResourceChangeResponse.
func (a applyManagedResourceChangeResponse) NewIdentity() providerschema.DynamicValueOut {
	return resourceIdentityOut(a.proto.NewIdentity)
}

type readManagedResourceResponse struct {
	proto *tfplugin6.ReadResource_Response
	common.SealedImpl
}

// Diagnostics implements providerops.ReadManagedResourceResponse.
func (r readManagedResourceResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: r.proto.Diagnostics}
}

// NewState implements providerops.ReadManagedResourceResponse.
func (r readManagedResourceResponse) NewState() providerschema.DynamicValueOut {
	if r.proto.NewState == nil {
		return nil
	}
	return dynamicValue{proto: r.proto.NewState}
}

// ProviderInternal implements providerops.ReadManagedResourceResponse.
func (r readManagedResourceResponse) ProviderInternal() []byte {
	return r.proto.Private
}

// Deferred implements providerops.ReadManagedResourceResponse.
func (r readManagedResourceResponse) Deferred() providerops.Deferred {
	if r.proto.Deferred == nil {
		return nil
	}
	return deferred{proto: r.proto.Deferred}
}

// NewIdentity implements providerops.ReadManagedResourceResponse.
func (r readManagedResourceResponse) NewIdentity() providerschema.DynamicValueOut {
	return resourceIdentityOut(r.proto.NewIdentity)
}

type importManagedResourceStateResponse struct {
	proto *tfplugin6.ImportResourceState_Response
	common.SealedImpl
}

// Diagnostics implements providerops.ImportManagedResourceStateResponse.
func (i importManagedResourceStateResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: i.proto.Diagnostics}
}

// ImportedResources implements providerops.ImportManagedResourceStateResponse.
func (i importManagedResourceStateResponse) ImportedResources() iter.Seq[providerops.ImportedManagedResource] {
	return common.MapSeq(slices.Values(i.proto.ImportedResources), func(protoRes *tfplugin6.ImportResourceState_ImportedResource) providerops.ImportedManagedResource {
		return importedManagedResource{proto: protoRes}
	})
}

// Deferred implements providerops.ImportManagedResourceStateResponse.
func (i importManagedResourceStateResponse) Deferred() providerops.Deferred {
	if i.proto.Deferred == nil {
		return nil
	}
	return deferred{proto: i.proto.Deferred}
}

type importedManagedResource struct {
	proto *tfplugin6.ImportResourceState_ImportedResource
	common.SealedImpl
}

// ResourceType implements providerops.ImportedManagedResource.
func (i importedManagedResource) ResourceType() string {
	return i.proto.TypeName
}

// State implements providerops.ImportedManagedResource.
func (i importedManagedResource) State() providerschema.DynamicValueOut {
	if i.proto.State == nil {
		return nil
	}
	return dynamicValue{proto: i.proto.State}
}

// ProviderInternal implements providerops.ImportedManagedResource.
func (i importedManagedResource) ProviderInternal() []byte {
	return i.proto.Private
}

// Identity implements providerops.ImportedManagedResource.
func (i importedManagedResource) Identity() providerschema.DynamicValueOut {
	return resourceIdentityOut(i.proto.Identity)
}
