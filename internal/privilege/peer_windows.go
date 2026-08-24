//go:build windows

package privilege

import (
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"time"
	"unsafe"
)

// windowsInspector inspects peers through Win32 process and token APIs.
type windowsInspector struct{}

// NewWindowsInspector returns the production PeerInspector.
func NewWindowsInspector() PeerInspector { return windowsInspector{} }

func (windowsInspector) InspectPeer(pid uint32) (PeerInfo, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return PeerInfo{}, err
	}
	defer windows.CloseHandle(h)

	exePath, err := queryImageName(h)
	if err != nil {
		return PeerInfo{}, err
	}
	sid, session, err := queryTokenIdentity(h)
	if err != nil {
		return PeerInfo{}, err
	}
	created, err := queryCreationTime(h)
	if err != nil {
		return PeerInfo{}, err
	}
	publisher, _ := authenticodeSubject(exePath) // unsigned images simply yield ""

	return PeerInfo{
		PID:          pid,
		ExePath:      exePath,
		SID:          sid,
		SessionID:    session,
		CreationTime: created,
		Publisher:    publisher,
	}, nil
}

func queryImageName(h windows.Handle) (string, error) {
	buf := make([]uint16, 1024)
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return "", fmt.Errorf("QueryFullProcessImageName: %w", err)
	}
	return windows.UTF16ToString(buf[:n]), nil
}

func queryTokenIdentity(h windows.Handle) (sid string, session uint32, err error) {
	var token windows.Token
	if err = windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token); err != nil {
		return "", 0, fmt.Errorf("OpenProcessToken: %w", err)
	}
	defer token.Close()

	user, uerr := token.GetTokenUser()
	if uerr != nil {
		return "", 0, fmt.Errorf("GetTokenUser: %w", uerr)
	}
	sid = user.User.Sid.String()

	var sessionID uint32
	var retLen uint32
	serr := windows.GetTokenInformation(token, windows.TokenSessionId, (*byte)(unsafe.Pointer(&sessionID)),
		uint32(unsafe.Sizeof(sessionID)), &retLen)
	if serr != nil {
		return "", 0, fmt.Errorf("TokenSessionId: %w", serr)
	}
	return sid, sessionID, nil
}

func queryCreationTime(h windows.Handle) (time.Time, error) {
	var create, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &create, &exit, &kernel, &user); err != nil {
		return time.Time{}, fmt.Errorf("GetProcessTimes: %w", err)
	}
	return time.Unix(0, create.Nanoseconds()), nil
}

// authenticodeSubject extracts the subject of the signer certificate from
// the image's Authenticode signature by reading the PE certificate table
// and decoding the PKCS#7 ContentInfo. It reports identity presentation;
// full chain validation against Microsoft roots is layered on top of this
// by the release-signing pipeline.
func authenticodeSubject(imagePath string) (string, error) {
	blob, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	if len(blob) < 0x40+24 || string(blob[:2]) != "MZ" {
		return "", fmt.Errorf("not a PE image")
	}
	peOff := int(uint32(blob[0x3C]) | uint32(blob[0x3D])<<8 | uint32(blob[0x3E])<<16 | uint32(blob[0x3F])<<24)
	if peOff <= 0 || peOff+24+224 > len(blob) {
		return "", fmt.Errorf("PE headers truncated")
	}
	magic := uint16(blob[peOff+24]) | uint16(blob[peOff+25])<<8
	const pe32plus = 0x20b
	ddBase := peOff + 24 + 112 // PE32+ data directories start
	if magic != pe32plus {
		ddBase = peOff + 24 + 96 // PE32
	}
	const certDirIndex = 4
	entry := ddBase + certDirIndex*8
	certOff := uint32(blob[entry]) | uint32(blob[entry+1])<<8 | uint32(blob[entry+2])<<16 | uint32(blob[entry+3])<<24
	certLen := uint32(blob[entry+4]) | uint32(blob[entry+5])<<8 | uint32(blob[entry+6])<<16 | uint32(blob[entry+7])<<24
	if certOff == 0 || certLen == 0 || int(certOff+certLen) > len(blob) {
		return "", fmt.Errorf("image has no certificate table")
	}
	certTable := blob[certOff : certOff+certLen]
	if len(certTable) < 8 {
		return "", fmt.Errorf("certificate entry too small")
	}
	wRevision := uint16(certTable[4]) | uint16(certTable[5])<<8
	wType := uint16(certTable[6]) | uint16(certTable[7])<<8
	if wRevision != 0x0200 || wType != 0x0002 { // WIN_CERT_REVISION_2_0, PKCS_SIGNED_DATA
		return "", fmt.Errorf("unsupported certificate revision/type")
	}
	return pkcs7SignerSubject(certTable[8:])
}

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,tag:0"`
}

type signedDataHead struct {
	Version          int
	DigestAlgorithms asn1.RawValue
	EncapContent     asn1.RawValue
	Certificates     asn1.RawValue `asn1:"explicit,tag:0,optional"`
}

// pkcs7SignerSubject decodes the SignedData certificate set and returns
// the subject of the first (signer) certificate.
func pkcs7SignerSubject(pkcs7 []byte) (string, error) {
	var ci contentInfo
	if _, err := asn1.Unmarshal(pkcs7, &ci); err != nil {
		return "", fmt.Errorf("ContentInfo: %w", err)
	}
	oidSignedData := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	if !ci.ContentType.Equal(oidSignedData) {
		return "", fmt.Errorf("unexpected content type %v", ci.ContentType)
	}
	var sd signedDataHead
	if _, err := asn1.Unmarshal(ci.Content.Bytes, &sd); err != nil {
		return "", fmt.Errorf("SignedData: %w", err)
	}
	if len(sd.Certificates.Bytes) == 0 {
		return "", fmt.Errorf("signature carries no certificates")
	}
	// The set holds X.509 certificate choices; the first element is the
	// signer's certificate.
	var inner asn1.RawValue
	if _, err := asn1.Unmarshal(sd.Certificates.Bytes, &inner); err != nil {
		return "", fmt.Errorf("certificate set: %w", err)
	}
	cert, err := x509.ParseCertificate(inner.FullBytes)
	if err != nil {
		return "", err
	}
	return cert.Subject.String(), nil
}

// SelfTokenIdentity reports the current process user SID and session ID.
func SelfTokenIdentity() (string, uint32, error) {
	return queryTokenIdentity(windows.CurrentProcess())
}

// AuthenticodeSubject extracts the signer subject of a signed image, or
// an empty string when the image is unsigned.
func AuthenticodeSubject(imagePath string) (string, error) {
	return authenticodeSubject(imagePath)
}
