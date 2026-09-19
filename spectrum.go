//go:build windows

package main

import (
	"math"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/degubites/go-wca/pkg/wca"
	"github.com/go-ole/go-ole"
)

const (
	spectrumBandCount = 31
	spectrumFFTSize   = 4096
)

var (
	spectrumMu     sync.Mutex
	spectrumLevels [spectrumBandCount]float64
	spectrumPeaks  [spectrumBandCount]float64
	spectrumOnline bool
	spectrumStopCh chan struct{}

	spectrumBrushBackground HBRUSH
	spectrumBrushGreenOff   HBRUSH
	spectrumBrushGreenOn    HBRUSH
	spectrumBrushAmberOff   HBRUSH
	spectrumBrushAmberOn    HBRUSH
	spectrumBrushRedOff     HBRUSH
	spectrumBrushRedOn      HBRUSH
)

type spectrumProcessor struct {
	sampleRate float64
	ring       [spectrumFFTSize]float64
	write      int
	filled     bool
	window     [spectrumFFTSize]float64
	fft        []complex128
	lastFFT    time.Time
	refDB      float64
	refInit    bool
	display    [spectrumBandCount]float64
	peaks      [spectrumBandCount]float64
	peakHold   [spectrumBandCount]int
}

func newSpectrumProcessor(sampleRate float64) *spectrumProcessor {
	p := &spectrumProcessor{
		sampleRate: sampleRate,
		fft:        make([]complex128, spectrumFFTSize),
	}
	for i := 0; i < spectrumFFTSize; i++ {
		p.window[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(spectrumFFTSize-1))
	}
	return p
}

func (p *spectrumProcessor) push(v float64) {
	p.ring[p.write] = v
	p.write++
	if p.write >= spectrumFFTSize {
		p.write = 0
		p.filled = true
	}
}

func (p *spectrumProcessor) maybeAnalyze() {
	if !p.filled {
		return
	}
	now := time.Now()
	if !p.lastFFT.IsZero() && now.Sub(p.lastFFT) < 25*time.Millisecond {
		return
	}
	p.lastFFT = now
	p.analyze()
}

func (p *spectrumProcessor) analyze() {
	for i := 0; i < spectrumFFTSize; i++ {
		s := p.ring[(p.write+i)%spectrumFFTSize]
		p.fft[i] = complex(s*p.window[i], 0)
	}
	fftInPlace(p.fft)

	const (
		minFreq      = 35.0
		maxFreq      = 16000.0
		dynamicRange = 46.0
	)
	var db [spectrumBandCount]float64
	frameMax := -160.0
	logSpan := math.Log(maxFreq / minFreq)

	for band := 0; band < spectrumBandCount; band++ {
		low := minFreq * math.Exp(logSpan*float64(band)/float64(spectrumBandCount))
		high := minFreq * math.Exp(logSpan*float64(band+1)/float64(spectrumBandCount))
		loBin := int(math.Ceil(low * float64(spectrumFFTSize) / p.sampleRate))
		hiBin := int(math.Floor(high * float64(spectrumFFTSize) / p.sampleRate))
		if loBin < 1 {
			loBin = 1
		}
		maxBin := spectrumFFTSize/2 - 1
		if hiBin > maxBin {
			hiBin = maxBin
		}
		if hiBin < loBin {
			hiBin = loBin
		}
		var power float64
		count := 0
		for k := loBin; k <= hiBin && k < spectrumFFTSize/2; k++ {
			re, im := real(p.fft[k]), imag(p.fft[k])
			power += re*re + im*im
			count++
		}
		if count == 0 {
			db[band] = -160
			continue
		}
		power /= float64(count) * float64(spectrumFFTSize*spectrumFFTSize)
		db[band] = 10 * math.Log10(power+1e-16)
		if db[band] > frameMax {
			frameMax = db[band]
		}
	}

	silent := frameMax < -95
	if !silent {
		desiredRef := frameMax + 3.0
		if !p.refInit {
			p.refDB = desiredRef
			p.refInit = true
		} else if desiredRef > p.refDB {
			// Back away quickly when the program suddenly gets louder.
			p.refDB += (desiredRef - p.refDB) * 0.22
		} else {
			// Recover sensitivity slowly so quiet passages do not make
			// the whole display visibly "breathe".
			p.refDB += (desiredRef - p.refDB) * 0.010
		}
	}

	for i := 0; i < spectrumBandCount; i++ {
		target := 0.0
		if !silent && p.refInit {
			target = (db[i] - (p.refDB - dynamicRange)) / dynamicRange
			if target < 0 {
				target = 0
			}
			if target > 1 {
				target = 1
			}
			// Slight visual expansion of the useful middle. This is a
			// display, not a calibrated instrument.
			target = math.Pow(target, 0.82)
		}

		if target > p.display[i] {
			p.display[i] += (target - p.display[i]) * 0.62
		} else {
			p.display[i] += (target - p.display[i]) * 0.18
		}
		if p.display[i] < 0.002 {
			p.display[i] = 0
		}

		if p.display[i] >= p.peaks[i] {
			p.peaks[i] = p.display[i]
			p.peakHold[i] = 16
		} else if p.peakHold[i] > 0 {
			p.peakHold[i]--
		} else {
			p.peaks[i] -= 0.025
			if p.peaks[i] < p.display[i] {
				p.peaks[i] = p.display[i]
			}
		}
	}

	spectrumMu.Lock()
	spectrumLevels = p.display
	spectrumPeaks = p.peaks
	spectrumMu.Unlock()
}

func fftInPlace(a []complex128) {
	n := len(a)
	j := 0
	for i := 1; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j &= ^bit
		}
		j |= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	for length := 2; length <= n; length <<= 1 {
		angle := -2 * math.Pi / float64(length)
		wlen := complex(math.Cos(angle), math.Sin(angle))
		half := length >> 1
		for i := 0; i < n; i += length {
			w := complex(1.0, 0.0)
			for k := 0; k < half; k++ {
				u := a[i+k]
				v := a[i+k+half] * w
				a[i+k] = u + v
				a[i+k+half] = u - v
				w *= wlen
			}
		}
	}
}

func initSpectrumGraphics() {
	if spectrumBrushBackground != 0 {
		return
	}
	spectrumBrushBackground = HBRUSH(must1(pCreateSolidBrush.Call(color(3, 10, 7))))
	spectrumBrushGreenOff = HBRUSH(must1(pCreateSolidBrush.Call(color(7, 37, 22))))
	spectrumBrushGreenOn = HBRUSH(must1(pCreateSolidBrush.Call(color(28, 235, 112))))
	spectrumBrushAmberOff = HBRUSH(must1(pCreateSolidBrush.Call(color(47, 36, 8))))
	spectrumBrushAmberOn = HBRUSH(must1(pCreateSolidBrush.Call(color(242, 191, 48))))
	spectrumBrushRedOff = HBRUSH(must1(pCreateSolidBrush.Call(color(48, 10, 7))))
	spectrumBrushRedOn = HBRUSH(must1(pCreateSolidBrush.Call(color(246, 55, 37))))
}

func shutdownSpectrumGraphics() {
	for _, b := range []HBRUSH{
		spectrumBrushBackground,
		spectrumBrushGreenOff, spectrumBrushGreenOn,
		spectrumBrushAmberOff, spectrumBrushAmberOn,
		spectrumBrushRedOff, spectrumBrushRedOn,
	} {
		if b != 0 {
			pDeleteObject.Call(uintptr(b))
		}
	}
	spectrumBrushBackground = 0
}

func spectrumBrushForSegment(seg, segments int, on bool) HBRUSH {
	f := float64(seg+1) / float64(segments)
	if f > 0.875 {
		if on {
			return spectrumBrushRedOn
		}
		return spectrumBrushRedOff
	}
	if f > 0.69 {
		if on {
			return spectrumBrushAmberOn
		}
		return spectrumBrushAmberOff
	}
	if on {
		return spectrumBrushGreenOn
	}
	return spectrumBrushGreenOff
}

func spectrumSnapshot() ([spectrumBandCount]float64, [spectrumBandCount]float64, bool) {
	spectrumMu.Lock()
	defer spectrumMu.Unlock()
	return spectrumLevels, spectrumPeaks, spectrumOnline
}

func drawSpectrumAnalyzer(hdc HDC, r RECT) {
	initSpectrumGraphics()
	fill(hdc, r, brushBlack)
	bezel := RECT{r.Left + 2, r.Top + 2, r.Right - 2, r.Bottom - 2}
	fill(hdc, bezel, brushPanel)
	drawRaisedFrame(hdc, bezel, false)
	face := RECT{r.Left + 9, r.Top + 9, r.Right - 9, r.Bottom - 9}
	fill(hdc, face, spectrumBrushBackground)
	drawInsetFrame(hdc, face)

	levels, peaks, online := spectrumSnapshot()

	const (
		segments = 16
		gapX     = int32(4)
		gapY     = int32(2)
	)
	bars := RECT{face.Left + 12, face.Top + 9, face.Right - 12, face.Bottom - 9}
	barW := (bars.Right - bars.Left - gapX*int32(spectrumBandCount-1)) / spectrumBandCount
	if barW < 3 {
		barW = 3
	}
	segH := (bars.Bottom - bars.Top - gapY*int32(segments-1)) / segments
	if segH < 2 {
		segH = 2
	}

	for band := 0; band < spectrumBandCount; band++ {
		x := bars.Left + int32(band)*(barW+gapX)
		lit := int(math.Round(levels[band] * float64(segments)))
		if lit < 0 {
			lit = 0
		}
		if lit > segments {
			lit = segments
		}
		peakSeg := int(math.Ceil(peaks[band] * float64(segments)))
		if peakSeg < 1 {
			peakSeg = 0
		}
		if peakSeg > segments {
			peakSeg = segments
		}
		for seg := 0; seg < segments; seg++ {
			yBottom := bars.Bottom - int32(seg)*(segH+gapY)
			sr := RECT{x, yBottom - segH, x + barW, yBottom}
			on := seg < lit || seg == peakSeg-1
			fill(hdc, sr, spectrumBrushForSegment(seg, segments, on))
		}
	}

	if !online {
		drawText(hdc, "AUDIO LOOPBACK STARTING", face, DT_CENTER|DT_VCENTER|DT_SINGLELINE, color(82, 105, 89), tinyFont)
	}
}

func startSpectrumAnalyzer() {
	initSpectrumGraphics()
	if spectrumStopCh != nil {
		return
	}
	spectrumStopCh = make(chan struct{})
	stop := spectrumStopCh
	go spectrumSupervisor(stop)
}

func stopSpectrumAnalyzer() {
	if spectrumStopCh != nil {
		close(spectrumStopCh)
		spectrumStopCh = nil
	}
	shutdownSpectrumGraphics()
}

func spectrumSupervisor(stop <-chan struct{}) {
	for {
		if spectrumStopped(stop) {
			return
		}
		_ = runSpectrumCapture(stop)
		spectrumMu.Lock()
		spectrumOnline = false
		spectrumMu.Unlock()
		select {
		case <-stop:
			return
		case <-time.After(time.Second):
		}
	}
}

func spectrumStopped(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

func runSpectrumCapture(stop <-chan struct{}) (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return err
	}
	defer ole.CoUninitialize()

	var enumerator *wca.IMMDeviceEnumerator
	if err = wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &enumerator); err != nil {
		return err
	}
	defer enumerator.Release()

	var device *wca.IMMDevice
	if err = enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &device); err != nil {
		return err
	}
	defer device.Release()

	var client *wca.IAudioClient
	if err = device.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &client); err != nil {
		return err
	}
	defer client.Release()

	var mix *wca.WAVEFORMATEX
	if err = client.GetMixFormat(&mix); err != nil {
		return err
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(mix)))

	channels := mix.NChannels
	if channels == 0 {
		channels = 2
	}
	rate := mix.NSamplesPerSec
	if rate == 0 {
		rate = 48000
	}
	blockAlign := uint16(channels * 4)
	captureFormat := wca.WAVEFORMATEX{
		WFormatTag:      3, // WAVE_FORMAT_IEEE_FLOAT
		NChannels:       channels,
		NSamplesPerSec:  rate,
		NAvgBytesPerSec: rate * uint32(blockAlign),
		NBlockAlign:     blockAlign,
		WBitsPerSample:  32,
		CbSize:          0,
	}

	flags := uint32(wca.AUDCLNT_STREAMFLAGS_LOOPBACK | wca.AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM | wca.AUDCLNT_STREAMFLAGS_SRC_DEFAULT_QUALITY)
	if err = client.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, flags, wca.REFERENCE_TIME(100*10000), 0, &captureFormat, nil); err != nil {
		return err
	}

	var capture *wca.IAudioCaptureClient
	if err = client.GetService(wca.IID_IAudioCaptureClient, &capture); err != nil {
		return err
	}
	defer capture.Release()

	if err = client.Start(); err != nil {
		return err
	}
	defer client.Stop()

	processor := newSpectrumProcessor(float64(rate))
	spectrumMu.Lock()
	spectrumOnline = true
	spectrumMu.Unlock()

	for {
		if spectrumStopped(stop) {
			return nil
		}

		var packetFrames uint32
		if err = capture.GetNextPacketSize(&packetFrames); err != nil {
			return err
		}
		if packetFrames == 0 {
			time.Sleep(4 * time.Millisecond)
			continue
		}

		for packetFrames > 0 {
			var data *byte
			var frames uint32
			var bufferFlags uint32
			var devicePosition, qpcPosition uint64
			if err = capture.GetBuffer(&data, &frames, &bufferFlags, &devicePosition, &qpcPosition); err != nil {
				return err
			}

			silent := bufferFlags&wca.AUDCLNT_BUFFERFLAGS_SILENT != 0
			if frames > 0 {
				if silent || data == nil {
					for f := uint32(0); f < frames; f++ {
						processor.push(0)
					}
				} else {
					base := uintptr(unsafe.Pointer(data))
					for f := uint32(0); f < frames; f++ {
						frameBase := base + uintptr(f)*uintptr(captureFormat.NBlockAlign)
						sum := 0.0
						for ch := uint16(0); ch < channels; ch++ {
							sample := *(*float32)(unsafe.Pointer(frameBase + uintptr(ch)*4))
							sum += float64(sample)
						}
						processor.push(sum / float64(channels))
					}
				}
			}

			if err = capture.ReleaseBuffer(frames); err != nil {
				return err
			}
			processor.maybeAnalyze()

			if err = capture.GetNextPacketSize(&packetFrames); err != nil {
				return err
			}
		}
	}
}
