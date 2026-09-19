# SMB Player PC

A lightweight Windows music player built around one rule: **the folder is the playlist**.

SMB Player PC uses the normal Windows filesystem. It does **not** implement SMB itself. Music can live on a local drive, mapped network drive, or UNC/network path and the application treats it as ordinary files and folders.

## Current version

**v0.2.6**

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
- Vintage receiver-style interface
- Dual analog VU meters driven by the actual Windows output-channel levels
- Double-buffered meter rendering to prevent flicker
- Backlit meter styling, fluorescent-style track/status display, power lamp, and labeled SEEK/VOLUME controls

## Current known regression

**v0.2.4 can browse network music folders but currently fails to open the remote files for playback.**

Observed behavior:

- Remote folders and MP3 files enumerate correctly.
- **Play Folder** on the remote location reports `CAN'T OPEN` essentially immediately.
- Double-clicking a single remote MP3 and waiting still reports `CAN'T OPEN` after roughly 3–5 seconds.
- A local Desktop copy of `My Name Is.mp3` plays successfully.
- **v0.1.2 is the known-good remote-playback baseline:** it successfully played a song from a hard drive at one house on a PC roughly three hours away.

That makes this a playback/open regression rather than a basic network-directory-access failure. See the open GitHub issue for the test record.

## Design direction

The interface is intentionally moving toward a **late-1970s / early-1980s stereo receiver** rather than a modern streaming-app design. Album art is not a priority; the animated VU meters are intended to be the visual centerpiece.

The next visual work is focused on making the meters and controls feel like actual hardware: correct proportions, convincing backlighting, more physical bezels/controls, and stronger stereo-component personality without sacrificing the practical file browser.

## Build

The project is written in Go using the native Windows API. No external Go packages are currently required.

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
