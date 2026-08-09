package tf6

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/opentofu/provider-client/tofuprovider/grpc/tfplugin6"
	"github.com/opentofu/provider-client/tofuprovider/internal/common"
	"github.com/opentofu/provider-client/tofuprovider/providerops"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// makeResourceIdentityData wraps an outgoing identity value in the protocol's
// ResourceIdentityData envelope, returning nil when the caller supplied no
// identity. Every identity request field is optional — resource identity is
// opt-in per resource type — so an absent value is normal, not an error.
func makeResourceIdentityData(dv providerschema.DynamicValueIn) (*tfplugin6.ResourceIdentityData, error) {
	if dv == providerschema.NoDynamicValue {
		return nil, nil
	}
	raw, err := makeDynamicValueMsgpack(dv)
	if err != nil {
		return nil, err
	}
	return &tfplugin6.ResourceIdentityData{IdentityData: raw}, nil
}

// resourceIdentityOut unwraps an incoming ResourceIdentityData envelope.
// Both the envelope and the value inside it are guarded: a provider that does
// not implement identity for a resource type leaves the field unset, and a
// provider may also send an empty envelope.
func resourceIdentityOut(proto *tfplugin6.ResourceIdentityData) providerschema.DynamicValueOut {
	if proto == nil || proto.IdentityData == nil {
		return nil
	}
	return dynamicValue{proto: proto.IdentityData}
}

// GetResourceIdentitySchemas implements tofuprovider.GRPCPluginProvider.
func (p *Provider) GetResourceIdentitySchemas(ctx context.Context, req *providerops.GetResourceIdentitySchemasRequest) (providerops.GetResourceIdentitySchemasResponse, error) {
	protoReq := &tfplugin6.GetResourceIdentitySchemas_Request{
		// There are currently no fields in
		// providerops.GetResourceIdentitySchemasRequest, so nothing to
		// populate here.
	}
	protoResp, err := p.client.GetResourceIdentitySchemas(ctx, protoReq)
	if err != nil {
		// Providers built against a protocol older than this operation reject
		// the call outright. Resource identity is optional, so report that as
		// "no identity schemas" rather than propagating an error that every
		// caller would have to special-case.
		if status.Code(err) == codes.Unimplemented {
			return getResourceIdentitySchemasResponse{}, nil
		}
		return nil, err
	}
	return getResourceIdentitySchemasResponse{proto: protoResp}, nil
}

type getResourceIdentitySchemasResponse struct {
	// proto is nil when the provider does not implement the operation; both
	// methods below tolerate that.
	proto *tfplugin6.GetResourceIdentitySchemas_Response

	common.SealedImpl
}

// Diagnostics implements providerops.GetResourceIdentitySchemasResponse.
func (g getResourceIdentitySchemasResponse) Diagnostics() providerops.Diagnostics {
	if g.proto == nil {
		return diagnostics{}
	}
	return diagnostics{proto: g.proto.Diagnostics}
}

// IdentitySchemas implements providerops.GetResourceIdentitySchemasResponse.
func (g getResourceIdentitySchemasResponse) IdentitySchemas() iter.Seq2[string, providerschema.ResourceIdentitySchema] {
	if g.proto == nil {
		return func(yield func(string, providerschema.ResourceIdentitySchema) bool) {}
	}
	return common.MapSeq2(maps.All(g.proto.IdentitySchemas), func(name string, proto *tfplugin6.ResourceIdentitySchema) (string, providerschema.ResourceIdentitySchema) {
		return name, resourceIdentitySchema{proto: proto}
	})
}

type resourceIdentitySchema struct {
	proto *tfplugin6.ResourceIdentitySchema

	common.SealedImpl
}

// Version implements providerschema.ResourceIdentitySchema.
func (s resourceIdentitySchema) Version() int64 {
	return s.proto.Version
}

// Attributes implements providerschema.ResourceIdentitySchema.
func (s resourceIdentitySchema) Attributes() iter.Seq2[string, providerschema.ResourceIdentityAttribute] {
	return common.MapSeqToSeq2(slices.Values(s.proto.IdentityAttributes), func(proto *tfplugin6.ResourceIdentitySchema_IdentityAttribute) (string, providerschema.ResourceIdentityAttribute) {
		return proto.Name, resourceIdentityAttribute{proto: proto}
	})
}

type resourceIdentityAttribute struct {
	proto *tfplugin6.ResourceIdentitySchema_IdentityAttribute

	common.SealedImpl
}

// Name implements providerschema.ResourceIdentityAttribute.
func (a resourceIdentityAttribute) Name() string {
	return a.proto.Name
}

// Type implements providerschema.ResourceIdentityAttribute.
func (a resourceIdentityAttribute) Type() providerschema.TypeConstraint {
	if len(a.proto.Type) == 0 {
		return nil
	}
	return common.CtyTypeJSON(a.proto.Type)
}

// RequiredForImport implements providerschema.ResourceIdentityAttribute.
func (a resourceIdentityAttribute) RequiredForImport() bool {
	return a.proto.RequiredForImport
}

// OptionalForImport implements providerschema.ResourceIdentityAttribute.
func (a resourceIdentityAttribute) OptionalForImport() bool {
	return a.proto.OptionalForImport
}

// DocDescription implements providerschema.ResourceIdentityAttribute.
func (a resourceIdentityAttribute) DocDescription() (string, providerschema.DocStringFormat) {
	// The identity attribute description is documented as Markdown, and the
	// message carries no separate description-kind field.
	return a.proto.Description, providerschema.DocStringMarkdown
}

// UpgradeResourceIdentity implements tofuprovider.GRPCPluginProvider.
func (p *Provider) UpgradeResourceIdentity(ctx context.Context, req *providerops.UpgradeResourceIdentityRequest) (providerops.UpgradeResourceIdentityResponse, error) {
	if req.PrevIdentityRaw.JSON == nil {
		return nil, fmt.Errorf("missing required PrevIdentityRaw value")
	}
	protoReq := &tfplugin6.UpgradeResourceIdentity_Request{
		TypeName: req.ResourceType,
		Version:  req.IdentitySchemaVersion,
		RawIdentity: &tfplugin6.RawState{
			Json: req.PrevIdentityRaw.JSON,
		},
	}
	protoResp, err := p.client.UpgradeResourceIdentity(ctx, protoReq)
	if err != nil {
		return nil, err
	}
	return upgradeResourceIdentityResponse{proto: protoResp}, nil
}

type upgradeResourceIdentityResponse struct {
	proto *tfplugin6.UpgradeResourceIdentity_Response

	common.SealedImpl
}

// Diagnostics implements providerops.UpgradeResourceIdentityResponse.
func (u upgradeResourceIdentityResponse) Diagnostics() providerops.Diagnostics {
	return diagnostics{proto: u.proto.Diagnostics}
}

// UpgradedIdentity implements providerops.UpgradeResourceIdentityResponse.
func (u upgradeResourceIdentityResponse) UpgradedIdentity() providerschema.DynamicValueOut {
	return resourceIdentityOut(u.proto.UpgradedIdentity)
}
