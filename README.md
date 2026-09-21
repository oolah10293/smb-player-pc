# SMB Player PC

A lightweight Windows music player built around one rule: **the folder is the playlist**.

SMB Player PC uses the normal Windows filesystem. It does **not** implement SMB itself. Music can live on a local drive, mapped network drive, or UNC/network path and the application treats it as ordinary files and folders.

## Current version

**v0.3.1 — playback/UI polish build**

v0.3.0 replaced MCI with Windows Media Foundation and added Shuffle. Runtime testing confirmed that the original unmodified `The Raconteurs - Level.mp3` and `The Red Jumpsuit Apparatus - Face Down.mp3` both play successfully directly from the network drive.

v0.3.1 keeps that backend and adds:

- Embedded Title/Artist metadata in the Now Playing display.
- Filename fallback when usable metadata is absent.
- Metadata reads happen asynchronously so network tag reads do not freeze the UI.
- Direct click-to-position behavior on the **seek** bar instead of native wide page jumps.
- Volume slider behavior intentionally remains unchanged.
- The former POWER lamp is now a playback/status lamp:
  - green = actively playing
  - yellow = loading or buffering/waiting for data
  - red = stopped, ready, paused, or error
- Media Foundation WAITING/STALLED events drive the buffering state.
- About a **1.5 second smooth fade-in** when resuming from pause.
- The same fade-in is used when audio resumes after a detected buffering interruption.
- Existing Shuffle, browser, network-path handling, visual design, and 31-band analyzer are preserved.

## Playback backend

Playback uses Windows **Media Foundation IMFMediaEngine** in audio-only mode.

The MCI backend was removed after a controlled A/B test showed that MCI could reject otherwise valid MP3s because of their tag/header layout. The original `Level.mp3` failed even when copied locally, while a copy with only the large embedded artwork removed played. The encoded MP3 audio payload was identical.

The Media Foundation backend handles open/play, pause/resume, seek, duration/position, volume/mute, end-of-track auto-advance, and loading/waiting/stalled playback events.

Local, mapped-drive, and UNC paths are converted to file URLs before being handed to Media Foundation. The spectrum analyzer remains a separate WASAPI loopback path.

## Metadata

Now Playing metadata is read with `github.com/dhowden/tag`, which supports MP3 ID3 tags plus MP4/M4A, OGG, and FLAC metadata.

The display uses `TITLE  •  ARTIST`. If Title is missing, the filename stem remains the fallback. Long display strings are ellipsized instead of running out of the display.

Metadata extraction is asynchronous and generation-checked so a slow network read from an old track cannot overwrite the currently playing track.

## Shuffle behavior

Shuffle preserves the core **folder = playlist** rule. With Shuffle ON, a cycle visits every track before rerandomizing, avoids an immediate repeat across cycle boundaries, and Previous/Next preserve actual shuffle history.

## Design direction

The analyzer remains the visual centerpiece. The VU-style app icon stays as an homage to the original analog-meter versions.

## Build

The project is written in Go using the native Windows API plus:
- `github.com/degubites/go-wca` for WASAPI/Core Audio access
- `github.com/go-ole/go-ole` for COM support
- `github.com/dhowden/tag` for embedded audio metadata
- Windows Media Foundation system components for playback

GitHub Actions builds a Windows executable artifact on every push to `main` and on manual workflow dispatch.

## Project philosophy

This is deliberately **not** a music-library database application. The filesystem remains the source of truth. Your folders are your playlists.
