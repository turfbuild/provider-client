package tf5_test

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
	"go.uber.org/mock/gomock"

	"github.com/opentofu/provider-client/tofuprovider/grpc/tfplugin5"
	"github.com/opentofu/provider-client/tofuprovider/internal/mockutil"
	"github.com/opentofu/provider-client/tofuprovider/internal/tf5"
	"github.com/opentofu/provider-client/tofuprovider/providerops"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// TestUpgradeManagedResourceState: the stored object travels raw, at the
// version it was saved under, and the provider's upgraded object comes back
// decodable against the current schema.
func TestUpgradeManagedResourceState(t *testing.T) {
	raw := []byte(`{"id":"abc123","dropped":true}`)
	testProviderCalls(t, map[string]providerCallTest[*providerops.UpgradeManagedResourceStateRequest, providerops.UpgradeManagedResourceStateResponse]{
		"upgrades": {
			Mock: func(expect *MockProviderClientMockRecorder) {
				expect.UpgradeResourceState(gomock.Any(), mockutil.Eq(&tfplugin5.UpgradeResourceState_Request{
					TypeName: "test",
					Version:  1,
					RawState: &tfplugin5.RawState{Json: raw},
				})).Return(&tfplugin5.UpgradeResourceState_Response{
					UpgradedState: &tfplugin5.DynamicValue{Msgpack: []byte{0x81, 0xa2, 0x69, 0x64, 0xa6, 0x61, 0x62, 0x63, 0x31, 0x32, 0x33}},
				}, nil)
			},
			Request: &providerops.UpgradeManagedResourceStateRequest{
				ResourceType:  "test",
				SchemaVersion: 1,
				PrevStateRaw:  providerschema.RawState{JSON: raw},
			},
			Check: func(t *testing.T, resp providerops.UpgradeManagedResourceStateResponse) {
				got := resp.UpgradedState()
				if got == nil {
					t.Fatal("upgraded state is nil, want a value")
				}
				v, err := got.AsCtyValue(cty.Object(map[string]cty.Type{"id": cty.String}))
				if err != nil {
					t.Fatalf("decoding the upgraded state: %s", err)
				}
				if want := cty.StringVal("abc123"); !v.GetAttr("id").RawEquals(want) {
					t.Errorf("wrong upgraded id\ngot:  %#v\nwant: %#v", v.GetAttr("id"), want)
				}
			},
		},
		"rejects a missing prior state": {
			Mock:      func(expect *MockProviderClientMockRecorder) {},
			Request:   &providerops.UpgradeManagedResourceStateRequest{ResourceType: "test"},
			WantError: "missing required PrevStateRaw value",
		},
	}, func(t *testing.T, provider *tf5.Provider, req *providerops.UpgradeManagedResourceStateRequest) (providerops.UpgradeManagedResourceStateResponse, error) {
		return provider.UpgradeManagedResourceState(t.Context(), req)
	})
}
