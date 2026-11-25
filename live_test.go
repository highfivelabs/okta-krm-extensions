// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

//go:build live

package extensions

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/highfivelabs/krm-rest-runtime/kindtest"
)

// The kinds hack/live/live.yaml lists, against the real API: create,
// update, drift repair, delete. Compiled only with -tags live, and skips,
// saying why, unless KRM_LIVE=1 and credentials are found.
// Results go to hack/live/records.json, which sets each kind's readiness.
func TestEveryKindLive(t *testing.T) {
	key, err := generateAPNSKey()
	if err != nil {
		t.Fatalf("generate throwaway APNs key: %v", err)
	}
	t.Setenv("OKTA_LIVE_APNS_KEY", key)
	kindtest.Live(t, Types, Vendor, nil, liveYAML)
}

// generateAPNSKey creates a throwaway ECDSA P-256 PKCS#8 PEM private key for
// Okta PushProvider APNs tests so no real or static key is ever committed.
func generateAPNSKey() (string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}

// Deletes every live-test leftover (KRM_LIVE=1 KRM_LIVE_SWEEP=1).
func TestLiveSweep(t *testing.T) {
	kindtest.LiveSweep(t, Vendor, liveYAML)
}
