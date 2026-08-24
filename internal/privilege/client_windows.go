//go:build windows

package privilege

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"EngineTools/internal/apperror"
	"golang.org/x/sys/windows"
)

// Client drives one elevated-helper transaction: create a private pipe,
// launch the installed helper with runas, verify the peer, exchange one
// protocol request, and read one response.
type Client struct {
	// HelperPath is the installed EngineTools.Helper.exe.
	HelperPath string
	// Inspector verifies the helper process before any bytes move. nil
	// selects the production Windows inspector.
	Inspector PeerInspector
	// Timeout bounds the whole exchange when ctx has no deadline.
	Timeout time.Duration
}

const defaultTimeout = 30 * time.Second

func (c Client) inspector() PeerInspector {
	if c.Inspector != nil {
		return c.Inspector
	}
	return NewWindowsInspector()
}

// Execute performs one privileged action through the helper.
func (c Client) Execute(ctx context.Context, action Action, params any) (Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		timeout := c.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	selfPID := uint32(os.Getpid())
	req, err := NewRequest(action, selfPID, params)
	if err != nil {
		return Response{}, err
	}

	pipeName, err := newPipeName()
	if err != nil {
		return Response{}, err
	}
	handle, err := createPrivatePipe(pipeName)
	if err != nil {
		return Response{}, apperror.E("privilege_pipe_failed", "errors.privilegePipeFailed", true, err)
	}
	defer windows.CloseHandle(handle)

	if err := launchHelperElevated(c.HelperPath, pipeName); err != nil {
		return Response{}, apperror.E("privilege_launch_failed", "errors.privilegeLaunchFailed", false, err)
	}

	// Wait for the elevated helper to attach to our pipe.
	connectErr := make(chan error, 1)
	go func() {
		err := windows.ConnectNamedPipe(handle, nil)
		if err == windows.ERROR_PIPE_CONNECTED {
			err = nil
		}
		connectErr <- err
	}()
	select {
	case err := <-connectErr:
		if err != nil {
			return Response{}, apperror.E("privilege_pipe_failed", "errors.privilegePipeFailed", true, err)
		}
	case <-ctx.Done():
		_ = windows.CancelIoEx(handle, nil)
		<-connectErr
		return Response{}, apperror.E("privilege_timeout", "errors.privilegeTimeout", true, ctx.Err())
	}

	// Verify WHO attached before exchanging anything.
	var peerPID uint32
	if err := windows.GetNamedPipeClientProcessId(handle, &peerPID); err != nil {
		return Response{}, apperror.E("privilege_peer_untrusted", "errors.privilegePeerUntrusted", false, err)
	}
	expect, expErr := selfExpectations(c.HelperPath)
	if expErr != nil {
		return Response{}, apperror.E("privilege_peer_untrusted", "errors.privilegePeerUntrusted", false, expErr)
	}
	if err := VerifyPeer(c.inspector(), peerPID, expect); err != nil {
		return Response{}, err
	}

	// Exchange exactly one request and one response line.
	wire, err := json.Marshal(req)
	if err != nil {
		return Response{}, apperror.E("protocol_io_failed", "errors.protocolIOFailed", false, err)
	}
	writeDone := make(chan error, 1)
	go func() {
		_, werr := writeAll(handle, append(wire, '\n'))
		writeDone <- werr
	}()
	select {
	case werr := <-writeDone:
		if werr != nil {
			return Response{}, apperror.E("protocol_io_failed", "errors.protocolIOFailed", true, werr)
		}
	case <-ctx.Done():
		_ = windows.CancelIoEx(handle, nil)
		<-writeDone
		return Response{}, apperror.E("privilege_timeout", "errors.privilegeTimeout", true, ctx.Err())
	}

	respCh := make(chan lineResult, 1)
	go func() { respCh <- readLine(handle) }()
	var res lineResult
	select {
	case res = <-respCh:
	case <-ctx.Done():
		_ = windows.CancelIoEx(handle, nil)
		res = <-respCh
		return Response{}, apperror.E("privilege_timeout", "errors.privilegeTimeout", true, ctx.Err())
	}
	if res.err != nil {
		return Response{}, apperror.E("protocol_io_failed", "errors.protocolIOFailed", true, res.err)
	}
	var resp Response
	if err := json.Unmarshal(res.line, &resp); err != nil {
		return Response{}, apperror.E("protocol_io_failed", "errors.protocolIOFailed", false, err)
	}
	return resp, nil
}

type lineResult struct {
	line []byte
	err  error
}

func readLine(h windows.Handle) lineResult {
	reader := bufio.NewReader(newHandleReader(h))
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return lineResult{err: err}
	}
	return lineResult{line: line}
}

// expectations describe the peer we are willing to talk to: same user,
// same session, freshly spawned from beside our own executable, signed by
// whoever signed us.
func selfExpectations(helperPath string) (PeerExpectation, error) {
	sid, session, err := queryTokenIdentity(windows.CurrentProcess())
	if err != nil {
		return PeerExpectation{}, err
	}
	root := filepath.Dir(helperPath)
	publisher := ""
	if self, exErr := os.Executable(); exErr == nil {
		publisher, _ = authenticodeSubject(self)
	}
	return PeerExpectation{
		SID:              sid,
		SessionID:        session,
		InstallRoot:      root,
		Publisher:        publisher,
		RequirePublisher: publisher != "",
	}, nil
}

func newPipeName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", apperror.E("privilege_pipe_failed", "errors.privilegePipeFailed", false, err)
	}
	return fmt.Sprintf(`\\.\pipe\enginetools-helper-%s`, hex.EncodeToString(b)), nil
}

// createPrivatePipe opens a duplex named pipe reachable only by SYSTEM and
// the current user.
func createPrivatePipe(name string) (windows.Handle, error) {
	sid, _, err := queryTokenIdentity(windows.CurrentProcess())
	if err != nil {
		return 0, err
	}
	sddl := fmt.Sprintf("D:P(A;;GA;;;SY)(A;;GA;;;%s)", sid)
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return 0, err
	}
	sa := &windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: sd,
	}
	nul16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	handle, err := windows.CreateNamedPipe(nul16,
		windows.PIPE_ACCESS_DUPLEX,
		windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT,
		1, 4096, 4096, 0, sa)
	if err != nil {
		return 0, err
	}
	return handle, nil
}

// launchHelperElevated spawns the installed helper through ShellExecute's
// runas verb. The command line carries ONLY the opaque pipe name -- never
// actions or parameters.
func launchHelperElevated(helperPath, pipeName string) error {
	if _, err := os.Stat(helperPath); err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(helperPath)
	params, _ := windows.UTF16PtrFromString(`--pipe ` + pipeName)
	ret, _, callErr := shellExecuteProc.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		0,
		0, // SW_HIDE
	)
	const seErrNoAssoc = 32
	if ret <= seErrNoAssoc && ret != 0 {
		return fmt.Errorf("ShellExecute runas failed: %v", callErr)
	}
	return nil
}

var (
	shellExecuteProc = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")
)

// handleReader adapts a raw HANDLE for bufio without buffering surprises.
type handleReader struct{ h windows.Handle }

func newHandleReader(h windows.Handle) *handleReader { return &handleReader{h: h} }

func (r *handleReader) Read(p []byte) (int, error) {
	var read uint32
	err := windows.ReadFile(r.h, p, &read, nil)
	if err != nil {
		return 0, err
	}
	if read == 0 {
		return 0, fmt.Errorf("pipe closed")
	}
	return int(read), nil
}

func writeAll(h windows.Handle, data []byte) (int, error) {
	total := 0
	for total < len(data) {
		var written uint32
		if err := windows.WriteFile(h, data[total:], &written, nil); err != nil {
			return total, err
		}
		total += int(written)
	}
	return total, nil
}
