package privilege

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"EngineTools/internal/apperror"
)

// PeerInfo is everything verifiable about the process on the other end of
// the named pipe.
type PeerInfo struct {
	PID          uint32
	ExePath      string
	SID          string
	SessionID    uint32
	CreationTime time.Time
	Publisher    string // Authenticode subject; empty when unsigned/unverified
}

// PeerExpectation is what the local side demands of its peer before any
// protocol bytes are exchanged.
type PeerExpectation struct {
	SID         string // exact user SID, e.g. S-1-5-21-...
	SessionID   uint32
	InstallRoot string // peer image must live under this directory
	Publisher   string // Authenticode subject substring for production builds
	// RequirePublisher relaxes the signature check only for developer
	// builds of the helper; production wiring always sets it true.
	RequirePublisher bool
}

// MaxPeerAge bounds how long ago the peer may have been created. A one-shot
// helper must be freshly spawned; an old process is either recycled or an
// impersonator.
const MaxPeerAge = 2 * time.Minute

// PeerInspector opens a process by PID and reports the fields above.
type PeerInspector interface {
	InspectPeer(pid uint32) (PeerInfo, error)
}

// VerifyPeer checks every expectation against the inspected peer. Any
// mismatch yields privilege_peer_untrusted without leaking which field
// diverged (that detail belongs to local logs, not to a possibly hostile
// counterpart).
func VerifyPeer(insp PeerInspector, pid uint32, expect PeerExpectation) error {
	info, err := insp.InspectPeer(pid)
	if err != nil {
		return apperror.E("privilege_peer_untrusted", "errors.privilegePeerUntrusted", false,
			fmt.Errorf("inspect pid %d: %w", pid, err))
	}
	if info.PID != pid {
		return peerUntrusted("pid identity mismatch")
	}
	if expect.SID != "" && info.SID != expect.SID {
		return peerUntrusted("sid mismatch")
	}
	if info.SessionID != expect.SessionID {
		return peerUntrusted("session mismatch")
	}
	if expect.InstallRoot != "" {
		// Normalize across platforms and case-insensitive filesystems:
		// Windows paths may arrive with either separator and any casing.
		norm := func(p string) string {
			return filepath.Clean(strings.ToLower(strings.ReplaceAll(p, "\\", "/")))
		}
		root := norm(expect.InstallRoot)
		exe := norm(info.ExePath)
		rel, relErr := filepath.Rel(root, exe)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return peerUntrusted("image outside install root")
		}
	}
	age := time.Since(info.CreationTime)
	if age < 0 || age > MaxPeerAge {
		return peerUntrusted("process not freshly created")
	}
	if expect.RequirePublisher && !strings.Contains(strings.ToLower(info.Publisher), strings.ToLower(expect.Publisher)) {
		return peerUntrusted("publisher mismatch")
	}
	return nil
}

func peerUntrusted(why string) error {
	return apperror.E("privilege_peer_untrusted", "errors.privilegePeerUntrusted", false,
		fmt.Errorf("%s", why))
}
