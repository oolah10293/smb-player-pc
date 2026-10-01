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

Detection must be based on the local network, **not specifically on Wi-Fi**. A hardwired PC on home Ethernet is just as much a HOUSE client as a laptop on home Wi-Fi.

The cross-project authority rule remains simple: HOUSE requires evidence that the configured Pi LAN address belongs to the **physically attached non-VPN Ethernet/Wi-Fi LAN**, plus the expected Pi identity. Normal controller/audio traffic does not have to be forcibly bound to that physical interface if the platform's VPN model makes that unreliable. Android v0.4.0 proved this distinction matters: explicit physical-`Network` transport stalled with Tailscale enabled even though normal routing could still reach the Pi.

**Tailscale/VPN reachability alone must not trigger HOUSE mode.** If a laptop is away from home but can route back through Tailscale, it remains STANDALONE. Windows may implement the same physical-presence authority with platform-appropriate interface/route inspection rather than copying Android socket mechanics.

There is no separate local music session inside the house. One active output simply means the shared house session currently has one renderer; powering up another output makes it join the same song at the current timestamp.

The shared session/output contract is maintained in [house-audio-server/docs/SESSION_BEHAVIOR.md](https://github.com/oolah10293/house-audio-server/blob/main/docs/SESSION_BEHAVIOR.md). Its 2026-09-30 clarification keeps transport commands separate from local renderer eligibility. Android's Bluetooth requirement and SMB Bluetooth lifecycle are phone-specific; they do not impose Bluetooth-only rendering or new standalone behavior on this Windows player.

The existing folder-first UI remains authoritative as a control model: **folders are playlists**.

Related projects:

- [house-audio-server](https://github.com/oolah10293/house-audio-server) — Raspberry Pi MPD/Snapserver backend plus control/discovery layer
- [house-audio-esp32](https://github.com/oolah10293/house-audio-esp32) — ESP32-S3 synchronized renderer nodes
- [smb-music-player](https://github.com/oolah10293/smb-music-player) — Android player/controller

The Raspberry Pi/Snapserver path and ESP32 renderer architecture are now proven well beyond the original prerequisite: two independent XIAO ESP32-S3 + PCM5102A renderers have played audibly in sync through different downstream audio systems, and passive-radio power-on/rejoin behavior works without a phone. Windows HOUSE implementation is still deferred until the shared controller-presence/output-state contract and remaining server session details are ready. See Issue #5 for the current architecture notes.

### Current house-audio proof relevant to Windows

The house backend now has real runtime proof for:

- shared MPD browse/queue/state/transport control through `house-audio-server`;
- renderer presence through hard power-off/reconnect;
- passive-node auto-start and same-session rejoin;
- passive-radio resume of an existing paused session;
- completed-drain handling through v0.6.2+: MPD may land paused at 0.0 on the next old-queue track, but that is treated as fresh idle; the next passive start loads a newly shuffled configured default rather than resurrecting the old queue;
- two simultaneous ESP32/PCM5102A outputs audibly synchronized;
- unattended renderer diagnostics for an occasional few-second single-node dropout still under investigation.

This means the future PC work is no longer blocked on proving the central audio architecture. It is primarily a HOUSE controller/client integration problem plus, if desired, adding a synchronized PC renderer.

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


### v0.7.0 passive-default server proof

The shared server now has a field-proven persisted passive default selector:

- current default can be read as `MP3s` or `Rap`;
- changing the setting does not disturb active playback;
- after a completed drain, the next passive S3 session uses the saved choice;
- a real test changed `MP3s` -> `Rap`, left the current song untouched, then after about ten minutes with the final S3 off, the next S3 power-on started a fresh Rap session (first observed track: Ludacris — *Southern Hospitality*).

Future Windows HOUSE control should use the same settings API rather than inventing a separate default-playlist mechanism.


### v0.8.0 controller backend status

The shared HOUSE controller/output API is now implemented and deployed on the permanent Pi.

v0.8.0 provides controller attach/heartbeat/detach, 5-second heartbeats with 15-second expiry, durable controller↔renderer ownership, and server policy for muted/unavailable outputs. Initial deployment checks show the existing S3 remains correctly classified as a passive renderer when no controller is attached. Physical controller pause/resume/expiry transitions still need field validation.

This API is shared infrastructure for Android, Windows, and browser clients; future SMB Player PC HOUSE mode should use it rather than inventing its own presence model.

Restart semantics are settled and implemented in server v0.8.1: a `house-audio-server` restart ends the old listening session and normalizes MPD to fresh idle. Live controller leases/output state/session state are not restored. Durable passive-default configuration and controller↔renderer ownership persist. The permanent Pi has field-proven the restart case with one passive S3 already present: startup reached ready and a fresh randomized Rap session started.


### v0.8.1 restart field proof

Server v0.8.1 is now installed on the permanent Pi. A service restart performed while one passive S3 remained powered discarded the old listening session and started a fresh randomized Rap session.

The server snapshot reported `startup.ready: true`, `lastAction: started_default_session`, one passive/audible renderer, zero controllers, and the persisted `Rap` default. This confirms the shared restart contract future Windows HOUSE mode should follow: reconnect with fresh controller state and never attempt to resurrect the pre-restart session.

Physical controller pause/resume/expiry transitions remain pending server acceptance tests.


### Android v0.4.0 integration finding relevant to Windows

The permanent Pi is now running server v0.8.2. The first Android HOUSE field pass exposed an important cross-client distinction:

- physical non-VPN LAN presence decides whether a client is HOUSE;
- ordinary HOUSE traffic can use normal platform routing;
- VPN reachability by itself never proves the client is home.

On Android, binding all HOUSE traffic to the physical network broke updates when Tailscale was enabled even though the Pi remained reachable through normal routing. Future Windows HOUSE code should preserve the **authority rule** without assuming the same interface-binding implementation.

The Android-specific playlist-mute and mute-button-placement corrections do not impose a Windows UI requirement.
