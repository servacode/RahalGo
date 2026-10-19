//go:build windows

package main

// ما يخصّ ويندوز: الإقلاعُ التلقائيّ، ونسخةٌ واحدة، ورسالةٌ للمحاسب، والعمليّةُ المنفصلة.

import (
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// setAutostart **يقلع مع ويندوز** — لحساب المستخدم، بلا صلاحيات مدير.
func setAutostart(exe string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue("RahalGoSync", `"`+exe+`"`)
}

func removeAutostart() {
	if k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE); err == nil {
		_ = k.DeleteValue("RahalGoSync")
		k.Close()
	}
}

var instance windows.Handle

// singleInstance **نسخةٌ واحدةٌ تعمل** — باسمٍ مشترك.
func singleInstance() bool {
	name, _ := windows.UTF16PtrFromString(`Local\RahalGoSyncAgent`)
	for i := 0; i < 10; i++ {
		h, err := windows.CreateMutex(nil, true, name)
		if err == nil {
			instance = h
			return true
		}
		if h != 0 {
			_ = windows.CloseHandle(h)
		}
		if err != windows.ERROR_ALREADY_EXISTS {
			return false
		}
		// **وبعد تحديثٍ قد تكون القديمةُ تخرج** — يُنتظر قليلاً.
		windows.SleepEx(1000, false)
	}
	return false
}

func releaseInstance() {
	if instance != 0 {
		_ = windows.ReleaseMutex(instance)
		_ = windows.CloseHandle(instance)
		instance = 0
	}
}

// notify **رسالةٌ واحدةٌ للمحاسب** — بالعربيّة ومن اليمين.
func notify(msg string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	box := user32.NewProc("MessageBoxW")
	text, _ := syscall.UTF16PtrFromString(strings.TrimSpace(msg))
	title, _ := syscall.UTF16PtrFromString("رحّال غو")
	const mbOK, mbInfo, mbRight, mbRTL, mbTop = 0x0, 0x40, 0x80000, 0x100000, 0x40000
	_, _, _ = box.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), mbOK|mbInfo|mbRight|mbRTL|mbTop)
}

// detach **عمليّةٌ منفصلةٌ بلا نافذة.**
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
}

// sqlInstances **نسخُ SQL Server المسجّلة على الجهاز.**
func sqlInstances() []string {
	var out []string
	for _, view := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Microsoft SQL Server\Instance Names\SQL`, registry.QUERY_VALUE|view)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(-1)
		k.Close()
		out = append(out, names...)
	}
	return out
}
