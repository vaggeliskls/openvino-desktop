//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const singleInstanceMutex = "Global\\TurintechOpenVINODesktop"

// ensureSingleInstance creates a named mutex. If one already exists, the
// running instance's window is brought to the foreground and this process exits.
func ensureSingleInstance() {
	mutexPtr, _ := syscall.UTF16PtrFromString(singleInstanceMutex)
	_, err := windows.CreateMutex(nil, false, mutexPtr)
	if err != windows.ERROR_ALREADY_EXISTS {
		return
	}

	user32 := syscall.NewLazyDLL("user32.dll")
	findWindow := user32.NewProc("FindWindowW")
	showWindow := user32.NewProc("ShowWindow")
	setForegroundWindow := user32.NewProc("SetForegroundWindow")

	titlePtr, _ := syscall.UTF16PtrFromString("Turintech - OpenVINO Desktop")
	hwnd, _, _ := findWindow.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd != 0 {
		const swRestore = 9
		showWindow.Call(hwnd, swRestore)
		setForegroundWindow.Call(hwnd)
	}
	os.Exit(0)
}
