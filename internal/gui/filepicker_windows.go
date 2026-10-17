//go:build windows

package gui

import (
	"runtime"
	"syscall"
	"unsafe"
)

// Здесь только системные диалоги Windows: выбор папки и выбор zip-архива.
// Пишем через winapi напрямую, чтобы не тащить в сборку лишние библиотеки.

var (
	comdlg32            = syscall.NewLazyDLL("comdlg32.dll")
	procGetOpenFileName = comdlg32.NewProc("GetOpenFileNameW")
	shell32             = syscall.NewLazyDLL("shell32.dll")
	procSHBrowseFor     = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPath       = shell32.NewProc("SHGetPathFromIDListW")
	ole32               = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx  = ole32.NewProc("CoInitializeEx")
	procCoUninitialize  = ole32.NewProc("CoUninitialize")
	procCoTaskMemFree   = ole32.NewProc("CoTaskMemFree")
)

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

type browseInfoW struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

const (
	ofnExplorer     = 0x00080000
	ofnFileMustExit = 0x00000800
	ofnPathMustExit = 0x00001000

	bifReturnFSDirs = 0x0001
	bifNewDialog    = 0x0040
	bifEditBox      = 0x0010
)

func utf16(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("")
	}
	return p
}

// pickZipDialog возвращает путь к выбранному архиву или пустую строку.
func pickZipDialog() string {
	buf := make([]uint16, 2048)
	filter := utf16("Архивы модов (*.zip)\x00*.zip\x00Все файлы (*.*)\x00*.*\x00\x00")
	title := utf16("Выбери архив с модом")
	ofn := openFileNameW{
		lStructSize:  uint32(unsafe.Sizeof(openFileNameW{})),
		lpstrFilter:  filter,
		nFilterIndex: 1,
		lpstrFile:    &buf[0],
		nMaxFile:     uint32(len(buf)),
		lpstrTitle:   title,
		flags:        ofnExplorer | ofnFileMustExit | ofnPathMustExit,
	}
	ret, _, _ := procGetOpenFileName.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(filter)
	runtime.KeepAlive(title)
	runtime.KeepAlive(buf)
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

// pickFolderDialog возвращает путь к выбранной папке или пустую строку.
func pickFolderDialog(title string) string {
	hr, _, _ := procCoInitializeEx.Call(0, 2) // COINIT_APARTMENTTHREADED
	if uint32(hr) <= 1 {
		defer procCoUninitialize.Call()
	}

	display := make([]uint16, 260)
	bi := browseInfoW{
		pszDisplayName: &display[0],
		lpszTitle:      utf16(title),
		ulFlags:        bifReturnFSDirs | bifNewDialog | bifEditBox,
	}
	pidl, _, _ := procSHBrowseFor.Call(uintptr(unsafe.Pointer(&bi)))
	runtime.KeepAlive(display)
	runtime.KeepAlive(bi)
	if pidl == 0 {
		return ""
	}
	defer procCoTaskMemFree.Call(pidl)

	out := make([]uint16, 2048)
	ok, _, _ := procSHGetPath.Call(pidl, uintptr(unsafe.Pointer(&out[0])))
	runtime.KeepAlive(out)
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(out)
}
