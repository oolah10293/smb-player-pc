# SMB Player PC

A lightweight Windows music player built around one rule: **the folder is the playlist**.

SMB Player PC uses the normal Windows filesystem. It does **not** implement SMB itself. Music can live on a local drive, mapped network drive, or UNC/network path and the application treats it as ordinary files and folders.

## Current version

**v0.2.7**

Current working features:

- Folder-first browsing
- Native Windows **Browse** folder picker
- Manual mapped-drive / UNC path entry
- **Play Folder**
- Double-click a track to start playback
- Previous / Play-Pause / Next
- Automatic next-track playback
- Seek control
- Volume control
- A-Z, Z-A, New-to-Old, and Old-to-New sorting
- Last-folder memory
- Async folder enumeration so slow/dead network paths do not freeze the UI
- Dark stereo-component-inspired interface
- Wide 31-band segmented spectrum analyzer
- Real analyzer data from Windows WASAPI shared-mode endpoint loopback capture
- 4096-point FFT with logarithmically spaced frequency bands from roughly 35 Hz to 16 kHz
- Common slow display AGC so the analyzer stays visually active without changing the audio
- Fast bar response with short peak-hold markers
- Green / amber / red LED-style segments
- Analyzer is effectively independent of the player's volume setting until mute, which is desirable for the visual display
- Existing VU-style application icon retained as an homage to the original analog-meter versions

## Playback / network status

Current playback is still handled by Windows MCI. The v0.2.7 spectrum analyzer is a **separate visualization branch** and does not replace or modify the playback backend.

Remote/network playback has been confirmed working again. The earlier v0.2.4 failure was not reproduced later, and VLC also showed buffering during the same degraded network conditions, so that incident is retained as historical evidence rather than treated as an active player regression.

## Design direction

The analyzer is now the visual centerpiece.

The goal is **visually interesting without being visually annoying**:
- movement should clearly correlate with the music
- different frequency regions should move independently
- the display should stay usefully occupied across quiet and loud material
- literal meter calibration is not important
- playback audio must remain untouched

The current implementation listens to the default Windows render endpoint, so other computer audio intentionally appears on the analyzer too.

The VU-style app icon stays.

## Build

The project is written in Go using the native Windows API plus:
- `github.com/degubites/go-wca` for WASAPI/Core Audio access
- `github.com/go-ole/go-ole` for COM support

To cross-compile from a machine with Go installed:

```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o SMBPlayerPC.exe .
```

On Windows PowerShell:

```powershell
go build -ldflags="-H windowsgui" -o SMBPlayerPC.exe .
```

GitHub Actions also builds a Windows executable artifact on every push to `main` and on manual workflow dispatch.

## Project philosophy

This is deliberately **not** a music-library database application. The filesystem remains the source of truth:

```text
Music/
├── CDs/
├── MP3s/
└── Rap/
```

Your folders are your playlists.
