# SMB Player PC

A lightweight Windows music player built around one rule: **the folder is the playlist**.

SMB Player PC uses the normal Windows filesystem. It does **not** implement SMB itself. Music can live on a local drive, mapped network drive, or UNC/network path and the application treats it as ordinary files and folders.

## Current version

**v0.3.0 — Media Foundation backend / Shuffle build**

The Windows executable builds successfully. Runtime validation of the new playback backend is the next step.

Current features:

- Folder-first browsing
- Native Windows **Browse** folder picker
- Manual mapped-drive / UNC path entry
- **Play Folder**
- Double-click a track to start playback
- Previous / Play-Pause / Next
- Automatic next-track playback
- **Shuffle ON/OFF**
- Seek control
- Volume control
- A-Z, Z-A, New-to-Old, and Old-to-New sorting
- Last-folder memory
- Async folder enumeration so slow/dead network paths do not freeze the UI
- Dark stereo-component-inspired interface
- Wide 31-band segmented spectrum analyzer
- Real analyzer data from Windows WASAPI shared-mode endpoint loopback capture
- 4096-point FFT with logarithmically spaced frequency bands from roughly 35 Hz to 16 kHz
- Common slow display AGC
- Fast bar response with short peak-hold markers
- Green / amber / red LED-style segments
- Existing VU-style application icon retained as an homage to the original analog-meter versions

## Playback backend

v0.3.0 removes Windows MCI from the playback path and uses the Windows **Media Foundation Media Engine** (`IMFMediaEngine`) in audio-only mode.

The change was made after a controlled test showed that MCI could reject otherwise valid MP3 files because of their metadata/header layout. The key regression file is the original, unmodified `The Raconteurs - Level.mp3`: MCI rejected it even from the local Desktop, while an otherwise identical copy with only the large embedded artwork removed played normally.

The new Media Foundation backend owns:

- open / play
- pause / resume
- seek
- duration and position
- volume / mute
- end-of-track notification and auto-advance

Local, mapped-drive, and UNC paths are converted to file URLs before being handed to Media Foundation.

The spectrum analyzer remains a separate WASAPI loopback path, so the visualizer was not rewritten as part of the backend change.

### v0.3.0 runtime acceptance tests

The build is complete, but these still need to be checked on a real Windows machine:

- original unmodified `The Raconteurs - Level.mp3`
- `The Red Jumpsuit Apparatus - Face Down.mp3`
- local playback
- mapped-drive / UNC playback
- pause / resume
- seek
- volume
- Previous / Next
- end-of-track auto-advance
- Shuffle
- spectrum analyzer behavior

The browser currently recognizes MP3, WAV, WMA, M4A, AAC, FLAC, and OGG. Actual decode support is intentionally not claimed until each format is runtime-tested through the new backend.

## Shuffle behavior

Shuffle preserves the core **folder = playlist** rule.

- **SHUFFLE: OFF** uses the folder's current sort order.
- **SHUFFLE: ON** chooses a random first track when Play Folder is pressed.
- A shuffle cycle visits every track before starting a new randomized cycle.
- The first track of a new cycle is prevented from immediately repeating the track that just played.
- Previous walks backward through actual shuffle history.
- After going backward, Next walks forward through that history before choosing a new shuffled track.

## Design direction

The analyzer is the visual centerpiece.

The goal is **visually interesting without being visually annoying**:
- movement should clearly correlate with the music
- different frequency regions should move independently
- the display should stay usefully occupied across quiet and loud material
- literal meter calibration is not important
- visualization processing must not alter playback audio

The current implementation listens to the default Windows render endpoint, so other computer audio intentionally appears on the analyzer too.

The VU-style app icon stays.

## Build

The project is written in Go using the native Windows API plus:
- `github.com/degubites/go-wca` for WASAPI/Core Audio access
- `github.com/go-ole/go-ole` for COM support
- Windows Media Foundation system components for playback

To cross-compile from a machine with Go installed:

```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o SMBPlayerPC.exe .
```

On Windows PowerShell:

```powershell
go build -ldflags="-H windowsgui" -o SMBPlayerPC.exe .
```

GitHub Actions builds a Windows executable artifact on every push to `main` and on manual workflow dispatch.

## Project philosophy

This is deliberately **not** a music-library database application. The filesystem remains the source of truth:

```text
Music/
├── CDs/
├── MP3s/
└── Rap/
```

Your folders are your playlists.
