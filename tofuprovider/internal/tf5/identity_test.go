package tf5_test

import (
	"maps"
	"testing"

	"github.com/zclconf/go-cty/cty"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/opentofu/provider-client/tofuprovider/grpc/tfplugin5"
	"github.com/opentofu/provider-client/tofuprovider/internal/mockutil"
	"github.com/opentofu/provider-client/tofuprovider/internal/tf5"
	"github.com/opentofu/provider-client/tofuprovider/providerops"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// identityMsgpack is the wire encoding of the identity object {"id":"abc123"},
// as a provider would send it.
func identityMsgpack(t *testing.T) []byte {
	t.Helper()
	ty := cty.Object(map[string]cty.Type{"id": cty.String})
	v := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("abc123")})
	dv := providerschema.NewDynamicValue(v, ty)
	// Round-trip through the same encoder the request path uses, so the
	// fixture cannot drift from the implementation.
	raw, err := providerschema.NewRawState(dv.Value(), dv.SerializationType())
	if err != nil {
		t.Fatalf("encoding identity fixture: %s", err)
	}
	return raw.JSON
}

func identityIn(t *testing.T) providerschema.DynamicValueIn {
	t.Helper()
	return providerschema.NewDynamicValue(
		cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("abc123")}),
		cty.Object(map[string]cty.Type{"id": cty.String}),
	)
}

// checkIdentityValue asserts that an identity returned by a response accessor
// decodes back to the fixture object.
func checkIdentityValue(t *testing.T, got providerschema.DynamicValueOut) {
	t.Helper()
	if got == nil {
		t.Fatal("identity is nil, want a value")
	}
	v, err := got.AsCtyValue(cty.Object(map[string]cty.Type{"id": cty.String}))
	if err != nil {
		t.Fatalf("decoding identity: %s", err)
	}
	if want := cty.StringVal("abc123"); !v.GetAttr("id").RawEquals(want) {
		t.Errorf("wrong identity id\ngot:  %#v\nwant: %#v", v.GetAttr("id"), want)
	}
}

// TestReadManagedResourceIdentity covers both directions on the read path: the
// request carries CurrentIdentity in the ResourceIdentityData envelope, and the
// response accessor tolerates every shape of absence.
func TestReadManagedResourceIdentity(t *testing.T) {
	testProviderCalls(t, map[string]providerCallTest[*providerops.ReadManagedResourceRequest, providerops.ReadManagedResourceResponse]{
		"sends current identity": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.ReadResource(gomock.Any(), mockutil.Eq(&tfplugin5.ReadResource_Request{
					TypeName:     "test",
					CurrentState: &tfplugin5.DynamicValue{Msgpack: []byte{0x81, 0xa2, 0x69, 0x64, 0xa6, 0x61, 0x62, 0x63, 0x31, 0x32, 0x33}},
					CurrentIdentity: &tfplugin5.ResourceIdentityData{
						IdentityData: &tfplugin5.DynamicValue{Msgpack: []byte{0x81, 0xa2, 0x69, 0x64, 0xa6, 0x61, 0x62, 0x63, 0x31, 0x32, 0x33}},
					},
				})).Return(&tfplugin5.ReadResource_Response{}, nil)
			},
			Request: &providerops.ReadManagedResourceRequest{
				ResourceType:    "test",
				CurrentState:    identityIn(t),
				CurrentIdentity: identityIn(t),
			},
		},
		"omits identity entirely when unset": {
			// The common case: the resource type declares no identity schema.
			// The envelope must be absent, not an empty message, or a provider
			// may read it as "identity present but empty".
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.ReadResource(gomock.Any(), mockutil.Eq(&tfplugin5.ReadResource_Request{
					TypeName:        "test",
					CurrentState:    &tfplugin5.DynamicValue{Msgpack: []byte{0x81, 0xa2, 0x69, 0x64, 0xa6, 0x61, 0x62, 0x63, 0x31, 0x32, 0x33}},
					CurrentIdentity: nil,
				})).Return(&tfplugin5.ReadResource_Response{}, nil)
			},
			Request: &providerops.ReadManagedResourceRequest{
				ResourceType: "test",
				CurrentState: identityIn(t),
			},
		},
		"reads new identity back": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.ReadResource(gomock.Any(), gomock.Any()).Return(&tfplugin5.ReadResource_Response{
					NewIdentity: &tfplugin5.ResourceIdentityData{
						IdentityData: &tfplugin5.DynamicValue{Msgpack: []byte{0x81, 0xa2, 0x69, 0x64, 0xa6, 0x61, 0x62, 0x63, 0x31, 0x32, 0x33}},
					},
				}, nil)
			},
			Request: &providerops.ReadManagedResourceRequest{ResourceType: "test", CurrentState: identityIn(t)},
			Check: func(t *testing.T, resp providerops.ReadManagedResourceResponse) {
				checkIdentityValue(t, resp.NewIdentity())
			},
		},
		"nil new identity when the provider sends none": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.ReadResource(gomock.Any(), gomock.Any()).Return(&tfplugin5.ReadResource_Response{}, nil)
			},
			Request: &providerops.ReadManagedResourceRequest{ResourceType: "test", CurrentState: identityIn(t)},
			Check: func(t *testing.T, resp providerops.ReadManagedResourceResponse) {
				if got := resp.NewIdentity(); got != nil {
					t.Errorf("expected nil identity, got %#v", got)
				}
			},
		},
		"nil new identity when the envelope is empty": {
			// A provider may send the envelope with no value inside it. The
			// inner field must be guarded too, or callers get a non-nil
			// DynamicValueOut that fails on decode.
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.ReadResource(gomock.Any(), gomock.Any()).Return(&tfplugin5.ReadResource_Response{
					NewIdentity: &tfplugin5.ResourceIdentityData{},
				}, nil)
			},
			Request: &providerops.ReadManagedResourceRequest{ResourceType: "test", CurrentState: identityIn(t)},
			Check: func(t *testing.T, resp providerops.ReadManagedResourceResponse) {
				if got := resp.NewIdentity(); got != nil {
					t.Errorf("expected nil identity for an empty envelope, got %#v", got)
				}
			},
		},
	}, func(t *testing.T, provider *tf5.Provider, req *providerops.ReadManagedResourceRequest) (providerops.ReadManagedResourceResponse, error) {
		return provider.ReadManagedResource(t.Context(), req)
	})
}

// TestImportManagedResourceStateIdentity covers identity-keyed import: the
// request carries the identity object instead of (or alongside) a string ID,
// and each imported resource can report its own identity back.
func TestImportManagedResourceStateIdentity(t *testing.T) {
	testProviderCalls(t, map[string]providerCallTest[*providerops.ImportManagedResourceStateRequest, providerops.ImportManagedResourceStateResponse]{
		"sends identity instead of an id": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.ImportResourceState(gomock.Any(), mockutil.Eq(&tfplugin5.ImportResourceState_Request{
					TypeName: "test",
					Identity: &tfplugin5.ResourceIdentityData{
						IdentityData: &tfplugin5.DynamicValue{Msgpack: []byte{0x81, 0xa2, 0x69, 0x64, 0xa6, 0x61, 0x62, 0x63, 0x31, 0x32, 0x33}},
					},
				})).Return(&tfplugin5.ImportResourceState_Response{}, nil)
			},
			Request: &providerops.ImportManagedResourceStateRequest{
				ResourceType: "test",
				Identity:     identityIn(t),
			},
		},
		"reads imported identity back": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.ImportResourceState(gomock.Any(), gomock.Any()).Return(&tfplugin5.ImportResourceState_Response{
					ImportedResources: []*tfplugin5.ImportResourceState_ImportedResource{
						{
							TypeName: "test",
							Identity: &tfplugin5.ResourceIdentityData{
								IdentityData: &tfplugin5.DynamicValue{Msgpack: []byte{0x81, 0xa2, 0x69, 0x64, 0xa6, 0x61, 0x62, 0x63, 0x31, 0x32, 0x33}},
							},
						},
					},
				}, nil)
			},
			Request: &providerops.ImportManagedResourceStateRequest{ResourceType: "test", ID: "abc123"},
			Check: func(t *testing.T, resp providerops.ImportManagedResourceStateResponse) {
				var n int
				for imported := range resp.ImportedResources() {
					n++
					checkIdentityValue(t, imported.Identity())
				}
				if n != 1 {
					t.Errorf("expected 1 imported resource, got %d", n)
				}
			},
		},
	}, func(t *testing.T, provider *tf5.Provider, req *providerops.ImportManagedResourceStateRequest) (providerops.ImportManagedResourceStateResponse, error) {
		return provider.ImportManagedResourceState(t.Context(), req)
	})
}

// TestGetResourceIdentitySchemas covers the schema-discovery operation,
// including the two "no identity" paths every caller must tolerate: a provider
// that returns an empty map, and one predating the RPC that rejects the call.
func TestGetResourceIdentitySchemas(t *testing.T) {
	testProviderCalls(t, map[string]providerCallTest[*providerops.GetResourceIdentitySchemasRequest, providerops.GetResourceIdentitySchemasResponse]{
		"returns schemas": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.GetResourceIdentitySchemas(gomock.Any(), mockutil.Eq(&tfplugin5.GetResourceIdentitySchemas_Request{})).Return(
					&tfplugin5.GetResourceIdentitySchemas_Response{
						IdentitySchemas: map[string]*tfplugin5.ResourceIdentitySchema{
							"test": {
								Version: 2,
								IdentityAttributes: []*tfplugin5.ResourceIdentitySchema_IdentityAttribute{
									{
										Name:              "id",
										Type:              []byte(`"string"`),
										RequiredForImport: true,
										Description:       "The ID.",
									},
									{
										Name:              "region",
										Type:              []byte(`"string"`),
										OptionalForImport: true,
									},
								},
							},
						},
					}, nil)
			},
			Request: &providerops.GetResourceIdentitySchemasRequest{},
			Check: func(t *testing.T, resp providerops.GetResourceIdentitySchemasResponse) {
				schemas := maps.Collect(resp.IdentitySchemas())
				if len(schemas) != 1 {
					t.Fatalf("expected 1 identity schema, got %d", len(schemas))
				}
				s, ok := schemas["test"]
				if !ok {
					t.Fatal(`no identity schema for "test"`)
				}
				if got, want := s.Version(), int64(2); got != want {
					t.Errorf("wrong version: got %d, want %d", got, want)
				}
				attrs := maps.Collect(s.Attributes())
				if len(attrs) != 2 {
					t.Fatalf("expected 2 identity attributes, got %d", len(attrs))
				}
				id := attrs["id"]
				if !id.RequiredForImport() {
					t.Error("id should be required for import")
				}
				if id.OptionalForImport() {
					t.Error("id should not be optional for import")
				}
				if got, _ := id.DocDescription(); got != "The ID." {
					t.Errorf("wrong description: %q", got)
				}
				ty, err := id.Type().AsCtyType()
				if err != nil {
					t.Fatalf("decoding attribute type: %s", err)
				}
				if !ty.Equals(cty.String) {
					t.Errorf("wrong attribute type: %#v", ty)
				}
				if !attrs["region"].OptionalForImport() {
					t.Error("region should be optional for import")
				}
			},
		},
		"empty when the provider implements no identity": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.GetResourceIdentitySchemas(gomock.Any(), gomock.Any()).Return(
					&tfplugin5.GetResourceIdentitySchemas_Response{}, nil)
			},
			Request: &providerops.GetResourceIdentitySchemasRequest{},
			Check: func(t *testing.T, resp providerops.GetResourceIdentitySchemasResponse) {
				if n := len(maps.Collect(resp.IdentitySchemas())); n != 0 {
					t.Errorf("expected no identity schemas, got %d", n)
				}
			},
		},
		"unimplemented is absorbed, not an error": {
			// Providers predating the operation reject the call. Callers should
			// not have to distinguish that from "implements no identity".
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.GetResourceIdentitySchemas(gomock.Any(), gomock.Any()).Return(
					nil, status.Error(codes.Unimplemented, "unknown method GetResourceIdentitySchemas"))
			},
			Request: &providerops.GetResourceIdentitySchemasRequest{},
			Check: func(t *testing.T, resp providerops.GetResourceIdentitySchemasResponse) {
				if resp == nil {
					t.Fatal("expected a response, got nil")
				}
				if n := len(maps.Collect(resp.IdentitySchemas())); n != 0 {
					t.Errorf("expected no identity schemas, got %d", n)
				}
				if resp.Diagnostics().HasErrors() {
					t.Error("expected no diagnostics for an unimplemented provider")
				}
			},
		},
	}, func(t *testing.T, provider *tf5.Provider, req *providerops.GetResourceIdentitySchemasRequest) (providerops.GetResourceIdentitySchemasResponse, error) {
		return provider.GetResourceIdentitySchemas(t.Context(), req)
	})
}

// TestUpgradeResourceIdentity covers the identity-side counterpart to
// UpgradeManagedResourceState.
func TestUpgradeResourceIdentity(t *testing.T) {
	raw := identityMsgpack(t)
	testProviderCalls(t, map[string]providerCallTest[*providerops.UpgradeResourceIdentityRequest, providerops.UpgradeResourceIdentityResponse]{
		"upgrades": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.UpgradeResourceIdentity(gomock.Any(), mockutil.Eq(&tfplugin5.UpgradeResourceIdentity_Request{
					TypeName:    "test",
					Version:     1,
					RawIdentity: &tfplugin5.RawState{Json: raw},
				})).Return(&tfplugin5.UpgradeResourceIdentity_Response{
					UpgradedIdentity: &tfplugin5.ResourceIdentityData{
						IdentityData: &tfplugin5.DynamicValue{Msgpack: []byte{0x81, 0xa2, 0x69, 0x64, 0xa6, 0x61, 0x62, 0x63, 0x31, 0x32, 0x33}},
					},
				}, nil)
			},
			Request: &providerops.UpgradeResourceIdentityRequest{
				ResourceType:          "test",
				IdentitySchemaVersion: 1,
				PrevIdentityRaw:       providerschema.RawState{JSON: raw},
			},
			Check: func(t *testing.T, resp providerops.UpgradeResourceIdentityResponse) {
				checkIdentityValue(t, resp.UpgradedIdentity())
			},
		},
		"rejects a missing prior identity": {
			Mock:      func(expect *MockProviderClientMockRecorder) {},
			Request:   &providerops.UpgradeResourceIdentityRequest{ResourceType: "test"},
			WantError: "missing required PrevIdentityRaw value",
		},
	}, func(t *testing.T, provider *tf5.Provider, req *providerops.UpgradeResourceIdentityRequest) (providerops.UpgradeResourceIdentityResponse, error) {
		return provider.UpgradeResourceIdentity(t.Context(), req)
	})
}
