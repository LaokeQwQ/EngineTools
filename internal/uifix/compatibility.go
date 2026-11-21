package uifix

import (
	"EngineTools/internal/storage"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// A profile identifies the complete ABI/call-site combination, not a displayed
// version string. Add a profile only after reviewing and testing that build.
type buildProfile struct {
	Name      string
	Hashes    map[string]string
	Callers   []uint32
	Automatic bool
	Cached    bool
}

const maxCacheCallers = 16
const cacheProtocol = 3
const cacheSharedWords = 16 + maxCacheCallers

var supportedBuilds = []buildProfile{
	{
		Name: "Engine DJ 5.1.0 / Qt 6.7.2 (Windows x64)",
		Hashes: map[string]string{
			"Engine DJ.exe":  "68ed3eb6a51cf0db4f0e1f868da71b83e20fca293a2c589693126c0c24414028",
			"Qt6Network.dll": "55733e8c1d6cb87a051f9a3c8cf510d1a59d4fa8a25bbf6eebd8254c60eefdf1",
			"Qt6Core.dll":    "8365fe1700df83c288fbc172113536a77009a8990e3e47ca69d42d3f0a7f9dbd",
		},
		Callers: []uint32{0xd1c766, 0xd1d403, 0xd71ff3, 0xd7438c},
	},
	{
		Name: "Engine DJ 5.3.0 / Qt 6.7.2 (Windows x64)",
		Hashes: map[string]string{
			"Engine DJ.exe":  "a4873db163484bcda37952148291e169b634299cf935d50730e5a1f8c6d06082",
			"Qt6Network.dll": "117efdc1315137794c184b865b569ea3221b8472fd66715b9bb4d6c5bd54f3fa",
			"Qt6Core.dll":    "8aa75c154ade9194051d8e781104d51f7717c5f175392c8a44ef0c8ad3dd62d6",
		},
		Callers: []uint32{0xd72cc6, 0xd73224, 0xd73963, 0xdc80a4, 0xdc8a13, 0xdcadac, 0xfac564},
	},
}

func selectBuild(hashes map[string]string) (*buildProfile, error) {
	for i := range supportedBuilds {
		profile := &supportedBuilds[i]
		match := true
		for name, expected := range profile.Hashes {
			if hashes[name] != expected {
				match = false
				break
			}
		}
		if match {
			return profile, nil
		}
	}
	return nil, fmt.Errorf("Unsupported Engine DJ/Qt build: no verified compatibility profile matches all three files")
}

func matchBuild(directory string) (*buildProfile, error) {
	hashes, err := hashBuildFiles(directory)
	if err != nil {
		return nil, err
	}
	profile, err := selectBuild(hashes)
	if err == nil {
		return profile, nil
	}
	settings, err := storage.ReadSettings()
	if err != nil {
		return nil, err
	}
	if profile := cachedCompatibility(hashes, settings.EngineDJCompatibility); profile != nil {
		return profile, nil
	}
	profile, err = scanAutomaticBuild(directory, hashes)
	if err != nil {
		return nil, fmt.Errorf("Unsupported Engine DJ/Qt layout: automatic detection failed: %w", err)
	}
	return profile, nil
}

func hashBuildFiles(directory string) (map[string]string, error) {
	hashes := make(map[string]string, 3)
	for _, name := range []string{"Engine DJ.exe", "Qt6Core.dll", "Qt6Network.dll"} {
		file, err := os.Open(filepath.Join(directory, name))
		if err != nil {
			return nil, err
		}
		digest := sha256.New()
		_, err = io.Copy(digest, file)
		file.Close()
		if err != nil {
			return nil, err
		}
		hashes[name] = hex.EncodeToString(digest.Sum(nil))
	}
	return hashes, nil
}
