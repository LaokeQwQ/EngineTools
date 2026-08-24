package privilege

import (
	"errors"
	"testing"
	"time"

	"EngineTools/internal/apperror"
)

type fakeInspector struct {
	info PeerInfo
	err  error
}

func (f fakeInspector) InspectPeer(pid uint32) (PeerInfo, error) {
	if f.err != nil {
		return PeerInfo{}, f.err
	}
	return f.info, nil
}

func basePeer(pid uint32) PeerInfo {
	return PeerInfo{
		PID:          pid,
		ExePath:      `C:\\Program Files\\EngineTools\\EngineTools.Helper.exe`,
		SID:          "S-1-5-21-1004336348-1177238915-682003330-1001",
		SessionID:    1,
		CreationTime: time.Now().Add(-5 * time.Second),
		Publisher:    "CN=Example Software Co., O=Example Software Co.",
	}
}

func baseExpectation() PeerExpectation {
	return PeerExpectation{
		SID:              "S-1-5-21-1004336348-1177238915-682003330-1001",
		SessionID:        1,
		InstallRoot:      `C:\\Program Files\\EngineTools`,
		Publisher:        "example software co",
		RequirePublisher: true,
	}
}

func TestVerifyPeer(t *testing.T) {
	const pid = uint32(4711)

	t.Run("matching peer succeeds", func(t *testing.T) {
		if err := VerifyPeer(fakeInspector{info: basePeer(pid)}, pid, baseExpectation()); err != nil {
			t.Fatalf("matching peer rejected: %v", err)
		}
	})

	t.Run("inspect failure is untrusted", func(t *testing.T) {
		insp := fakeInspector{err: contextDead}
		if err := VerifyPeer(insp, pid, baseExpectation()); appErrorCode(err) != "privilege_peer_untrusted" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("wrong SID is untrusted", func(t *testing.T) {
		peer := basePeer(pid)
		peer.SID = "S-1-5-21-999-999-999-999"
		if err := VerifyPeer(fakeInspector{info: peer}, pid, baseExpectation()); appErrorCode(err) != "privilege_peer_untrusted" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("wrong session is untrusted", func(t *testing.T) {
		peer := basePeer(pid)
		peer.SessionID = 2
		if err := VerifyPeer(fakeInspector{info: peer}, pid, baseExpectation()); appErrorCode(err) != "privilege_peer_untrusted" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("stale creation time is untrusted", func(t *testing.T) {
		peer := basePeer(pid)
		peer.CreationTime = time.Now().Add(-time.Hour)
		if err := VerifyPeer(fakeInspector{info: peer}, pid, baseExpectation()); appErrorCode(err) != "privilege_peer_untrusted" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("future creation time is untrusted", func(t *testing.T) {
		peer := basePeer(pid)
		peer.CreationTime = time.Now().Add(5 * time.Minute)
		if err := VerifyPeer(fakeInspector{info: peer}, pid, baseExpectation()); appErrorCode(err) != "privilege_peer_untrusted" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("image outside install root is untrusted", func(t *testing.T) {
		peer := basePeer(pid)
		peer.ExePath = `C:\\Windows\\Temp\\EngineTools.Helper.exe`
		if err := VerifyPeer(fakeInspector{info: peer}, pid, baseExpectation()); appErrorCode(err) != "privilege_peer_untrusted" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("wrong publisher is untrusted when required", func(t *testing.T) {
		peer := basePeer(pid)
		peer.Publisher = "CN=Evil Corp"
		if err := VerifyPeer(fakeInspector{info: peer}, pid, baseExpectation()); appErrorCode(err) != "privilege_peer_untrusted" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("unsigned peer passes only when publisher not required", func(t *testing.T) {
		peer := basePeer(pid)
		peer.Publisher = ""
		expect := baseExpectation()
		expect.RequirePublisher = false
		if err := VerifyPeer(fakeInspector{info: peer}, pid, expect); err != nil {
			t.Fatalf("dev-mode unsigned peer rejected: %v", err)
		}
	})

	t.Run("pid identity mismatch is untrusted", func(t *testing.T) {
		peer := basePeer(pid)
		peer.PID = pid + 1
		if err := VerifyPeer(fakeInspector{info: peer}, pid, baseExpectation()); appErrorCode(err) != "privilege_peer_untrusted" {
			t.Fatalf("err = %v", err)
		}
	})
}

var contextDead = errors.New("process is gone")

func appErrorCode(err error) string { return apperror.CodeOf(err) }
