//go:build windows

// Command enginetools-helper is a one-shot elevated worker. It speaks the
// allowlisted protocol from internal/privilege over a private named pipe,
// verifies the requesting app peer-to-peer, performs exactly one action,
// prints one JSON response, and exits. It accepts no paths, PIDs, registry
// keys, or arbitrary commands on its command line -- only an opaque pipe
// name.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"EngineTools/internal/apperror"
	"EngineTools/internal/privilege"
	"golang.org/x/sys/windows"
)

// Exit codes are part of the helper's contract.
const (
	exitOK           = 0
	exitRejected     = 10 // request failed validation
	exitActionFailed = 20 // action ran and reported failure
	exitProtocol     = 30 // pipe/IO/verification problem
)

func main() {
	pipeName := flag.String("pipe", "", "private named pipe to connect to")
	flag.Parse()
	if *pipeName == "" {
		os.Exit(exitProtocol)
	}
	code := run(*pipeName)
	os.Exit(code)
}

func run(pipeName string) int {
	// The helper must be freshly spawned for every transaction.
	self, err := os.Executable()
	if err != nil {
		return exitProtocol
	}

	target, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		return exitProtocol
	}
	handle, err := windows.CreateFile(target,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return exitProtocol
	}
	defer windows.CloseHandle(handle)

	// Verify the requesting app before reading anything from the pipe.
	var serverPID uint32
	if err := windows.GetNamedPipeServerProcessId(handle, &serverPID); err != nil {
		return exitProtocol
	}
	inspector := privilege.NewWindowsInspector()
	sid, session, err := privilege.SelfTokenIdentity()
	if err != nil {
		return exitProtocol
	}
	publisher := ""
	if subject, subjErr := privilege.AuthenticodeSubject(self); subjErr == nil {
		publisher = subject
	}
	expect := privilege.PeerExpectation{
		SID:              sid,
		SessionID:        session,
		InstallRoot:      filepath.Dir(self),
		Publisher:        publisher,
		RequirePublisher: publisher != "",
	}
	if err := privilege.VerifyPeer(inspector, serverPID, expect); err != nil {
		fmt.Fprintln(os.Stderr, "peer verification failed:", err)
		return exitProtocol
	}

	// Exactly one request line in, one response line out.
	reader := bufio.NewReader(newFileReader(handle))
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return exitProtocol
	}
	var req privilege.Request
	if err := json.Unmarshal(line, &req); err != nil {
		writeResponse(handle, privilege.Response{OK: false, Code: "protocol_invalid", MessageKey: "errors.protocolInvalid"})
		return exitRejected
	}
	if err := privilege.ValidateRequest(req, time.Now()); err != nil {
		writeResponse(handle, privilege.Response{OK: false, Code: apperror.CodeOf(err), MessageKey: publicKey(err)})
		return exitRejected
	}

	resp := dispatch(req.Action, req.Params)
	writeResponse(handle, resp)
	if resp.OK {
		return exitOK
	}
	return exitActionFailed
}

func writeResponse(handle windows.Handle, resp privilege.Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	total := 0
	for total < len(data) {
		var written uint32
		if err := windows.WriteFile(handle, data[total:], &written, nil); err != nil {
			return
		}
		total += int(written)
	}
	nl := []byte{'\n'}
	var written uint32
	_ = windows.WriteFile(handle, nl, &written, nil)
}

func publicKey(err error) string {
	return apperror.PublicFrom(err).MessageKey
}

// fileReader adapts the raw pipe HANDLE for bufio.
type fileReader struct{ h windows.Handle }

func newFileReader(h windows.Handle) *fileReader { return &fileReader{h: h} }

func (r *fileReader) Read(p []byte) (int, error) {
	var read uint32
	if err := windows.ReadFile(r.h, p, &read, nil); err != nil {
		return 0, err
	}
	if read == 0 {
		return 0, io.EOF
	}
	return int(read), nil
}
