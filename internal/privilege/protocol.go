// Package privilege defines the allowlisted request/response protocol
// spoken between the normal-user Wails app and the one-shot elevated
// helper. Only fixed actions with tightly validated parameters cross the
// boundary; arbitrary commands, paths, PIDs, and registry keys cannot be
// expressed.
package privilege

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"EngineTools/internal/apperror"
)

// ProtocolVersion is bumped on any wire-incompatible change; the helper
// refuses everything else.
const ProtocolVersion = 1

// MaxClockSkew bounds how far IssuedAt may drift from the helper's clock.
const MaxClockSkew = 60 * time.Second

// MinNonceBytes is the minimum decoded nonce entropy; the client sends it
// hex-encoded so the string form is twice this length.
const MinNonceBytes = 32

// Action enumerates every privileged operation the helper will ever run.
type Action string

const (
	ApplyManifest  Action = "manifest.apply"
	RemoveManifest Action = "manifest.remove"
	StopEngine     Action = "process.stop-engine"
	CleanEngineMSI Action = "msi.clean-engine"
	ReleaseVolume  Action = "usb.release-volume"
)

func (a Action) valid() bool {
	switch a {
	case ApplyManifest, RemoveManifest, StopEngine, CleanEngineMSI, ReleaseVolume:
		return true
	}
	return false
}

// Request is the signed envelope the helper validates before doing work.
type Request struct {
	Version      int             `json:"version"`
	Action       Action          `json:"action"`
	Nonce        string          `json:"nonce"`
	IssuedAt     time.Time       `json:"issuedAt"`
	CallerPID    uint32          `json:"callerPid"`
	Params       json.RawMessage `json:"params"`
	ParamsSHA256 string          `json:"paramsSha256"`
}

// Response is the helper's single-line verdict.
type Response struct {
	OK         bool   `json:"ok"`
	Code       string `json:"code,omitempty"`
	MessageKey string `json:"messageKey,omitempty"`
}

// NewRequest builds a fresh envelope for one action, marshaling and
// hashing its typed parameters.
func NewRequest(action Action, callerPID uint32, params any) (Request, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return Request{}, apperror.E("privilege_request_build", "errors.privilegeRequestBuild", false, err)
	}
	nonce := make([]byte, MinNonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		return Request{}, apperror.E("privilege_request_build", "errors.privilegeRequestBuild", false, err)
	}
	req := Request{
		Version:      ProtocolVersion,
		Action:       action,
		Nonce:        hex.EncodeToString(nonce),
		IssuedAt:     time.Now().UTC(),
		CallerPID:    callerPID,
		Params:       raw,
		ParamsSHA256: hashParams(raw),
	}
	if err := ValidateRequest(req, time.Now()); err != nil {
		return Request{}, err
	}
	return req, nil
}

func hashParams(raw json.RawMessage) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// ValidateRequest enforces, in order: known protocol version, allowlisted
// action, strong nonce, timestamp freshness, constant-time parameter
// digest match, and per-action parameter shape. Every rejection is a
// stable public code; causes never carry user-controlled text.
func ValidateRequest(req Request, now time.Time) error {
	if req.Version != ProtocolVersion {
		return apperror.E("privilege_version_mismatch", "errors.privilegeVersionMismatch", false,
			fmt.Errorf("protocol version %d", req.Version))
	}
	if !req.Action.valid() {
		return apperror.E("privilege_action_denied", "errors.privilegeActionDenied", false,
			fmt.Errorf("action %q is not allowlisted", string(req.Action)))
	}
	if len(req.Nonce) < MinNonceBytes*2 {
		return apperror.E("privilege_nonce_weak", "errors.privilegeNonceWeak", false,
			fmt.Errorf("nonce is %d chars, want >= %d", len(req.Nonce), MinNonceBytes*2))
	}
	skew := now.Sub(req.IssuedAt)
	if skew < -MaxClockSkew || skew > MaxClockSkew {
		return apperror.E("privilege_stale", "errors.privilegeStale", false,
			fmt.Errorf("issued %s ago", skew))
	}
	want := hashParams(req.Params)
	if subtle.ConstantTimeCompare([]byte(want), []byte(req.ParamsSHA256)) != 1 {
		return apperror.E("privilege_digest_mismatch", "errors.privilegeDigestMismatch", false,
			fmt.Errorf("params digest mismatch"))
	}
	if err := validateParamsFor(req.Action, req.Params); err != nil {
		return err
	}
	return nil
}

// validateParamsFor decodes params strictly against the per-action shape
// and applies its field rules. Unknown JSON fields are rejected so a
// compromised client cannot smuggle extra intent.
func validateParamsFor(action Action, raw json.RawMessage) error {
	switch action {
	case ApplyManifest, RemoveManifest:
		var p InstallParams
		if err := strictUnmarshal(raw, &p); err != nil {
			return err
		}
		if err := validateIdentifier("installId", p.InstallID); err != nil {
			return err
		}
	case StopEngine:
		var p NoParams
		if err := strictUnmarshal(raw, &p); err != nil {
			return err
		}
	case CleanEngineMSI:
		var p ProductParams
		if err := strictUnmarshal(raw, &p); err != nil {
			return err
		}
		if err := validateIdentifier("productId", p.ProductID); err != nil {
			return err
		}
	case ReleaseVolume:
		var p VolumeParams
		if err := strictUnmarshal(raw, &p); err != nil {
			return err
		}
		if err := validateIdentifier("volumeId", p.VolumeID); err != nil {
			return err
		}
	default:
		return apperror.E("privilege_action_denied", "errors.privilegeActionDenied", false, nil)
	}
	return nil
}

func strictUnmarshal(raw json.RawMessage, dst any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperror.E("privilege_params_invalid", "errors.privilegeParamsInvalid", false, err)
	}
	return nil
}
