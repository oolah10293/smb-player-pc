//go:build windows

package main

import (
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"
)

const (
	MF_MEDIA_ENGINE_EVENT_ERROR          = 5
	MF_MEDIA_ENGINE_EVENT_PAUSE          = 9
	MF_MEDIA_ENGINE_EVENT_LOADEDMETADATA = 10
	MF_MEDIA_ENGINE_EVENT_PLAYING        = 13
	MF_MEDIA_ENGINE_EVENT_ENDED          = 19
	MF_MEDIA_ENGINE_EVENT_DURATIONCHANGE = 21

	mfVersion              = 0x00020070
	mfStartupFull          = 0
	mfMediaEngineAudioOnly = 0x00000001
	clsctxInprocServer     = 0x00000001

	sOK          = 0x00000000
	eNoInterface = 0x80004002
	ePointer     = 0x80004003
)

var (
	mfplat   = syscall.NewLazyDLL("mfplat.dll")
	oleaut32 = syscall.NewLazyDLL("oleaut32.dll")

	pMFStartup          = mfplat.NewProc("MFStartup")
	pMFShutdown         = mfplat.NewProc("MFShutdown")
	pMFCreateAttributes = mfplat.NewProc("MFCreateAttributes")
	pSysAllocString     = oleaut32.NewProc("SysAllocString")
	pSysFreeString      = oleaut32.NewProc("SysFreeString")

	iidIUnknown = guid(
		0x00000000, 0x0000, 0x0000,
		0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46,
	)
	clsidMFMediaEngineClassFactory = guid(
		0xB44392DA, 0x499B, 0x446B,
		0xA4, 0xCB, 0x00, 0x5F, 0xEA, 0xD0, 0xE6, 0xD5,
	)
	iidMFMediaEngineClassFactory = guid(
		0x4D645ACE, 0x26AA, 0x4688,
		0x9B, 0xE1, 0xDF, 0x35, 0x16, 0x99, 0x0B, 0x93,
	)
	iidMFMediaEngineNotify = guid(
		0xFEE7C112, 0xE776, 0x42B5,
		0x9B, 0xBF, 0x00, 0x48, 0x52, 0x4E, 0x2B, 0xD5,
	)
	mfMediaEngineCallback = guid(
		0xC60381B8, 0x83A4, 0x41F8,
		0xA3, 0xD0, 0xDE, 0x05, 0x07, 0x68, 0x49, 0xA9,
	)

	mediaEngine            *COMObject
	mediaFoundationStarted bool
	mediaNotifyVtbl        [4]uintptr
	mediaNotifyObj         mediaEngineNotify
)

type mediaEngineNotify struct {
	Vtbl *uintptr
	refs uint32
}

func guidEqual(a, b *GUID) bool {
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func mediaNotifyQueryInterface(this, riid, ppv uintptr) uintptr {
	if ppv == 0 {
		return ePointer
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	if riid == 0 {
		return eNoInterface
	}
	id := (*GUID)(unsafe.Pointer(riid))
	if !guidEqual(id, &iidIUnknown) && !guidEqual(id, &iidMFMediaEngineNotify) {
		return eNoInterface
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = this
	mediaNotifyAddRef(this)
	return sOK
}

func mediaNotifyAddRef(this uintptr) uintptr {
	obj := (*mediaEngineNotify)(unsafe.Pointer(this))
	return uintptr(atomic.AddUint32(&obj.refs, 1))
}

func mediaNotifyRelease(this uintptr) uintptr {
	obj := (*mediaEngineNotify)(unsafe.Pointer(this))
	for {
		old := atomic.LoadUint32(&obj.refs)
		if old == 0 {
			return 0
		}
		if atomic.CompareAndSwapUint32(&obj.refs, old, old-1) {
			return uintptr(old - 1)
		}
	}
}

func mediaNotifyEvent(this, eventCode, param1, param2 uintptr) uintptr {
	if mainHwnd != 0 {
		pPostMessageW.Call(uintptr(mainHwnd), WM_MEDIA_EVENT, eventCode, param1)
	}
	return sOK
}

func hresultError(what string, hr uintptr) error {
	if int32(hr) >= 0 {
		return nil
	}
	return fmt.Errorf("%s failed (HRESULT 0x%08X)", what, uint32(hr))
}

func initMediaNotify() {
	mediaNotifyVtbl[0] = syscall.NewCallback(mediaNotifyQueryInterface)
	mediaNotifyVtbl[1] = syscall.NewCallback(mediaNotifyAddRef)
	mediaNotifyVtbl[2] = syscall.NewCallback(mediaNotifyRelease)
	mediaNotifyVtbl[3] = syscall.NewCallback(mediaNotifyEvent)
	mediaNotifyObj.Vtbl = &mediaNotifyVtbl[0]
	atomic.StoreUint32(&mediaNotifyObj.refs, 1)
}

func initMediaEngine() error {
	if mediaEngine != nil {
		return nil
	}

	hr, _, _ := pMFStartup.Call(mfVersion, mfStartupFull)
	if err := hresultError("MFStartup", hr); err != nil {
		return err
	}
	mediaFoundationStarted = true

	initMediaNotify()

	var attrs *COMObject
	hr, _, _ = pMFCreateAttributes.Call(
		uintptr(unsafe.Pointer(&attrs)),
		1,
	)
	if err := hresultError("MFCreateAttributes", hr); err != nil || attrs == nil {
		shutdownMediaEngine()
		if err != nil {
			return err
		}
		return fmt.Errorf("MFCreateAttributes returned no attribute store")
	}

	hr = comCall(
		attrs,
		27, // IMFAttributes::SetUnknown
		uintptr(unsafe.Pointer(&mfMediaEngineCallback)),
		uintptr(unsafe.Pointer(&mediaNotifyObj)),
	)
	if err := hresultError("IMFAttributes::SetUnknown(MF_MEDIA_ENGINE_CALLBACK)", hr); err != nil {
		comRelease(attrs)
		shutdownMediaEngine()
		return err
	}

	var factory *COMObject
	hr, _, _ = pCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidMFMediaEngineClassFactory)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidMFMediaEngineClassFactory)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if err := hresultError("CoCreateInstance(MFMediaEngineClassFactory)", hr); err != nil || factory == nil {
		comRelease(attrs)
		shutdownMediaEngine()
		if err != nil {
			return err
		}
		return fmt.Errorf("Media Engine class factory was not created")
	}

	var engine *COMObject
	hr = comCall(
		factory,
		3, // IMFMediaEngineClassFactory::CreateInstance
		mfMediaEngineAudioOnly,
		uintptr(unsafe.Pointer(attrs)),
		uintptr(unsafe.Pointer(&engine)),
	)
	comRelease(factory)
	comRelease(attrs)
	if err := hresultError("IMFMediaEngineClassFactory::CreateInstance", hr); err != nil || engine == nil {
		shutdownMediaEngine()
		if err != nil {
			return err
		}
		return fmt.Errorf("Media Engine was not created")
	}

	mediaEngine = engine
	// A new source is explicitly played by mediaOpen. Keep loop disabled.
	_ = hresultError("IMFMediaEngine::SetAutoPlay", comCall(mediaEngine, 29, 0))
	_ = hresultError("IMFMediaEngine::SetLoop", comCall(mediaEngine, 31, 0))
	if err := mediaSetVolume(volume); err != nil {
		shutdownMediaEngine()
		return err
	}
	return nil
}

func shutdownMediaEngine() {
	if mediaEngine != nil {
		comCall(mediaEngine, 42) // IMFMediaEngine::Shutdown
		comRelease(mediaEngine)
		mediaEngine = nil
	}
	if mediaFoundationStarted {
		pMFShutdown.Call()
		mediaFoundationStarted = false
	}
}

func mediaFileURL(path string) string {
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, `\\`) {
		rest := strings.TrimPrefix(clean, `\\`)
		parts := strings.SplitN(rest, `\`, 2)
		if len(parts) == 2 {
			u := url.URL{
				Scheme: "file",
				Host:   parts[0],
				Path:   "/" + strings.ReplaceAll(parts[1], `\`, "/"),
			}
			return u.String()
		}
	}
	slash := strings.ReplaceAll(clean, `\`, "/")
	if len(slash) >= 2 && slash[1] == ':' {
		slash = "/" + slash
	}
	return (&url.URL{Scheme: "file", Path: slash}).String()
}

func mediaOpen(path string) error {
	if mediaEngine == nil {
		return fmt.Errorf("Media Engine is not initialized")
	}
	uri := mediaFileURL(path)
	bstr, _, _ := pSysAllocString.Call(uintptr(unsafe.Pointer(wstr(uri))))
	if bstr == 0 {
		return fmt.Errorf("could not allocate media source URL")
	}
	defer pSysFreeString.Call(bstr)

	hr := comCall(mediaEngine, 6, bstr) // IMFMediaEngine::SetSource
	if err := hresultError("IMFMediaEngine::SetSource", hr); err != nil {
		return err
	}
	// Play can be requested before loading finishes. Media Engine completes
	// the load asynchronously and reports PLAYING/ERROR through our callback.
	return mediaPlay()
}

func mediaPlay() error {
	if mediaEngine == nil {
		return fmt.Errorf("Media Engine is not initialized")
	}
	return hresultError("IMFMediaEngine::Play", comCall(mediaEngine, 32))
}

func mediaPause() error {
	if mediaEngine == nil {
		return fmt.Errorf("Media Engine is not initialized")
	}
	return hresultError("IMFMediaEngine::Pause", comCall(mediaEngine, 33))
}

func comCallDoubleResult(obj *COMObject, index int) float64 {
	if obj == nil || obj.Vtbl == nil {
		return math.NaN()
	}
	fn := *(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(obj.Vtbl)) + uintptr(index)*unsafe.Sizeof(uintptr(0))))
	_, r2, _ := syscall.SyscallN(fn, uintptr(unsafe.Pointer(obj)))
	return math.Float64frombits(uint64(r2))
}

func mediaSetPosition(ms int) error {
	if mediaEngine == nil {
		return fmt.Errorf("Media Engine is not initialized")
	}
	if ms < 0 {
		ms = 0
	}
	seconds := float64(ms) / 1000.0
	return hresultError(
		"IMFMediaEngine::SetCurrentTime",
		comCall(mediaEngine, 17, uintptr(math.Float64bits(seconds))),
	)
}

func mediaPositionMs() int {
	seconds := comCallDoubleResult(mediaEngine, 16)
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
		return 0
	}
	return int(seconds*1000.0 + 0.5)
}

func mediaDurationMs() int {
	seconds := comCallDoubleResult(mediaEngine, 19)
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
		return 0
	}
	return int(seconds*1000.0 + 0.5)
}

func mediaSetVolume(v int) error {
	if mediaEngine == nil {
		return nil
	}
	if v < 0 {
		v = 0
	}
	if v > 1000 {
		v = 1000
	}
	muted := uintptr(0)
	if v == 0 {
		muted = 1
	}
	if err := hresultError("IMFMediaEngine::SetMuted", comCall(mediaEngine, 35, muted)); err != nil {
		return err
	}
	level := float64(v) / 1000.0
	return hresultError(
		"IMFMediaEngine::SetVolume",
		comCall(mediaEngine, 37, uintptr(math.Float64bits(level))),
	)
}
