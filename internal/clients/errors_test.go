package clients

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	hydra "github.com/ory/hydra-client-go/v2"
)

func ptr(s string) *string { return &s }

// newServer stands up a Hydra-shaped endpoint that answers every request with
// the given status and body.
func newServer(t *testing.T, status int, contentType, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// The regression this file exists for. GenericOpenAPIError's own Error() is
// just the status line, because the SDK's formatErrorMessage reflects on
// "Title"/"Detail" fields that ErrorOAuth2 does not have. Without unpacking
// the model, a failing reconcile reports nothing but "400 Bad Request" on the
// managed resource's Synced condition.
//
// Driving a real server through the client covers the whole chain, including
// the SDK internals this depends on — a future SDK bump that changes the error
// shape fails here rather than silently degrading diagnostics in the field.
func TestAPIError_SurfacesHydraDetail(t *testing.T) {
	const body = `{
		"error": "invalid_request",
		"error_description": "The redirect_uri is not valid",
		"error_hint": "Add the URI to the client's allowed redirect URIs"
	}`
	srv := newServer(t, http.StatusBadRequest, "application/json", body)
	c := NewHydraClient(srv.URL, "")

	_, err := c.UpdateOAuth2Client(context.Background(), "some-client", hydra.OAuth2Client{})
	if err == nil {
		t.Fatal("expected an error from a 400 response")
	}

	got := err.Error()
	for _, want := range []string{
		`failed to update OAuth2 client "some-client"`,
		"invalid_request",
		"The redirect_uri is not valid",
		"Add the URI to the client's allowed redirect URIs",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("error is missing %q\ngot: %s", want, got)
		}
	}
}

// The enriched error must stay unwrappable, so callers can still reach the
// status code or decoded model via errors.As. Formatting the SDK error with
// %s instead of %w yields an identical string while quietly severing the
// chain, which is easy to do and invisible without this test.
func TestAPIError_PreservesErrorChain(t *testing.T) {
	srv := newServer(t, http.StatusConflict, "application/json",
		`{"error":"conflict","error_description":"already exists"}`)
	c := NewHydraClient(srv.URL, "")

	_, err := c.CreateOAuth2Client(context.Background(), hydra.OAuth2Client{})
	if err == nil {
		t.Fatal("expected an error from a 409 response")
	}

	var apiErr *hydra.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("GenericOpenAPIError is no longer reachable via errors.As: %v", err)
	}
	if len(apiErr.Body()) == 0 {
		t.Error("unwrapped error carries no body")
	}
}

// A response that is not an ErrorOAuth2 payload — a proxy or ingress erroring
// before the request reaches Hydra — should still yield something readable
// rather than a bare status line.
func TestAPIError_FallsBackToBody(t *testing.T) {
	srv := newServer(t, http.StatusBadGateway, "text/html", "<html>502 upstream unavailable</html>")
	c := NewHydraClient(srv.URL, "")

	_, err := c.GetOAuth2Client(context.Background(), "some-client")
	if err == nil {
		t.Fatal("expected an error from a 502 response")
	}
	if !strings.Contains(err.Error(), "upstream unavailable") {
		t.Errorf("error should carry the raw body, got: %s", err.Error())
	}
}

// 404 stays a non-error on the read and delete paths — absence is how
// Crossplane learns the external resource needs creating, and delete is
// idempotent.
func TestNotFoundIsNotAnError(t *testing.T) {
	srv := newServer(t, http.StatusNotFound, "application/json", `{"error":"Not Found"}`)
	c := NewHydraClient(srv.URL, "")

	client, err := c.GetOAuth2Client(context.Background(), "missing")
	if err != nil {
		t.Errorf("Get on 404: unexpected error: %v", err)
	}
	if client != nil {
		t.Errorf("Get on 404: expected nil client, got %+v", client)
	}

	if err := c.DeleteOAuth2Client(context.Background(), "missing"); err != nil {
		t.Errorf("Delete on 404: unexpected error: %v", err)
	}
}

// Transport-level failures carry no GenericOpenAPIError, so the original error
// must pass through unwrapped rather than being swallowed.
func TestAPIError_NonAPIErrorPassesThrough(t *testing.T) {
	c := NewHydraClient("http://127.0.0.1:1", "")

	_, err := c.CreateOAuth2Client(context.Background(), hydra.OAuth2Client{})
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if !strings.Contains(err.Error(), "failed to create OAuth2 client") {
		t.Errorf("error lost its operation context: %s", err.Error())
	}
}

func TestDetailFromModel(t *testing.T) {
	for _, tc := range []struct {
		name  string
		model interface{}
		want  string
	}{
		{
			name:  "all fields joined most-specific-last",
			model: hydra.ErrorOAuth2{Error: ptr("invalid_request"), ErrorDescription: ptr("bad thing"), ErrorHint: ptr("try this")},
			want:  "invalid_request: bad thing: try this",
		},
		{
			name:  "empty strings skipped",
			model: hydra.ErrorOAuth2{Error: ptr("invalid_request"), ErrorDescription: ptr("")},
			want:  "invalid_request",
		},
		{
			name:  "not an ErrorOAuth2",
			model: struct{ Other string }{Other: "x"},
			want:  "",
		},
		{
			name:  "nil model",
			model: nil,
			want:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := detailFromModel(tc.model); got != tc.want {
				t.Errorf("detailFromModel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDetailFromBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"json error envelope", `{"error":"x","error_description":"y"}`, "x: y"},
		{"json without known fields", `{"unrelated":"z"}`, `{"unrelated":"z"}`},
		{"non-json", "boom", "boom"},
		{"empty", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := detailFromBody([]byte(tc.body)); got != tc.want {
				t.Errorf("detailFromBody() = %q, want %q", got, tc.want)
			}
		})
	}
}

// A huge untyped body must not blow up the condition message it ends up in.
func TestDetailFromBody_Truncates(t *testing.T) {
	got := detailFromBody([]byte(strings.Repeat("x", 2000)))
	if len(got) > 600 {
		t.Errorf("body not truncated, length %d", len(got))
	}
	if !strings.HasSuffix(got, "(truncated)") {
		t.Errorf("truncated body should say so, got suffix: %q", got[max(0, len(got)-20):])
	}
}
