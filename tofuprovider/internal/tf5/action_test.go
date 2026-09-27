package tf5_test

import (
	"io"
	"maps"
	"testing"

	"github.com/zclconf/go-cty/cty"
	gomock "go.uber.org/mock/gomock"
	grpc "google.golang.org/grpc"

	"github.com/opentofu/provider-client/tofuprovider/grpc/tfplugin5"
	"github.com/opentofu/provider-client/tofuprovider/internal/mockutil"
	"github.com/opentofu/provider-client/tofuprovider/internal/tf5"
	"github.com/opentofu/provider-client/tofuprovider/providerops"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// The action operations on the older protocol. tfplugin 5.11 defines the same
// three RPCs and the same messages as 6.10, so these tests mirror the tf6
// ones case for case; the transports must not drift.

// fakeInvokeActionStream is a minimal grpc.ServerStreamingClient that yields a
// fixed sequence of events and then io.EOF. The wrapper only calls Recv; the
// embedded nil ClientStream satisfies the rest of the interface.
type fakeInvokeActionStream struct {
	grpc.ClientStream
	events []*tfplugin5.InvokeAction_Event
	pos    int
}

func (s *fakeInvokeActionStream) Recv() (*tfplugin5.InvokeAction_Event, error) {
	if s.pos >= len(s.events) {
		return nil, io.EOF
	}
	ev := s.events[s.pos]
	s.pos++
	return ev, nil
}

func progressEvent(msg string) *tfplugin5.InvokeAction_Event {
	return &tfplugin5.InvokeAction_Event{
		Type: &tfplugin5.InvokeAction_Event_Progress_{
			Progress: &tfplugin5.InvokeAction_Event_Progress{Message: msg},
		},
	}
}

func completedEvent(diags ...*tfplugin5.Diagnostic) *tfplugin5.InvokeAction_Event {
	return &tfplugin5.InvokeAction_Event{
		Type: &tfplugin5.InvokeAction_Event_Completed_{
			Completed: &tfplugin5.InvokeAction_Event_Completed{Diagnostics: diags},
		},
	}
}

func TestInvokeAction(t *testing.T) {
	testProviderCalls(t,
		map[string]providerCallTest[*providerops.InvokeActionRequest, providerops.InvokeActionResponse]{
			"progress then completed": {
				Mock: func(expect *MockProviderClientMockRecorder) {
					expect.InvokeAction(
						mockutil.AnyContext(),
						mockutil.Eq(&tfplugin5.InvokeAction_Request{
							ActionType:         "do_action",
							Config:             &tfplugin5.DynamicValue{Msgpack: mockutil.MsgPack("cfg")},
							ClientCapabilities: &tfplugin5.ClientCapabilities{DeferralAllowed: true},
						}),
					).Return(
						&fakeInvokeActionStream{events: []*tfplugin5.InvokeAction_Event{
							progressEvent("step 1"),
							progressEvent("step 2"),
							completedEvent(),
						}}, nil,
					)
				},
				Request: &providerops.InvokeActionRequest{
					ActionType:         "do_action",
					Config:             providerschema.NewDynamicValue(cty.StringVal("cfg"), cty.String),
					ClientCapabilities: &providerops.ClientCapabilities{SupportsDeferral: true},
				},
				Check: func(t *testing.T, resp providerops.InvokeActionResponse) {
					var progresses []string
					completedSeen := false
					for {
						ev, err := resp.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							t.Fatalf("unexpected Recv error: %s", err)
						}
						if msg, ok := ev.Progress(); ok {
							progresses = append(progresses, msg)
							continue
						}
						if diags, ok := ev.Completed(); ok {
							completedSeen = true
							if diags.HasErrors() {
								t.Errorf("unexpected error diagnostics in completion")
							}
							continue
						}
						t.Errorf("event is neither progress nor completed")
					}
					if want := []string{"step 1", "step 2"}; len(progresses) != len(want) ||
						progresses[0] != want[0] || progresses[1] != want[1] {
						t.Errorf("wrong progress messages\ngot:  %v\nwant: %v", progresses, want)
					}
					if !completedSeen {
						t.Errorf("never saw completion event")
					}
				},
			},
			"completed with errors": {
				Mock: func(expect *MockProviderClientMockRecorder) {
					expect.InvokeAction(mockutil.AnyContext(), gomock.Any()).Return(
						&fakeInvokeActionStream{events: []*tfplugin5.InvokeAction_Event{
							completedEvent(&tfplugin5.Diagnostic{
								Severity: tfplugin5.Diagnostic_ERROR,
								Summary:  "action invocation failed",
							}),
						}}, nil,
					)
				},
				Request: &providerops.InvokeActionRequest{
					ActionType: "do_action",
					Config:     providerschema.NewDynamicValue(cty.StringVal("cfg"), cty.String),
				},
				Check: func(t *testing.T, resp providerops.InvokeActionResponse) {
					ev, err := resp.Recv()
					if err != nil {
						t.Fatalf("unexpected Recv error: %s", err)
					}
					diags, ok := ev.Completed()
					if !ok {
						t.Fatal("first event is not the completion")
					}
					if !diags.HasErrors() {
						t.Error("completion lost its error diagnostics")
					}
					if _, err := resp.Recv(); err != io.EOF {
						t.Errorf("stream did not end after completion: %v", err)
					}
				},
			},
		},
		func(t *testing.T, provider *tf5.Provider, req *providerops.InvokeActionRequest) (providerops.InvokeActionResponse, error) {
			return provider.InvokeAction(t.Context(), req)
		},
	)
}

func TestValidateActionConfig(t *testing.T) {
	testProviderCalls(t,
		map[string]providerCallTest[*providerops.ValidateActionConfigRequest, providerops.ValidateActionConfigResponse]{
			"clean": {
				Mock: func(expect *MockProviderClientMockRecorder) {
					expect.ValidateActionConfig(
						mockutil.AnyContext(),
						mockutil.Eq(&tfplugin5.ValidateActionConfig_Request{
							ActionType: "do_action",
							Config:     &tfplugin5.DynamicValue{Msgpack: mockutil.MsgPack("cfg")},
						}),
					).Return(&tfplugin5.ValidateActionConfig_Response{}, nil)
				},
				Request: &providerops.ValidateActionConfigRequest{
					ActionType: "do_action",
					Config:     providerschema.NewDynamicValue(cty.StringVal("cfg"), cty.String),
				},
				Check: func(t *testing.T, resp providerops.ValidateActionConfigResponse) {
					mockutil.AssertNoDiags(t, resp.Diagnostics())
				},
			},
			"diagnostics": {
				Mock: func(expect *MockProviderClientMockRecorder) {
					expect.ValidateActionConfig(mockutil.AnyContext(), gomock.Any()).Return(
						&tfplugin5.ValidateActionConfig_Response{
							Diagnostics: []*tfplugin5.Diagnostic{{
								Severity: tfplugin5.Diagnostic_ERROR,
								Summary:  "Invalid action configuration",
								Detail:   "The provider disagrees.",
							}},
						}, nil)
				},
				Request: &providerops.ValidateActionConfigRequest{
					ActionType: "do_action",
					Config:     providerschema.NewDynamicValue(cty.StringVal("cfg"), cty.String),
				},
				Check: func(t *testing.T, resp providerops.ValidateActionConfigResponse) {
					if !resp.Diagnostics().HasErrors() {
						t.Error("expected error diagnostics")
					}
				},
			},
		},
		func(t *testing.T, provider *tf5.Provider, req *providerops.ValidateActionConfigRequest) (providerops.ValidateActionConfigResponse, error) {
			return provider.ValidateActionConfig(t.Context(), req)
		},
	)
}

func TestPlanAction(t *testing.T) {
	testProviderCalls(t,
		map[string]providerCallTest[*providerops.PlanActionRequest, providerops.PlanActionResponse]{
			"invocable": {
				Mock: func(expect *MockProviderClientMockRecorder) {
					expect.PlanAction(
						mockutil.AnyContext(),
						mockutil.Eq(&tfplugin5.PlanAction_Request{
							ActionType:         "do_action",
							Config:             &tfplugin5.DynamicValue{Msgpack: mockutil.MsgPack("cfg")},
							ClientCapabilities: &tfplugin5.ClientCapabilities{DeferralAllowed: true},
						}),
					).Return(&tfplugin5.PlanAction_Response{}, nil)
				},
				Request: &providerops.PlanActionRequest{
					ActionType:         "do_action",
					Config:             providerschema.NewDynamicValue(cty.StringVal("cfg"), cty.String),
					ClientCapabilities: &providerops.ClientCapabilities{SupportsDeferral: true},
				},
				Check: func(t *testing.T, resp providerops.PlanActionResponse) {
					mockutil.AssertNoDiags(t, resp.Diagnostics())
					if resp.Deferred() != nil {
						t.Errorf("unexpected deferral: %v", resp.Deferred().Reason())
					}
				},
			},
			"deferred": {
				Mock: func(expect *MockProviderClientMockRecorder) {
					expect.PlanAction(mockutil.AnyContext(), gomock.Any()).Return(
						&tfplugin5.PlanAction_Response{
							Deferred: &tfplugin5.Deferred{Reason: tfplugin5.Deferred_PROVIDER_CONFIG_UNKNOWN},
						}, nil)
				},
				Request: &providerops.PlanActionRequest{
					ActionType:         "do_action",
					Config:             providerschema.NewDynamicValue(cty.StringVal("cfg"), cty.String),
					ClientCapabilities: &providerops.ClientCapabilities{SupportsDeferral: true},
				},
				Check: func(t *testing.T, resp providerops.PlanActionResponse) {
					mockutil.AssertNoDiags(t, resp.Diagnostics())
					def := resp.Deferred()
					if def == nil {
						t.Fatal("expected a deferral")
					}
					if got, want := def.Reason(), providerops.DeferredBecauseProviderConfigUnknown; got != want {
						t.Errorf("wrong deferral reason\ngot:  %v\nwant: %v", got, want)
					}
				},
			},
			"diagnostics": {
				Mock: func(expect *MockProviderClientMockRecorder) {
					expect.PlanAction(mockutil.AnyContext(), gomock.Any()).Return(
						&tfplugin5.PlanAction_Response{
							Diagnostics: []*tfplugin5.Diagnostic{{
								Severity: tfplugin5.Diagnostic_ERROR,
								Summary:  "Cannot plan action",
								Detail:   "The provider disagrees.",
							}},
						}, nil)
				},
				Request: &providerops.PlanActionRequest{
					ActionType: "do_action",
					Config:     providerschema.NewDynamicValue(cty.StringVal("cfg"), cty.String),
				},
				Check: func(t *testing.T, resp providerops.PlanActionResponse) {
					if !resp.Diagnostics().HasErrors() {
						t.Error("expected error diagnostics")
					}
				},
			},
		},
		func(t *testing.T, provider *tf5.Provider, req *providerops.PlanActionRequest) (providerops.PlanActionResponse, error) {
			return provider.PlanAction(t.Context(), req)
		},
	)
}

// TestGetProviderSchemaActions pins that a 5.11 provider's action schemas are
// read from the response rather than dropped, which is what the transport did
// before it implemented actions.
func TestGetProviderSchemaActions(t *testing.T) {
	testProviderCalls(t,
		map[string]providerCallTest[*providerops.GetProviderSchemaRequest, providerops.GetProviderSchemaResponse]{
			"action schemas": {
				Mock: func(expect *MockProviderClientMockRecorder) {
					expect.GetSchema(
						mockutil.AnyContext(),
						mockutil.Eq(&tfplugin5.GetProviderSchema_Request{}),
					).Return(
						&tfplugin5.GetProviderSchema_Response{
							ActionSchemas: map[string]*tfplugin5.ActionSchema{
								"do_action": {
									Schema: &tfplugin5.Schema{
										Block: &tfplugin5.Schema_Block{
											Attributes: []*tfplugin5.Schema_Attribute{
												{Name: "name", Type: []byte(`"string"`), Required: true},
											},
										},
									},
								},
							},
						}, nil,
					)
				},
				Request: &providerops.GetProviderSchemaRequest{},
				Check: func(t *testing.T, resp providerops.GetProviderSchemaResponse) {
					mockutil.AssertNoDiags(t, resp.Diagnostics())
					schemas := maps.Collect(resp.ProviderSchema().ActionSchemas())
					s, ok := schemas["do_action"]
					if !ok {
						t.Fatalf("action schemas lack do_action: %v", slicesOfKeys(schemas))
					}
					attrs := maps.Collect(s.Attributes())
					attr, ok := attrs["name"]
					if !ok {
						t.Fatalf("do_action's schema lacks the name attribute")
					}
					if got, want := attr.Usage(), providerschema.AttributeRequired; got != want {
						t.Errorf("wrong usage for name\ngot:  %v\nwant: %v", got, want)
					}
				},
			},
		},
		func(t *testing.T, provider *tf5.Provider, req *providerops.GetProviderSchemaRequest) (providerops.GetProviderSchemaResponse, error) {
			return provider.GetProviderSchema(t.Context(), req)
		},
	)
}

func slicesOfKeys[V any](m map[string]V) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
