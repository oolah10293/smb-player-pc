//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dhowden/tag"
)

var (
	trackMetadataMu     sync.RWMutex
	trackMetadataTitle  string
	trackMetadataArtist string
	trackMetadataGen    uint32
)

func clearTrackMetadata() {
	atomic.AddUint32(&trackMetadataGen, 1)
	trackMetadataMu.Lock()
	trackMetadataTitle = ""
	trackMetadataArtist = ""
	trackMetadataMu.Unlock()
}

func requestTrackMetadata(path string) {
	gen := atomic.AddUint32(&trackMetadataGen, 1)

	trackMetadataMu.Lock()
	trackMetadataTitle = ""
	trackMetadataArtist = ""
	trackMetadataMu.Unlock()

	go func() {
		var title, artist string
		f, err := os.Open(path)
		if err == nil {
			m, readErr := tag.ReadFrom(f)
			_ = f.Close()
			if readErr == nil {
				title = strings.TrimSpace(m.Title())
				artist = strings.TrimSpace(m.Artist())
			}
		}

		// A slow network/tag read from an old track must never overwrite
		// metadata for the track that is now playing.
		if gen != atomic.LoadUint32(&trackMetadataGen) {
			return
		}
		trackMetadataMu.Lock()
		trackMetadataTitle = title
		trackMetadataArtist = artist
		trackMetadataMu.Unlock()

		if mainHwnd != 0 {
			pPostMessageW.Call(uintptr(mainHwnd), WM_METADATA_DONE, uintptr(gen), 0)
		}
	}()
}

func metadataGenerationMatches(gen uint32) bool {
	return gen == atomic.LoadUint32(&trackMetadataGen)
}

func nowPlayingDisplay(current string) string {
	if current == "" {
		return "NO TRACK LOADED"
	}
	if strings.HasPrefix(current, "CAN'T OPEN:") ||
		strings.HasPrefix(current, "AUDIO INIT ERROR:") ||
		strings.HasPrefix(current, "FOLDER ERROR:") {
		return strings.ToUpper(current)
	}

	fallback := strings.TrimSuffix(filepath.Base(current), filepath.Ext(current))

	trackMetadataMu.RLock()
	title := trackMetadataTitle
	artist := trackMetadataArtist
	trackMetadataMu.RUnlock()

	if title == "" {
		title = fallback
	}
	if artist != "" {
		return strings.ToUpper(title + "  •  " + artist)
	}
	return strings.ToUpper(title)
}
