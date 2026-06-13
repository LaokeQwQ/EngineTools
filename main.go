package main

import (
	"embed"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsWindows "github.com/wailsapp/wails/v2/pkg/options/windows"
	"golang.org/x/sys/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func isAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	var token windows.Token
	currentProcess, _ := windows.GetCurrentProcess()
	err = windows.OpenProcessToken(currentProcess, windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

func runAsAdmin() {
	exe, err := os.Executable()
	if err != nil {
		return
	}

	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecuteW := shell32.NewProc("ShellExecuteW")

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	parameters, _ := syscall.UTF16PtrFromString("")
	directory, _ := syscall.UTF16PtrFromString("")
	showCmd := uintptr(1)

	shellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(parameters)),
		uintptr(unsafe.Pointer(directory)),
		showCmd,
	)
}

func main() {
	if !isAdmin() {
		runAsAdmin()
		return
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Engine Tools",
		Width:            520,
		Height:           680,
		MinWidth:         480,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &wailsWindows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:                false,
			DisableWindowIcon:                  false,
			WebviewBrowserPath:                 "",
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}