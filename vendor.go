// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"net/http"

	"github.com/highfivelabs/krm-rest-runtime/rest"
)

// Vendor is how the manager reaches Okta.
var Vendor = &rest.Vendor{
	Name:           "okta",
	DefaultBaseURL: "", // every Connection names its Okta
	CredentialKeys: []string{"apiToken"},
	Authorize:      func(r *http.Request, c map[string]string) { r.Header.Set("Authorization", "SSWS "+c["apiToken"]) },
}
