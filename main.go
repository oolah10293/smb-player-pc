//go:build windows

package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type HWND uintptr
type HDC uintptr
type HBRUSH uintptr
type HPEN uintptr
type HFONT uintptr

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MSG struct {
	Hwnd           HWND
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
	LPrivate       uint32
}
type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground HBRUSH
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}
type PAINTSTRUCT struct {
	Hdc         HDC
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}
type DRAWITEMSTRUCT struct {
	CtlType, CtlID, ItemID, ItemAction, ItemState uint32
	HwndItem                                      HWND
	Hdc                                           HDC
	RcItem                                        RECT
	ItemData                                      uintptr
}
type INITCOMMONCONTROLSEX struct{ DwSize, DwICC uint32 }
type ICONINFO struct {
	FIcon               int32
	XHotspot, YHotspot uint32
	Pad                 uint32
	HbmMask, HbmColor  uintptr
}
type GUID struct {
	Data1        uint32
	Data2, Data3 uint16
	Data4        [8]byte
}
type COMObject struct{ Vtbl *uintptr }
type BROWSEINFO struct {
	HwndOwner      HWND
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

type Entry struct {
	Name, Path string
	IsDir      bool
	ModTime    time.Time
}
type BrowseResult struct {
	Generation uint32
	Path       string
	Entries    []Entry
	Err        error
}

const (
	WS_OVERLAPPEDWINDOW      = 0x00CF0000
	WS_VISIBLE               = 0x10000000
	WS_CHILD                 = 0x40000000
	WS_TABSTOP               = 0x00010000
	WS_VSCROLL               = 0x00200000
	ES_AUTOHSCROLL           = 0x0080
	LBS_NOTIFY               = 0x0001
	LBS_NOINTEGRALHEIGHT     = 0x0100
	BS_OWNERDRAW             = 0x0000000B
	SW_SHOW                  = 5
	IDC_ARROW                = 32512
	IDI_APPLICATION          = 32512
	WM_CREATE                = 0x0001
	WM_DESTROY               = 0x0002
	WM_SIZE                  = 0x0005
	WM_PAINT                 = 0x000F
	WM_ERASEBKGND            = 0x0014
	WM_COMMAND               = 0x0111
	WM_TIMER                 = 0x0113
	WM_HSCROLL               = 0x0114
	WM_DRAWITEM              = 0x002B
	WM_SETICON               = 0x0080
	ICON_SMALL               = 0
	ICON_BIG                 = 1
	WM_CTLCOLORSTATIC        = 0x0138
	WM_CTLCOLOREDIT          = 0x0133
	WM_CTLCOLORLISTBOX       = 0x0134
	WM_SETFONT               = 0x0030
	WM_APP                   = 0x8000
	LB_ADDSTRING             = 0x0180
	LB_RESETCONTENT          = 0x0184
	LB_GETCURSEL             = 0x0188
	LB_SETCURSEL             = 0x0186
	LBN_DBLCLK               = 2
	TBM_GETPOS               = 0x0400
	TBM_SETPOS               = 0x0405
	TBM_SETRANGE             = 0x0406
	TB_ENDTRACK              = 8
	TB_THUMBPOSITION         = 4
	TB_THUMBTRACK            = 5
	ICC_BAR_CLASSES          = 0x00000004
	ODS_SELECTED             = 0x0001
	DT_CENTER                = 0x00000001
	DT_VCENTER               = 0x00000004
	DT_SINGLELINE            = 0x00000020
	DT_LEFT                  = 0x00000000
	DT_RIGHT                 = 0x00000002
	TRANSPARENT              = 1
	PS_SOLID                 = 0
	FW_BOLD                  = 700
	FW_NORMAL                = 400
	CLSCTX_ALL               = 23
	COINIT_APARTMENTTHREADED = 0x2
	SRCCOPY                  = 0x00CC0020
	BIF_RETURNONLYFSDIRS     = 0x0001
	BIF_NEWDIALOGSTYLE       = 0x0040
	BIF_USENEWUI             = BIF_NEWDIALOGSTYLE | 0x0010
	panelTopHeight           = int32(286)

	ID_PATH        = 1001
	ID_GO          = 1002
	ID_UP          = 1003
	ID_PLAYFOLDER  = 1004
	ID_SORT        = 1005
	ID_LIST        = 1006
	ID_PREV        = 1007
	ID_PLAY        = 1008
	ID_NEXT        = 1009
	ID_SEEK        = 1010
	ID_VOL         = 1011
	ID_BROWSE      = 1012
	ID_SHUFFLE     = 1013
	TIMER_UI       = 1
	WM_BROWSE_DONE = WM_APP + 1
	WM_MEDIA_EVENT = WM_APP + 2
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")

	pRegisterClassExW     = user32.NewProc("RegisterClassExW")
	pCreateWindowExW      = user32.NewProc("CreateWindowExW")
	pDefWindowProcW       = user32.NewProc("DefWindowProcW")
	pShowWindow           = user32.NewProc("ShowWindow")
	pUpdateWindow         = user32.NewProc("UpdateWindow")
	pGetMessageW          = user32.NewProc("GetMessageW")
	pTranslateMessage     = user32.NewProc("TranslateMessage")
	pDispatchMessageW     = user32.NewProc("DispatchMessageW")
	pPostQuitMessage      = user32.NewProc("PostQuitMessage")
	pLoadCursorW          = user32.NewProc("LoadCursorW")
	pLoadIconW            = user32.NewProc("LoadIconW")
	pBeginPaint           = user32.NewProc("BeginPaint")
	pEndPaint             = user32.NewProc("EndPaint")
	pGetClientRect        = user32.NewProc("GetClientRect")
	pMoveWindow           = user32.NewProc("MoveWindow")
	pSetWindowTextW       = user32.NewProc("SetWindowTextW")
	pGetWindowTextW       = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	pSendMessageW         = user32.NewProc("SendMessageW")
	pPostMessageW         = user32.NewProc("PostMessageW")
	pSetTimer             = user32.NewProc("SetTimer")
	pKillTimer            = user32.NewProc("KillTimer")
	pInvalidateRect       = user32.NewProc("InvalidateRect")
	pFillRect             = user32.NewProc("FillRect")
	pFrameRect            = user32.NewProc("FrameRect")
	pDrawTextW            = user32.NewProc("DrawTextW")
	pGetDC                = user32.NewProc("GetDC")
	pReleaseDC            = user32.NewProc("ReleaseDC")
	pCreateIconIndirect   = user32.NewProc("CreateIconIndirect")
	pDestroyIcon          = user32.NewProc("DestroyIcon")

	pSetBkColor             = gdi32.NewProc("SetBkColor")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pMoveToEx               = gdi32.NewProc("MoveToEx")
	pLineTo                 = gdi32.NewProc("LineTo")
	pRectangle              = gdi32.NewProc("Rectangle")
	pEllipse                = gdi32.NewProc("Ellipse")
	pCreateFontW            = gdi32.NewProc("CreateFontW")
	pCreateBitmap           = gdi32.NewProc("CreateBitmap")

	pGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	pTimeBeginPeriod      = winmm.NewProc("timeBeginPeriod")
	pTimeEndPeriod        = winmm.NewProc("timeEndPeriod")
	pCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	pCoUninitialize       = ole32.NewProc("CoUninitialize")
	pCoCreateInstance     = ole32.NewProc("CoCreateInstance")
	pCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
	pSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")
	pInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")
	pSetWindowTheme       = uxtheme.NewProc("SetWindowTheme")
)

var (
	mainHwnd, pathEdit, listBox, btnGo, btnBrowse, btnUp, btnPlayFolder, btnSort, btnShuffle, btnPrev, btnPlay, btnNext, seekBar, volBar HWND
	hInstance                                                                                                                uintptr
	appFont, displayFont, meterFont, tinyFont                                                                                HFONT
	brushDark, brushPanel, brushEdit, brushButton, brushButtonDown, brushCream, brushBlack, brushDisplay, brushLampOn        HBRUSH
	penBlack, penRed, penScale, penBevelLight, penBevelDark, penMeterLight, penMeterGlow, penPanelAccent, penLampGlow       HPEN
	currentEntries                                                                                                           []Entry
	currentDir                                                                                                               string
	sortMode                                                                                                                 int
	playlist                                                                                                                 []string
	currentIndex                                                                                                             = -1
	currentTrack                                                                                                             string
	playing, paused                                                                                                          bool
	trackLengthMs, trackPosMs                                                                                                int
	volume                                                                                                                   = 780
	leftMeter, rightMeter                                                                                                    float64
	leftMeterVelocity, rightMeterVelocity                                                                                    float64
	appIcon                                                                                                                   uintptr
	appIconOwned                                                                                                              bool
	meterInfo                                                                                                                *COMObject
	browseMu                                                                                                                 sync.Mutex
	browseResult                                                                                                             BrowseResult
	browseGen                                                                                                                uint32
	timerTicks                                                                                                               int
	loading                                                                                                                  bool
	shuffleMode                                                                                                              bool
	shuffleQueue, shuffleHistory                                                                                             []int
	shuffleHistoryPos                                                                                                        = -1
)

func wstr(s string) *uint16      { p, _ := syscall.UTF16PtrFromString(s); return p }
func color(r, g, b byte) uintptr { return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16) }
func loword(v uintptr) uint16    { return uint16(v & 0xffff) }
func hiword(v uintptr) uint16    { return uint16((v >> 16) & 0xffff) }
func makeLong(lo, hi int) uintptr {
	return uintptr(uint32(uint16(lo)) | uint32(uint16(hi))<<16)
}
func max32(a, b int32) int32 {
	if a > b { return a }
	return b
}
func send(hwnd HWND, msg uint32, w, l uintptr) uintptr {
	r, _, _ := pSendMessageW.Call(uintptr(hwnd), uintptr(msg), w, l)
	return r
}
func setText(hwnd HWND, s string) { pSetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(wstr(s)))) }
func getText(hwnd HWND) string {
	n, _, _ := pGetWindowTextLengthW.Call(uintptr(hwnd))
	buf := make([]uint16, int(n)+1)
	pGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}
func createWindow(ex uintptr, cls, text string, style uintptr, x, y, w, h int32, parent HWND, id int) HWND {
	r, _, _ := pCreateWindowExW.Call(ex, uintptr(unsafe.Pointer(wstr(cls))), uintptr(unsafe.Pointer(wstr(text))), style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(parent), uintptr(id), hInstance, 0)
	return HWND(r)
}

func guid(d1 uint32, d2, d3 uint16, d4 ...byte) GUID {
	g := GUID{Data1: d1, Data2: d2, Data3: d3}
	copy(g.Data4[:], d4)
	return g
}
func comCall(obj *COMObject, index int, args ...uintptr) uintptr {
	if obj == nil || obj.Vtbl == nil { return ^uintptr(0) }
	fn := *(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(obj.Vtbl)) + uintptr(index)*unsafe.Sizeof(uintptr(0))))
	all := append([]uintptr{uintptr(unsafe.Pointer(obj))}, args...)
	r, _, _ := syscall.SyscallN(fn, all...)
	return r
}
func comRelease(obj *COMObject) { if obj != nil { comCall(obj, 2) } }

func initAudioMeter() {
	pCoInitializeEx.Call(0, COINIT_APARTMENTTHREADED)
	clsid := guid(0xBCDE0395, 0xE52F, 0x467C, 0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E)
	iidEnum := guid(0xA95664D2, 0x9614, 0x4F35, 0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6)
	var enumerator *COMObject
	hr, _, _ := pCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsid)), 0, CLSCTX_ALL, uintptr(unsafe.Pointer(&iidEnum)), uintptr(unsafe.Pointer(&enumerator)))
	if int32(hr) < 0 || enumerator == nil { return }
	defer comRelease(enumerator)
	var device *COMObject
	if int32(comCall(enumerator, 4, 0, 1, uintptr(unsafe.Pointer(&device)))) < 0 || device == nil { return }
	defer comRelease(device)
	iidMeter := guid(0xC02216F6, 0x8C67, 0x4B5B, 0x9D, 0x00, 0xD0, 0x08, 0xE7, 0x3E, 0x00, 0x64)
	var mi *COMObject
	if int32(comCall(device, 3, uintptr(unsafe.Pointer(&iidMeter)), CLSCTX_ALL, 0, uintptr(unsafe.Pointer(&mi)))) >= 0 {
		meterInfo = mi
	}
}
func readMeter() (float64, float64) {
	if meterInfo == nil { return 0, 0 }
	var ch uint32
	if int32(comCall(meterInfo, 4, uintptr(unsafe.Pointer(&ch)))) < 0 || ch == 0 {
		var f float32
		if int32(comCall(meterInfo, 3, uintptr(unsafe.Pointer(&f)))) < 0 { return 0, 0 }
		return float64(f), float64(f)
	}
	vals := make([]float32, ch)
	if int32(comCall(meterInfo, 5, uintptr(ch), uintptr(unsafe.Pointer(&vals[0])))) < 0 { return 0, 0 }
	l, r := float64(vals[0]), float64(vals[0])
	if ch > 1 { r = float64(vals[1]) }
	return l, r
}
func meterMap(v float64) float64 {
	if v <= 0 { return 0 }
	db := 20 * math.Log10(v)
	if db < -36 { db = -36 }
	if db > 0 { db = 0 }

	// Keep the scale recognizably VU-like, but deliberately expand the
	// middle of the range so modern compressed music still produces
	// visible needle travel instead of parking around one spot.
	f := (db + 36) / 36
	f = 0.5 + (f-0.5)*1.45
	if f < 0 { f = 0 }
	if f > 1 { f = 1 }
	return f
}

func stepNeedle(pos, velocity, target float64) (float64, float64) {
	// Fast, lightly damped second-order motion. The needle is allowed to
	// move quickly; the goal is continuous physical-looking travel rather
	// than slow averaging or frame-to-frame teleporting.
	const dt = 0.010
	const spring = 1600.0
	const damping = 50.0
	accel := spring*(target-pos) - damping*velocity
	velocity += accel * dt
	pos += velocity * dt
	if pos < 0 {
		pos = 0
		if velocity < 0 { velocity = 0 }
	}
	if pos > 1 {
		pos = 1
		if velocity > 0 { velocity = 0 }
	}
	return pos, velocity
}

func playbackFailure(path string) {
	currentTrack = "CAN'T OPEN: " + filepath.Base(path)
	playing, paused, loading = false, false, false
	trackLengthMs, trackPosMs = 0, 0
	setText(btnPlay, "PLAY")
	send(seekBar, TBM_SETPOS, 1, 0)
	pInvalidateRect.Call(uintptr(mainHwnd), 0, 1)
}

func playFile(path string, index int) {
	currentTrack = path
	currentIndex = index
	trackLengthMs, trackPosMs = 0, 0
	playing, paused, loading = true, false, true
	setText(btnPlay, "PAUSE")
	send(seekBar, TBM_SETPOS, 1, 0)
	if err := mediaOpen(path); err != nil {
		playbackFailure(path)
		return
	}
	pInvalidateRect.Call(uintptr(mainHwnd), 0, 1)
}

func isAudio(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3", ".wav", ".wma", ".m4a", ".aac", ".flac", ".ogg":
		return true
	}
	return false
}

func folderPlaylist() ([]string, map[string]int) {
	var p []string
	idx := map[string]int{}
	for _, e := range currentEntries {
		if !e.IsDir && isAudio(e.Path) {
			idx[e.Path] = len(p)
			p = append(p, e.Path)
		}
	}
	return p, idx
}

func buildShuffleQueue(exclude int) {
	shuffleQueue = nil
	if len(playlist) <= 1 { return }
	perm := rand.Perm(len(playlist))
	for _, i := range perm {
		if i != exclude { shuffleQueue = append(shuffleQueue, i) }
	}
}

func buildShuffleCycle(current int) {
	if len(playlist) == 0 { shuffleQueue = nil; return }
	shuffleQueue = rand.Perm(len(playlist))
	if len(shuffleQueue) > 1 && shuffleQueue[0] == current {
		j := 1 + rand.Intn(len(shuffleQueue)-1)
		shuffleQueue[0], shuffleQueue[j] = shuffleQueue[j], shuffleQueue[0]
	}
}

func resetShuffleState(start int) {
	shuffleQueue = nil
	shuffleHistory = nil
	shuffleHistoryPos = -1
	if !shuffleMode || start < 0 || start >= len(playlist) { return }
	shuffleHistory = []int{start}
	shuffleHistoryPos = 0
	buildShuffleQueue(start)
}

func startFolderPlayback(p []string) {
	if len(p) == 0 { return }
	playlist = p
	if shuffleMode {
		i := rand.Intn(len(playlist))
		resetShuffleState(i)
		playFile(playlist[i], i)
		return
	}
	resetShuffleState(-1)
	playFile(playlist[0], 0)
}

func nextTrack(delta int) {
	if len(playlist) == 0 { return }
	if shuffleMode {
		if delta < 0 {
			if shuffleHistoryPos > 0 {
				shuffleHistoryPos--
				i := shuffleHistory[shuffleHistoryPos]
				playFile(playlist[i], i)
			}
			return
		}
		if shuffleHistoryPos+1 < len(shuffleHistory) {
			shuffleHistoryPos++
			i := shuffleHistory[shuffleHistoryPos]
			playFile(playlist[i], i)
			return
		}
		if len(shuffleQueue) == 0 { buildShuffleCycle(currentIndex) }
		if len(shuffleQueue) == 0 { return }
		i := shuffleQueue[0]
		shuffleQueue = shuffleQueue[1:]
		if shuffleHistoryPos+1 < len(shuffleHistory) {
			shuffleHistory = shuffleHistory[:shuffleHistoryPos+1]
		}
		shuffleHistory = append(shuffleHistory, i)
		shuffleHistoryPos = len(shuffleHistory)-1
		playFile(playlist[i], i)
		return
	}
	i := currentIndex
	if i < 0 { i = 0 } else { i += delta }
	if i < 0 { i = len(playlist)-1 }
	if i >= len(playlist) { i = 0 }
	playFile(playlist[i], i)
}

func togglePlay() {
	if currentTrack == "" || strings.HasPrefix(currentTrack, "CAN'T") || strings.HasPrefix(currentTrack, "AUDIO INIT ERROR") {
		return
	}
	if paused {
		if mediaPlay() == nil {
			playing, paused, loading = true, false, false
			setText(btnPlay, "PAUSE")
		}
		return
	}
	if playing {
		if mediaPause() == nil {
			paused = true
			setText(btnPlay, "PLAY")
		}
		return
	}
	if mediaPlay() == nil {
		playing, paused = true, false
		setText(btnPlay, "PAUSE")
	}
}

func toggleShuffle() {
	shuffleMode = !shuffleMode
	if shuffleMode {
		setText(btnShuffle, "SHUFFLE: ON")
		if currentIndex >= 0 && currentIndex < len(playlist) { resetShuffleState(currentIndex) }
	} else {
		setText(btnShuffle, "SHUFFLE: OFF")
		shuffleQueue = nil
		shuffleHistory = nil
		shuffleHistoryPos = -1
	}
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir != b.IsDir { return a.IsDir }
		switch sortMode {
		case 1:
			return strings.ToLower(a.Name) > strings.ToLower(b.Name)
		case 2:
			return a.ModTime.After(b.ModTime)
		case 3:
			return a.ModTime.Before(b.ModTime)
		default:
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
	})
}
func populateList() {
	send(listBox, LB_RESETCONTENT, 0, 0)
	for _, e := range currentEntries {
		name := e.Name
		if e.IsDir { name = "[DIR]  " + name }
		send(listBox, LB_ADDSTRING, 0, uintptr(unsafe.Pointer(wstr(name))))
	}
}
func loadFolderAsync(path string) {
	path = strings.TrimSpace(path)
	if path == "" { return }
	browseGen++
	gen := browseGen
	setText(btnGo, "...")
	go func() {
		clean := filepath.Clean(path)
		des, err := os.ReadDir(clean)
		var entries []Entry
		if err == nil {
			entries = make([]Entry, 0, len(des))
			for _, de := range des {
				p := filepath.Join(clean, de.Name())
				isDir := de.IsDir()
				if !isDir && !isAudio(p) { continue }
				var mt time.Time
				if info, e := de.Info(); e == nil { mt = info.ModTime() }
				entries = append(entries, Entry{Name: de.Name(), Path: p, IsDir: isDir, ModTime: mt})
			}
			sortEntries(entries)
		}
		browseMu.Lock()
		browseResult = BrowseResult{Generation: gen, Path: clean, Entries: entries, Err: err}
		browseMu.Unlock()
		pPostMessageW.Call(uintptr(mainHwnd), WM_BROWSE_DONE, uintptr(gen), 0)
	}()
}
func playSelectedRow(row int) {
	if row < 0 || row >= len(currentEntries) { return }
	e := currentEntries[row]
	if e.IsDir {
		setText(pathEdit, e.Path)
		loadFolderAsync(e.Path)
		return
	}
	p, idx := folderPlaylist()
	playlist = p
	i := idx[e.Path]
	resetShuffleState(i)
	playFile(e.Path, i)
}
func chooseFolder(owner HWND) string {
	display := make([]uint16, 260)
	bi := BROWSEINFO{HwndOwner: owner, PszDisplayName: &display[0], LpszTitle: wstr("Choose music folder"), UlFlags: BIF_RETURNONLYFSDIRS | BIF_USENEWUI}
	pidl, _, _ := pSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 { return "" }
	defer pCoTaskMemFree.Call(pidl)
	buf := make([]uint16, 32768)
	r, _, _ := pSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&buf[0])))
	if r == 0 { return "" }
	return syscall.UTF16ToString(buf)
}
func saveLastFolder(path string) {
	go func() {
		d, err := os.UserConfigDir(); if err != nil { return }
		d = filepath.Join(d, "SMBPlayerPC")
		if os.MkdirAll(d, 0755) != nil { return }
		_ = os.WriteFile(filepath.Join(d, "lastfolder.txt"), []byte(path), 0644)
	}()
}
func loadLastFolder() string {
	d, err := os.UserConfigDir(); if err != nil { return "" }
	b, err := os.ReadFile(filepath.Join(d, "SMBPlayerPC", "lastfolder.txt")); if err != nil { return "" }
	return strings.TrimSpace(string(b))
}

func drawText(hdc HDC, text string, r RECT, flags uint32, col uintptr, font HFONT) {
	pSetTextColor.Call(uintptr(hdc), col)
	pSetBkMode.Call(uintptr(hdc), TRANSPARENT)
	if font != 0 { old, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(font)); defer pSelectObject.Call(uintptr(hdc), old) }
	pDrawTextW.Call(uintptr(hdc), uintptr(unsafe.Pointer(wstr(text))), uintptr(^uint32(0)), uintptr(unsafe.Pointer(&r)), uintptr(flags))
}
func fill(hdc HDC, r RECT, b HBRUSH) { pFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&r)), uintptr(b)) }
func frame(hdc HDC, r RECT, b HBRUSH) { pFrameRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&r)), uintptr(b)) }
func line(hdc HDC, x1, y1, x2, y2 int32, pen HPEN) {
	old, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(pen))
	pMoveToEx.Call(uintptr(hdc), uintptr(x1), uintptr(y1), 0)
	pLineTo.Call(uintptr(hdc), uintptr(x2), uintptr(y2))
	pSelectObject.Call(uintptr(hdc), old)
}
func meterAngle(frac float64) float64 {
	if frac < 0 { frac = 0 }
	if frac > 1 { frac = 1 }
	return (-55.0 + 110.0*frac) * math.Pi / 180
}
func meterPoint(cx, cy int32, radius float64, frac float64) (int32, int32) {
	a := meterAngle(frac)
	return int32(float64(cx) + math.Sin(a)*radius), int32(float64(cy) - math.Cos(a)*radius)
}
func drawInsetFrame(hdc HDC, r RECT) {
	line(hdc, r.Left, r.Top, r.Right-1, r.Top, penBevelDark)
	line(hdc, r.Left, r.Top, r.Left, r.Bottom-1, penBevelDark)
	line(hdc, r.Left, r.Bottom-1, r.Right-1, r.Bottom-1, penBevelLight)
	line(hdc, r.Right-1, r.Top, r.Right-1, r.Bottom-1, penBevelLight)
}
func drawRaisedFrame(hdc HDC, r RECT, pressed bool) {
	topLeft, bottomRight := penBevelLight, penBevelDark
	if pressed { topLeft, bottomRight = bottomRight, topLeft }
	line(hdc, r.Left, r.Top, r.Right-1, r.Top, topLeft)
	line(hdc, r.Left, r.Top, r.Left, r.Bottom-1, topLeft)
	line(hdc, r.Left, r.Bottom-1, r.Right-1, r.Bottom-1, bottomRight)
	line(hdc, r.Right-1, r.Top, r.Right-1, r.Bottom-1, bottomRight)
}

func makeAppIcon() uintptr {
	screenR, _, _ := pGetDC.Call(0)
	if screenR == 0 { return 0 }
	defer pReleaseDC.Call(0, screenR)

	memR, _, _ := pCreateCompatibleDC.Call(screenR)
	if memR == 0 { return 0 }
	defer pDeleteDC.Call(memR)
	mem := HDC(memR)

	bmpR, _, _ := pCreateCompatibleBitmap.Call(screenR, 32, 32)
	if bmpR == 0 { return 0 }
	old, _, _ := pSelectObject.Call(memR, bmpR)

	fill(mem, RECT{0, 0, 32, 32}, brushBlack)
	fill(mem, RECT{2, 3, 30, 29}, brushPanel)
	fill(mem, RECT{5, 6, 27, 25}, brushCream)
	line(mem, 7, 9, 25, 9, penMeterLight)
	line(mem, 8, 18, 24, 18, penScale)
	line(mem, 16, 23, 23, 12, penRed)
	oldB, _, _ := pSelectObject.Call(memR, uintptr(brushBlack))
	oldP, _, _ := pSelectObject.Call(memR, uintptr(penBlack))
	pEllipse.Call(memR, 14, 21, 18, 25)
	pSelectObject.Call(memR, oldP)
	pSelectObject.Call(memR, oldB)

	pSelectObject.Call(memR, old)
	maskR, _, _ := pCreateBitmap.Call(32, 32, 1, 1, 0)
	if maskR == 0 {
		pDeleteObject.Call(bmpR)
		return 0
	}
	ii := ICONINFO{FIcon: 1, HbmMask: maskR, HbmColor: bmpR}
	ico, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	pDeleteObject.Call(maskR)
	pDeleteObject.Call(bmpR)
	return ico
}
func drawMeter(hdc HDC, r RECT, level float64, label string) {
	fill(hdc, r, brushBlack)
	bezel := RECT{r.Left + 2, r.Top + 2, r.Right - 2, r.Bottom - 2}
	fill(hdc, bezel, brushPanel)
	drawRaisedFrame(hdc, bezel, false)
	face := RECT{r.Left + 10, r.Top + 10, r.Right - 10, r.Bottom - 10}
	fill(hdc, face, brushCream)
	drawInsetFrame(hdc, face)

	// Warm backlight texture; use persistent pens so high-rate animation
	// does not constantly create/delete GDI objects.
	for i := face.Top + 3; i < face.Bottom-3; i += 4 {
		line(hdc, face.Left+3, i, face.Right-4, i, penMeterGlow)
	}
	line(hdc, face.Left+18, face.Top+14, face.Right-22, face.Top+14, penMeterLight)
	line(hdc, face.Left+24, face.Top+18, face.Right-34, face.Top+18, penMeterLight)

	cx := (face.Left + face.Right) / 2
	cy := face.Bottom - 18
	h := float64(face.Bottom - face.Top)
	outer := math.Min(float64(face.Right-face.Left)*0.28, h*0.62)
	needleR := outer - 10

	var px, py int32
	for i := 0; i <= 80; i++ {
		f := float64(i) / 80
		x, y := meterPoint(cx, cy, outer, f)
		if i > 0 { line(hdc, px, py, x, y, penScale) }
		px, py = x, y
	}
	for i := 0; i <= 24; i++ {
		f := float64(i) / 24
		p := penScale
		if f >= 0.84 { p = penRed }
		length := 5.0
		if i%2 == 0 { length = 8 }
		if i%6 == 0 { length = 11 }
		x1, y1 := meterPoint(cx, cy, outer+1, f)
		x2, y2 := meterPoint(cx, cy, outer-length, f)
		line(hdc, x1, y1, x2, y2, p)
	}

	major := []struct{ db int; frac float64 }{
		{-36, 0.00}, {-20, 0.34}, {-10, 0.60}, {-5, 0.76}, {-3, 0.85}, {0, 1.00},
	}
	for _, m := range major {
		x, y := meterPoint(cx, cy, outer+17, m.frac)
		col := color(48, 42, 32)
		if m.db >= -3 { col = color(163, 45, 34) }
		drawText(hdc, fmt.Sprintf("%d", m.db), RECT{x - 17, y - 8, x + 17, y + 10}, DT_CENTER|DT_SINGLELINE, col, meterFont)
	}
	drawText(hdc, "dB", RECT{cx - 24, face.Top + 8, cx + 24, face.Top + 24}, DT_CENTER|DT_SINGLELINE, color(112, 96, 60), tinyFont)

	ang := meterAngle(level)
	nx := int32(float64(cx) + math.Sin(ang)*needleR)
	ny := int32(float64(cy) - math.Cos(ang)*needleR)
	line(hdc, cx, cy, nx, ny, penRed)
	bx := int32(float64(cx) - math.Sin(ang)*12)
	by := int32(float64(cy) + math.Cos(ang)*12)
	line(hdc, cx, cy, bx, by, penRed)
	oldB, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(brushBlack))
	oldP, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(penBlack))
	pEllipse.Call(uintptr(hdc), uintptr(cx-6), uintptr(cy-6), uintptr(cx+7), uintptr(cy+7))
	pSelectObject.Call(uintptr(hdc), oldP)
	pSelectObject.Call(uintptr(hdc), oldB)

	// Keep these markings away from the pivot/needle sweep.
	drawText(hdc, label, RECT{face.Left + 12, face.Bottom - 25, cx - 20, face.Bottom - 7}, DT_LEFT|DT_SINGLELINE, color(64, 55, 40), tinyFont)
	drawText(hdc, "VU", RECT{cx + 20, face.Bottom - 25, face.Right - 12, face.Bottom - 7}, DT_RIGHT|DT_SINGLELINE, color(128, 104, 61), tinyFont)
}
func formatTime(ms int) string {
	if ms < 0 { ms = 0 }
	s := ms / 1000
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
func meterRects(hwnd HWND) (RECT, RECT) {
	var rc RECT
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	gap := int32(30)
	meterW := (rc.Right - 120 - gap) / 2
	if meterW > 390 { meterW = 390 }
	if meterW < 300 { meterW = 300 }
	meterH := int32(166)
	left := (rc.Right - (meterW*2 + gap)) / 2
	top := int32(22)
	return RECT{left, top, left + meterW, top + meterH}, RECT{left + meterW + gap, top, left + meterW*2 + gap, top + meterH}
}
func drawTopPanel(hdc HDC, rc RECT) {
	fill(hdc, RECT{0, 0, rc.Right, panelTopHeight}, brushDark)
	top := RECT{10, 10, rc.Right - 10, panelTopHeight - 8}
	fill(hdc, top, brushPanel)
	for y := top.Top + 1; y < top.Bottom-1; y += 4 { line(hdc, top.Left+1, y, top.Right-2, y, penPanelAccent) }
	frame(hdc, top, brushBlack); drawInsetFrame(hdc, top)
	leftR, rightR := meterRects(mainHwnd)
	analyzerR := RECT{leftR.Left, leftR.Top, rightR.Right, rightR.Bottom}
	drawSpectrumAnalyzer(hdc, analyzerR)

	oldB, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(brushLampOn))
	oldP, _, _ := pSelectObject.Call(uintptr(hdc), uintptr(penLampGlow))
	pEllipse.Call(uintptr(hdc), 32, 206, 48, 222)
	pSelectObject.Call(uintptr(hdc), oldP); pSelectObject.Call(uintptr(hdc), oldB)
	drawText(hdc, "POWER", RECT{54, 204, 104, 223}, DT_LEFT|DT_VCENTER|DT_SINGLELINE, color(164, 177, 160), tinyFont)

	displayR := RECT{120, 192, rc.Right - 24, 248}
	fill(hdc, displayR, brushDisplay); drawInsetFrame(hdc, displayR)
	title := "NO TRACK LOADED"
	if currentTrack != "" { title = strings.ToUpper(strings.TrimSuffix(filepath.Base(currentTrack), filepath.Ext(currentTrack))) }
	drawText(hdc, title, RECT{136, 198, rc.Right - 36, 221}, DT_CENTER|DT_VCENTER|DT_SINGLELINE, color(242, 184, 65), displayFont)
	status := "READY"; if loading { status = "LOADING" } else if playing { if paused { status = "PAUSED" } else { status = "PLAYING" } }
	drawText(hdc, fmt.Sprintf("%s    %s  /  %s", status, formatTime(trackPosMs), formatTime(trackLengthMs)), RECT{136, 222, rc.Right - 36, 242}, DT_CENTER|DT_VCENTER|DT_SINGLELINE, color(127, 224, 117), appFont)
	drawText(hdc, "NETWORK AUDIO", RECT{26, 234, 110, 252}, DT_LEFT|DT_SINGLELINE, color(175, 159, 118), tinyFont)
	drawText(hdc, "REMOTE AUDIO DECK  •  v0.3.0", RECT{rc.Right / 2, 252, rc.Right - 26, 270}, DT_RIGHT|DT_SINGLELINE, color(125, 128, 119), tinyFont)
}
func paintMain(hwnd HWND) {
	var ps PAINTSTRUCT
	hdcR, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	hdc := HDC(hdcR)
	var rc RECT
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	if ps.RcPaint.Top < panelTopHeight && rc.Right > 0 {
		panelH := panelTopHeight
		if rc.Bottom < panelH { panelH = rc.Bottom }
		if panelH > 0 {
			memR, _, _ := pCreateCompatibleDC.Call(uintptr(hdc))
			if memR != 0 {
				mem := HDC(memR)
				bmpR, _, _ := pCreateCompatibleBitmap.Call(uintptr(hdc), uintptr(rc.Right), uintptr(panelH))
				if bmpR != 0 {
					old, _, _ := pSelectObject.Call(uintptr(mem), bmpR)
					drawTopPanel(mem, rc)
					pBitBlt.Call(uintptr(hdc), 0, 0, uintptr(rc.Right), uintptr(panelH), uintptr(mem), 0, 0, SRCCOPY)
					pSelectObject.Call(uintptr(mem), old)
					pDeleteObject.Call(bmpR)
				}
				pDeleteDC.Call(uintptr(mem))
			}
		}
	}
	if ps.RcPaint.Bottom > panelTopHeight {
		bottom := ps.RcPaint
		if bottom.Top < panelTopHeight { bottom.Top = panelTopHeight }
		fill(hdc, bottom, brushDark)

		w, h := rc.Right, rc.Bottom
		margin := int32(12)
		controlY := h - 54
		labelTop := controlY - 25
		seekX := margin + 265
		seekW := max32(120, w-seekX-170)
		volX := seekX + seekW + 10
		volW := int32(140)

		drawText(hdc, "SEEK", RECT{seekX, labelTop, seekX + seekW, labelTop + 16}, DT_CENTER|DT_SINGLELINE, color(175, 159, 118), tinyFont)
		drawText(hdc, "VOLUME", RECT{volX, labelTop, volX + volW, labelTop + 16}, DT_CENTER|DT_SINGLELINE, color(175, 159, 118), tinyFont)

		listY := int32(372)
		listBottom := labelTop - 10
		if listBottom > listY+20 {
			fr := RECT{margin - 2, listY - 2, w - margin + 2, listBottom + 2}
			frame(hdc, fr, brushBlack)
			drawInsetFrame(hdc, fr)
		}
	}
	pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
}
func layout(hwnd HWND) {
	var rc RECT
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w, h := rc.Right, rc.Bottom
	margin := int32(12)
	y := int32(296)
	goW, browseW, gap := int32(68), int32(92), int32(6)
	pathW := max32(180, w-2*margin-(goW+browseW+gap*2))
	pMoveWindow.Call(uintptr(pathEdit), uintptr(margin), uintptr(y), uintptr(pathW), 28, 1)
	bx := margin + pathW + gap
	pMoveWindow.Call(uintptr(btnBrowse), uintptr(bx), uintptr(y), uintptr(browseW), 28, 1)
	pMoveWindow.Call(uintptr(btnGo), uintptr(bx+browseW+gap), uintptr(y), uintptr(goW), 28, 1)

	y = 332
	bw := int32(118)
	pMoveWindow.Call(uintptr(btnUp), uintptr(margin), uintptr(y), uintptr(bw), 30, 1)
	pMoveWindow.Call(uintptr(btnPlayFolder), uintptr(margin+bw+8), uintptr(y), 150, 30, 1)
	pMoveWindow.Call(uintptr(btnSort), uintptr(margin+bw+166), uintptr(y), 140, 30, 1)
	pMoveWindow.Call(uintptr(btnShuffle), uintptr(margin+bw+314), uintptr(y), 140, 30, 1)

	controlY := h - 54
	labelTop := controlY - 25
	listY := int32(372)
	listBottom := labelTop - 10
	listH := listBottom - listY
	if listH < 80 { listH = 80 }
	pMoveWindow.Call(uintptr(listBox), uintptr(margin), uintptr(listY), uintptr(w-2*margin), uintptr(listH), 1)

	pMoveWindow.Call(uintptr(btnPrev), uintptr(margin), uintptr(controlY), 74, 34, 1)
	pMoveWindow.Call(uintptr(btnPlay), uintptr(margin+82), uintptr(controlY), 86, 34, 1)
	pMoveWindow.Call(uintptr(btnNext), uintptr(margin+176), uintptr(controlY), 74, 34, 1)
	seekX := margin + 265
	seekW := max32(120, w-seekX-170)
	pMoveWindow.Call(uintptr(seekBar), uintptr(seekX), uintptr(controlY+2), uintptr(seekW), 30, 1)
	pMoveWindow.Call(uintptr(volBar), uintptr(seekX+seekW+10), uintptr(controlY+2), 140, 30, 1)
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		mainHwnd = HWND(hwnd); initAudioMeter(); startSpectrumAnalyzer()
		pathEdit = createWindow(0, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, 0, 0, 0, 0, mainHwnd, ID_PATH)
		btnBrowse = createWindow(0, "BUTTON", "BROWSE", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_BROWSE)
		btnGo = createWindow(0, "BUTTON", "GO", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_GO)
		btnUp = createWindow(0, "BUTTON", "UP", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_UP)
		btnPlayFolder = createWindow(0, "BUTTON", "PLAY FOLDER", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_PLAYFOLDER)
		btnSort = createWindow(0, "BUTTON", "SORT: A-Z", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_SORT)
		btnShuffle = createWindow(0, "BUTTON", "SHUFFLE: OFF", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_SHUFFLE)
		listBox = createWindow(0, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_VSCROLL|LBS_NOTIFY|LBS_NOINTEGRALHEIGHT, 0, 0, 0, 0, mainHwnd, ID_LIST)
		btnPrev = createWindow(0, "BUTTON", "|<<", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_PREV)
		btnPlay = createWindow(0, "BUTTON", "PLAY", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_PLAY)
		btnNext = createWindow(0, "BUTTON", ">>|", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 0, 0, 0, mainHwnd, ID_NEXT)
		seekBar = createWindow(0, "msctls_trackbar32", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP, 0, 0, 0, 0, mainHwnd, ID_SEEK)
		volBar = createWindow(0, "msctls_trackbar32", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP, 0, 0, 0, 0, mainHwnd, ID_VOL)
		send(seekBar, TBM_SETRANGE, 1, makeLong(0, 1000)); send(volBar, TBM_SETRANGE, 1, makeLong(0, 1000)); send(volBar, TBM_SETPOS, 1, uintptr(volume))
		pSetWindowTheme.Call(uintptr(seekBar), uintptr(unsafe.Pointer(wstr(""))), uintptr(unsafe.Pointer(wstr(""))))
		pSetWindowTheme.Call(uintptr(volBar), uintptr(unsafe.Pointer(wstr(""))), uintptr(unsafe.Pointer(wstr(""))))
		for _, c := range []HWND{pathEdit, listBox, btnGo, btnBrowse, btnUp, btnPlayFolder, btnSort, btnShuffle, btnPrev, btnPlay, btnNext} { send(c, WM_SETFONT, uintptr(appFont), 1) }
		if err := initMediaEngine(); err != nil { currentTrack = "AUDIO INIT ERROR: " + err.Error() }
		pSetTimer.Call(hwnd, TIMER_UI, 10, 0)
		if s := loadLastFolder(); s != "" { setText(pathEdit, s) } else if home, err := os.UserHomeDir(); err == nil { setText(pathEdit, home) }
		layout(mainHwnd); return 0
	case WM_SIZE:
		layout(HWND(hwnd)); return 0
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		paintMain(HWND(hwnd)); return 0
	case WM_CTLCOLORLISTBOX, WM_CTLCOLOREDIT, WM_CTLCOLORSTATIC:
		hdc := HDC(wParam); pSetTextColor.Call(uintptr(hdc), color(220, 220, 205)); pSetBkColor.Call(uintptr(hdc), color(22, 24, 23)); return uintptr(brushEdit)
	case WM_DRAWITEM:
		di := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam)); if di != nil {
			down := (di.ItemState & ODS_SELECTED) != 0; b := brushButton; if down { b = brushButtonDown }
			fill(di.Hdc, di.RcItem, b); drawRaisedFrame(di.Hdc, di.RcItem, down)
			inner := RECT{di.RcItem.Left + 2, di.RcItem.Top + 2, di.RcItem.Right - 2, di.RcItem.Bottom - 2}; frame(di.Hdc, inner, brushBlack)
			col := color(231, 181, 69); if down { col = color(255, 219, 124) }
			tr := di.RcItem; if down { tr.Left++; tr.Right++; tr.Top++; tr.Bottom++ }
			drawText(di.Hdc, getText(di.HwndItem), tr, DT_CENTER|DT_VCENTER|DT_SINGLELINE, col, appFont); return 1
		}
	case WM_COMMAND:
		id, code := loword(wParam), hiword(wParam)
		switch id {
		case ID_BROWSE:
			if code == 0 { if p := chooseFolder(mainHwnd); p != "" { setText(pathEdit, p); loadFolderAsync(p) } }
		case ID_GO:
			if code == 0 { loadFolderAsync(getText(pathEdit)) }
		case ID_UP:
			if code == 0 { p := strings.TrimSpace(getText(pathEdit)); if p != "" { parent := filepath.Dir(p); setText(pathEdit, parent); loadFolderAsync(parent) } }
		case ID_PLAYFOLDER:
			if code == 0 { p, _ := folderPlaylist(); startFolderPlayback(p) }
		case ID_SORT:
			if code == 0 { sortMode = (sortMode+1)%4; labels := []string{"SORT: A-Z", "SORT: Z-A", "SORT: NEW", "SORT: OLD"}; setText(btnSort, labels[sortMode]); sortEntries(currentEntries); populateList() }
		case ID_SHUFFLE:
			if code == 0 { toggleShuffle() }
		case ID_PREV:
			if code == 0 { nextTrack(-1) }
		case ID_PLAY:
			if code == 0 { togglePlay() }
		case ID_NEXT:
			if code == 0 { nextTrack(1) }
		case ID_LIST:
			if code == LBN_DBLCLK { playSelectedRow(int(send(listBox, LB_GETCURSEL, 0, 0))) }
		}
		return 0
	case WM_HSCROLL:
		src := HWND(lParam); code := loword(wParam)
		if src == volBar { volume = int(send(volBar, TBM_GETPOS, 0, 0)); _ = mediaSetVolume(volume); return 0 }
		if src == seekBar && (code == TB_ENDTRACK || code == TB_THUMBPOSITION || code == TB_THUMBTRACK) && trackLengthMs > 0 {
			pos := int(send(seekBar, TBM_GETPOS, 0, 0)); ms := trackLengthMs * pos / 1000; if mediaSetPosition(ms) == nil { trackPosMs = ms }; return 0
		}
	case WM_TIMER:
		if wParam == TIMER_UI {
			timerTicks++
			if timerTicks%2 == 0 {
				leftR, rightR := meterRects(HWND(hwnd))
				analyzerR := RECT{leftR.Left, leftR.Top, rightR.Right, rightR.Bottom}
				pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&analyzerR)), 0)
			}
			if timerTicks%20 == 0 {
				if playing || paused {
					trackPosMs = mediaPositionMs()
					if trackLengthMs <= 0 { trackLengthMs = mediaDurationMs() }
					if trackLengthMs > 0 { send(seekBar, TBM_SETPOS, 1, uintptr(trackPosMs*1000/trackLengthMs)) }
				}
				statusR := RECT{20, 188, 10000, 248}
				pInvalidateRect.Call(hwnd, uintptr(unsafe.Pointer(&statusR)), 0)
			}
		}
		return 0
	case WM_BROWSE_DONE:
		browseMu.Lock(); br := browseResult; browseMu.Unlock()
		if br.Generation != uint32(wParam) || br.Generation != browseGen { return 0 }
		setText(btnGo, "GO")
		if br.Err != nil { currentTrack = "FOLDER ERROR: " + br.Err.Error(); pInvalidateRect.Call(hwnd, 0, 1); return 0 }
		currentDir, currentEntries = br.Path, br.Entries; setText(pathEdit, currentDir); populateList(); saveLastFolder(currentDir); return 0
	case WM_MEDIA_EVENT:
		switch uint32(wParam) {
		case MF_MEDIA_ENGINE_EVENT_ERROR:
			failed := currentTrack
			playbackFailure(failed)
		case MF_MEDIA_ENGINE_EVENT_LOADEDMETADATA, MF_MEDIA_ENGINE_EVENT_DURATIONCHANGE:
			trackLengthMs = mediaDurationMs()
			pInvalidateRect.Call(hwnd, 0, 1)
		case MF_MEDIA_ENGINE_EVENT_PLAYING:
			loading, playing, paused = false, true, false
			setText(btnPlay, "PAUSE")
			pInvalidateRect.Call(hwnd, 0, 1)
		case MF_MEDIA_ENGINE_EVENT_PAUSE:
			// SetSource can generate state-change events while the next file is
			// loading. Do not let those make the new track look manually paused.
			if playing && !loading { paused = true; setText(btnPlay, "PLAY"); pInvalidateRect.Call(hwnd, 0, 1) }
		case MF_MEDIA_ENGINE_EVENT_ENDED:
			// Likewise, ignore an old source's terminal event while a replacement
			// source is still loading.
			if !loading && playing && !paused { nextTrack(1) }
		}
		return 0
	case WM_DESTROY:
		pKillTimer.Call(hwnd, TIMER_UI); shutdownMediaEngine(); stopSpectrumAnalyzer(); if meterInfo != nil { comRelease(meterInfo); meterInfo = nil }; pCoUninitialize.Call(); pPostQuitMessage.Call(0); return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam); return r
}

func main() {
	runtime.LockOSThread()
	pTimeBeginPeriod.Call(1)
	defer pTimeEndPeriod.Call(1)
	icc := INITCOMMONCONTROLSEX{uint32(unsafe.Sizeof(INITCOMMONCONTROLSEX{})), ICC_BAR_CLASSES}; pInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
	brushDark = HBRUSH(must1(pCreateSolidBrush.Call(color(14, 16, 16))))
	brushPanel = HBRUSH(must1(pCreateSolidBrush.Call(color(37, 39, 38))))
	brushEdit = HBRUSH(must1(pCreateSolidBrush.Call(color(22, 24, 23))))
	brushButton = HBRUSH(must1(pCreateSolidBrush.Call(color(52, 53, 50))))
	brushButtonDown = HBRUSH(must1(pCreateSolidBrush.Call(color(78, 74, 62))))
	brushCream = HBRUSH(must1(pCreateSolidBrush.Call(color(231, 220, 182))))
	brushDisplay = HBRUSH(must1(pCreateSolidBrush.Call(color(8, 18, 14))))
	brushLampOn = HBRUSH(must1(pCreateSolidBrush.Call(color(104, 226, 82))))
	brushBlack = HBRUSH(must1(pCreateSolidBrush.Call(color(25, 25, 22))))
	penBlack = HPEN(must1(pCreatePen.Call(PS_SOLID, 1, color(35, 32, 26))))
	penRed = HPEN(must1(pCreatePen.Call(PS_SOLID, 2, color(178, 45, 34))))
	penScale = HPEN(must1(pCreatePen.Call(PS_SOLID, 1, color(76, 68, 50))))
	penBevelLight = HPEN(must1(pCreatePen.Call(PS_SOLID, 1, color(92, 95, 88))))
	penBevelDark = HPEN(must1(pCreatePen.Call(PS_SOLID, 1, color(15, 16, 15))))
	penMeterLight = HPEN(must1(pCreatePen.Call(PS_SOLID, 1, color(250, 243, 215))))
	penMeterGlow = HPEN(must1(pCreatePen.Call(PS_SOLID, 1, color(236, 228, 196))))
	penPanelAccent = HPEN(must1(pCreatePen.Call(PS_SOLID, 1, color(46, 49, 47))))
	penLampGlow = HPEN(must1(pCreatePen.Call(PS_SOLID, 1, color(178, 255, 152))))
	appFont = HFONT(must1(pCreateFontW.Call(17, 0, 0, 0, FW_NORMAL, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(wstr("Segoe UI"))))))
	displayFont = HFONT(must1(pCreateFontW.Call(24, 0, 0, 0, FW_BOLD, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(wstr("Consolas"))))))
	meterFont = HFONT(must1(pCreateFontW.Call(15, 0, 0, 0, FW_BOLD, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(wstr("Segoe UI"))))))
	tinyFont = HFONT(must1(pCreateFontW.Call(12, 0, 0, 0, FW_BOLD, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(wstr("Segoe UI"))))))
	hInst := must1(pGetModuleHandleW.Call(0)); hInstance = hInst
	cur := must1(pLoadCursorW.Call(0, IDC_ARROW))
	ico := makeAppIcon()
	if ico != 0 {
		appIcon = ico
		appIconOwned = true
	} else {
		ico = must1(pLoadIconW.Call(0, IDI_APPLICATION))
	}
	className := wstr("SMBPlayerPC_V030")
	wc := WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), Style: 0x0003, LpfnWndProc: syscall.NewCallback(wndProc), HInstance: hInst, HIcon: ico, HCursor: cur, HbrBackground: brushDark, LpszClassName: className, HIconSm: ico}
	if r := must1(pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))); r == 0 { return }
	hwnd := createWindow(0, "SMBPlayerPC_V030", "SMB Player PC v0.3.0", WS_OVERLAPPEDWINDOW, -2147483648, -2147483648, 980, 720, 0, 0)
	if hwnd == 0 { return }
	pSendMessageW.Call(uintptr(hwnd), WM_SETICON, ICON_BIG, ico)
	pSendMessageW.Call(uintptr(hwnd), WM_SETICON, ICON_SMALL, ico)
	pShowWindow.Call(uintptr(hwnd), SW_SHOW); pUpdateWindow.Call(uintptr(hwnd))
	var msg MSG
	for { r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0); if int32(r) <= 0 { break }; pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg))); pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg))) }
	if appIconOwned && appIcon != 0 { pDestroyIcon.Call(appIcon) }
}
func must1(r uintptr, _ uintptr, _ error) uintptr { return r }
