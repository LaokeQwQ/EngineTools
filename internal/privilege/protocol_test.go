package privilege

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"EngineTools/internal/apperror"
)

func validRequest(t *testing.T, action Action, params any) Request {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	req := Request{
		Version:      ProtocolVersion,
		Action:       action,
		Nonce:        strings.Repeat("a", MinNonceBytes*2),
		IssuedAt:     time.Now().UTC(),
		CallerPID:    4242,
		Params:       raw,
		ParamsSHA256: hex.EncodeToString(sum[:]),
	}
	return req
}

func TestValidateRequest(t *testing.T) {
	now := time.Now()

	t.Run("unknown action is denied", func(t *testing.T) {
		req := Request{Version: 1, Action: Action("shell.exec"), Nonce: strings.Repeat("a", 64), IssuedAt: time.Now().UTC()}
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_action_denied" {
			t.Fatal(err)
		}
	})

	t.Run("stale timestamp is rejected", func(t *testing.T) {
		req := validRequest(t, ApplyManifest, InstallParams{InstallID: "engine-default"})
		req.IssuedAt = now.Add(-2 * time.Minute)
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_stale" {
			t.Fatal(err)
		}
	})

	t.Run("future timestamp beyond skew is rejected", func(t *testing.T) {
		req := validRequest(t, ApplyManifest, InstallParams{InstallID: "engine-default"})
		req.IssuedAt = now.Add(5 * time.Minute)
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_stale" {
			t.Fatal(err)
		}
	})

	t.Run("short nonce is rejected", func(t *testing.T) {
		req := validRequest(t, ApplyManifest, InstallParams{InstallID: "engine-default"})
		req.Nonce = strings.Repeat("a", 16)
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_nonce_weak" {
			t.Fatal(err)
		}
	})

	t.Run("digest mismatch is rejected", func(t *testing.T) {
		req := validRequest(t, ApplyManifest, InstallParams{InstallID: "engine-default"})
		req.ParamsSHA256 = strings.Repeat("0", 64)
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_digest_mismatch" {
			t.Fatal(err)
		}
	})

	t.Run("absolute path as installId is rejected", func(t *testing.T) {
		req := validRequest(t, ApplyManifest, InstallParams{InstallID: `C:\Program Files\Engine DJ`})
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_params_invalid" {
			t.Fatal(err)
		}
	})

	t.Run("traversal in installId is rejected", func(t *testing.T) {
		req := validRequest(t, RemoveManifest, InstallParams{InstallID: "..\\..\\windows"})
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_params_invalid" {
			t.Fatal(err)
		}
	})

	t.Run("pid supplied to stop-engine is rejected", func(t *testing.T) {
		req := validRequest(t, StopEngine, map[string]any{"pid": 4242})
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_params_invalid" {
			t.Fatal(err)
		}
	})

	t.Run("wrong version is rejected", func(t *testing.T) {
		req := validRequest(t, StopEngine, NoParams{})
		req.Version = 99
		if err := ValidateRequest(req, now); apperror.CodeOf(err) != "privilege_version_mismatch" {
			t.Fatal(err)
		}
	})

	t.Run("every allowlisted action accepts its own params", func(t *testing.T) {
		cases := []struct {
			action Action
			params any
		}{
			{ApplyManifest, InstallParams{InstallID: "engine-default"}},
			{RemoveManifest, InstallParams{InstallID: "engine-usb-1"}},
			{StopEngine, NoParams{}},
			{CleanEngineMSI, ProductParams{ProductID: "{D95C8B1A-8E5F-4A2F-9B31-2C6E7D8A1101}"}},
			{ReleaseVolume, VolumeParams{VolumeID: "vol-a1b2c3d4-1000"}},
		}
		for _, tc := range cases {
			req := validRequest(t, tc.action, tc.params)
			if err := ValidateRequest(req, now); err != nil {
				t.Fatalf("%s: %v", tc.action, err)
			}
		}
	})
}

func TestNewRequestRoundTripsAndValidates(t *testing.T) {
	req, err := NewRequest(CleanEngineMSI, 1234, ProductParams{ProductID: "{ABC}"})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if req.Version != ProtocolVersion || len(req.Nonce) < MinNonceBytes*2 || req.CallerPID != 1234 {
		t.Fatalf("bad envelope: %+v", req)
	}
	if err := ValidateRequest(req, time.Now()); err != nil {
		t.Fatalf("self validation failed: %v", err)
	}
	// tamper with params after signing -> digest must catch it
	req.Params = json.RawMessage(`{"productCode":"{ABC}"}`)
	if err := ValidateRequest(req, time.Now()); apperror.CodeOf(err) != "privilege_digest_mismatch" {
		t.Fatal(err)
	}
}
