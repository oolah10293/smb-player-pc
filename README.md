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

v0.2.7 still uses Windows MCI for playback. The spectrum analyzer is a **separate visualization branch** and does not replace or modify the playback backend.

Remote/network playback itself has been confirmed working again. However, MCI has now exposed a more serious local-file compatibility problem: some otherwise valid MP3 files are rejected with `CAN'T OPEN`.

A controlled test isolated one reproducible case:

- Original `The Raconteurs - Level.mp3`: fails every time in SMB Player.
- Copying that original file to the local Desktop does **not** fix it, ruling out SMB/network access.
- The original file had a roughly 195 KB ID3 block dominated by embedded album art.
- A test copy with only the embedded cover-art frame removed had a roughly 4 KB ID3 block.
- The MP3 audio payload of the original and stripped test copy was byte-for-byte identical.
- The stripped test copy played successfully.

In spot testing the same folder, roughly half the sampled MP3 files opened and roughly half returned `CAN'T OPEN`, with failures reproducible by file. This makes MCI playback compatibility a release-blocking reliability problem rather than an isolated bad track.

**Decision:** v0.2.7 is the last planned MCI-based build. The next backend should use a modern Windows playback path while preserving the existing browser, network-path behavior, UI, and WASAPI analyzer.

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
