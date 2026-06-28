package tf6_test

import (
	"io"
	"testing"

	"github.com/zclconf/go-cty/cty"
	grpc "google.golang.org/grpc"

	"github.com/opentofu/provider-client/tofuprovider/grpc/tfplugin6"
	"github.com/opentofu/provider-client/tofuprovider/internal/mockutil"
	"github.com/opentofu/provider-client/tofuprovider/internal/tf6"
	"github.com/opentofu/provider-client/tofuprovider/providerops"
	"github.com/opentofu/provider-client/tofuprovider/providerschema"
)

// fakeInvokeActionStream is a minimal grpc.ServerStreamingClient that yields a
// fixed sequence of events and then io.EOF. The wrapper only calls Recv; the
// embedded nil ClientStream satisfies the rest of the interface.
type fakeInvokeActionStream struct {
	grpc.ClientStream
	events []*tfplugin6.InvokeAction_Event
	pos    int
}

func (s *fakeInvokeActionStream) Recv() (*tfplugin6.InvokeAction_Event, error) {
	if s.pos >= len(s.events) {
		return nil, io.EOF
	}
	ev := s.events[s.pos]
	s.pos++
	return ev, nil
}

func progressEvent(msg string) *tfplugin6.InvokeAction_Event {
	return &tfplugin6.InvokeAction_Event{
		Type: &tfplugin6.InvokeAction_Event_Progress_{
			Progress: &tfplugin6.InvokeAction_Event_Progress{Message: msg},
		},
	}
}

func completedEvent(diags ...*tfplugin6.Diagnostic) *tfplugin6.InvokeAction_Event {
	return &tfplugin6.InvokeAction_Event{
		Type: &tfplugin6.InvokeAction_Event_Completed_{
			Completed: &tfplugin6.InvokeAction_Event_Completed{Diagnostics: diags},
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
						mockutil.Eq(&tfplugin6.InvokeAction_Request{
							ActionType: "do_action",
							Config:     &tfplugin6.DynamicValue{Msgpack: mockutil.MsgPack("cfg")},
						}),
					).Return(
						&fakeInvokeActionStream{events: []*tfplugin6.InvokeAction_Event{
							progressEvent("step 1"),
							progressEvent("step 2"),
							completedEvent(),
						}}, nil,
					)
				},
				Request: &providerops.InvokeActionRequest{
					ActionType: "do_action",
					Config:     providerschema.NewDynamicValue(cty.StringVal("cfg"), cty.String),
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
		},
		func(t *testing.T, provider *tf6.Provider, req *providerops.InvokeActionRequest) (providerops.InvokeActionResponse, error) {
			return provider.InvokeAction(t.Context(), req)
		},
	)
}
