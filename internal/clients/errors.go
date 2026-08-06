package clients

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	hydra "github.com/ory/hydra-client-go/v2"
)

// apiError renders a Hydra API error with the detail the server actually sent.
//
// The SDK's own error message is just the HTTP status line. Its
// formatErrorMessage reflects on fields named "Title" and "Detail", but the
// ErrorOAuth2 model has neither — it carries Error, ErrorDescription,
// ErrorHint and ErrorDebug — so nothing is appended and callers are left with
// bare text like "400 Bad Request". The useful payload is still reachable via
// GenericOpenAPIError's Model() and Body(); this pulls it back out.
//
// Preferring Model() over Body() keeps the message short: Body() is the raw
// JSON, which ends up embedded verbatim in a managed resource's Synced
// condition. Body() is used only when the response did not decode into
// ErrorOAuth2 (e.g. a proxy returning HTML), where raw text beats nothing.
func apiError(op string, err error) error {
	if err == nil {
		return nil
	}

	var genErr *hydra.GenericOpenAPIError
	if !errors.As(err, &genErr) {
		return fmt.Errorf("%s: %w", op, err)
	}

	// %w throughout: the SDK error stays unwrappable, so callers can still
	// errors.As for GenericOpenAPIError to reach the status code or model.
	// Formatting it with %s would flatten the chain for a cosmetically
	// identical string.
	if detail := detailFromModel(genErr.Model()); detail != "" {
		return fmt.Errorf("%s: %w: %s", op, err, detail)
	}
	if detail := detailFromBody(genErr.Body()); detail != "" {
		return fmt.Errorf("%s: %w: %s", op, err, detail)
	}
	return fmt.Errorf("%s: %w", op, err)
}

// detailFromModel formats the decoded ErrorOAuth2 payload, most specific
// field first. ErrorDebug is only populated when Hydra runs in dev mode.
func detailFromModel(model interface{}) string {
	e, ok := model.(hydra.ErrorOAuth2)
	if !ok {
		return ""
	}

	parts := make([]string, 0, 4)
	for _, s := range []*string{e.Error, e.ErrorDescription, e.ErrorHint, e.ErrorDebug} {
		if s != nil && *s != "" {
			parts = append(parts, *s)
		}
	}
	return strings.Join(parts, ": ")
}

// detailFromBody is the fallback for a response that did not decode into
// ErrorOAuth2. It attempts the same fields via a loose decode, then gives up
// and returns the trimmed raw body.
func detailFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var loose struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		ErrorHint        string `json:"error_hint"`
	}
	if err := json.Unmarshal(body, &loose); err == nil {
		parts := make([]string, 0, 3)
		for _, s := range []string{loose.Error, loose.ErrorDescription, loose.ErrorHint} {
			if s != "" {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, ": ")
		}
	}

	return truncate(strings.TrimSpace(string(body)), 512)
}

// truncate bounds an untyped body so it cannot blow up the condition message
// on a managed resource.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "… (truncated)"
}
