// Package api exposes the hand-maintained OpenAPI contract.
//
// The spec at openapi.yaml is the source of truth for the HTTP API and is
// embedded into the binary so it can be served as live documentation without
// shipping the file separately. Keep it in sync with the handlers by hand.
package api

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
