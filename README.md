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

## Future whole-house audio integration

This player is planned to become one controller/client for the synchronized house-audio system while preserving its current standalone behavior.

The permanent backend is now planned to run on the existing Raspberry Pi that already owns the music files. The Pi will read the library locally, use MPD for the one shared playback session, and use Snapserver for synchronized distribution. The Windows app remains a controller and optional renderer rather than the authority for the house session.

The mode should be selected automatically:

- **HOUSE** — the PC discovers and verifies the house-audio service directly on the local home LAN. The UI controls the **one shared house playback session** and the PC may also act as a synchronized renderer.
- **STANDALONE** — the house service is not present on the local LAN, so the program behaves exactly as it does today using Media Foundation and normal local/mapped/UNC files.

Detection must be based on the local network, **not specifically on Wi-Fi**. A hardwired PC on home Ethernet is just as much a HOUSE client as a laptop on home Wi-Fi. Preferred detection is mDNS/DNS-SD plus a short LAN handshake, with a reserved LAN address only as a fallback.

**Tailscale/VPN reachability alone must not trigger HOUSE mode.** If a laptop is away from home but can route back through Tailscale, it remains STANDALONE. Any fixed-address fallback should verify that the route is through a normal LAN interface rather than a VPN/tunnel.

There is no separate local music session inside the house. One active output simply means the shared house session currently has one renderer; powering up another output makes it join the same song at the current timestamp.

The existing folder-first UI remains authoritative as a control model: **folders are playlists**.

Related projects:

- [house-audio-server](https://github.com/oolah10293/house-audio-server) — Raspberry Pi MPD/Snapserver backend plus control/discovery layer
- [house-audio-esp32](https://github.com/oolah10293/house-audio-esp32) — ESP32-S3 synchronized renderer nodes
- [smb-music-player](https://github.com/oolah10293/smb-music-player) — Android player/controller

Implementation is intentionally deferred until the Raspberry Pi Snapserver path and ESP32-S3 renderer path are proven. See Issue #5 for the current architecture notes.

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
