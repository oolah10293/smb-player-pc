# Project Status — 2026-09-20

## Current version

**v0.3.0 — new playback-backend candidate**

The source compiles successfully in GitHub Actions as a 64-bit Windows GUI executable. Runtime validation on Windows is still required before the Media Foundation migration is considered complete.

## What remains preserved from v0.2.7

- Folder-first browser and folder-as-playlist model
- Native Windows Browse folder picker
- Manual mapped-drive / UNC path entry
- Async directory/network enumeration
- Last-folder memory
- Existing transport layout
- Current dark stereo-component visual direction
- Wide 31-band WASAPI loopback spectrum analyzer
- Current analyzer FFT / common display AGC behavior
- Existing VU-style application icon

## Playback backend change

The MCI playback path has been removed.

v0.3.0 now uses Windows Media Foundation's **IMFMediaEngine** in audio-only mode. Media Foundation is started on the Win32 UI/COM thread, an IMFMediaEngineNotify callback posts playback events back to the application window, and local/mapped/UNC paths are converted to file URLs for SetSource.

The backend now handles:
- source open/load
- play
- pause/resume
- seek
- duration/position
- volume/mute
- end-of-track notification

The existing WASAPI analyzer remains a separate endpoint-loopback path.

## Why MCI was removed

The decision came from a reproducible controlled test rather than a generic compatibility concern.

Observed spot test:
- roughly half of the sampled MP3s in one folder played
- roughly half returned CAN'T OPEN
- failures were reproducible by file
- `The Red Jumpsuit Apparatus - Face Down.mp3` played every time
- `The Raconteurs - Level.mp3` failed every time

Controlled A/B test on `Level.mp3`:
1. Original failed from the normal music location.
2. The same original copied to the local Desktop still failed, ruling out SMB/network access.
3. Original ID3 block was about 195,352 bytes, dominated by embedded PNG artwork.
4. A test copy had only the artwork frame removed; its ID3 block dropped to about 4,160 bytes.
5. The MP3 audio payload remained byte-for-byte identical.
6. The stripped copy played successfully.

Conclusion: MCI was rejecting the MP3 because of container/tag/header handling, not because of the encoded audio or network path. No guessed tag-size threshold is used in the replacement.

## Failure-path fixes included in v0.3.0

- A real playback/open failure now resets elapsed and total time to 0:00 / 0:00.
- The seek control is reset on failure.
- PLAY after a failed open no longer blindly starts playlist item 0.
- Media Engine source changes use a LOADING state.
- Pause/end events generated while a replacement source is still loading are ignored so an old source cannot easily disturb the new selection.

## Shuffle added

v0.3.0 adds a **SHUFFLE: OFF / SHUFFLE: ON** control.

Behavior:
- Play Folder starts at a random track when Shuffle is ON.
- Every track is visited once in the initial shuffle cycle.
- Subsequent cycles are re-randomized.
- A cycle boundary avoids an immediate repeat of the track that just played.
- Previous follows actual shuffle history.
- After going backward, Next follows forward history before taking a new shuffled choice.
- With Shuffle OFF, normal sorted folder order is unchanged.

## v0.3.0 runtime acceptance tests

Still to validate on Windows:

- [ ] Original unmodified `The Raconteurs - Level.mp3` plays.
- [ ] `The Red Jumpsuit Apparatus - Face Down.mp3` still plays.
- [ ] Local playback works.
- [ ] Mapped-drive playback works.
- [ ] UNC playback works.
- [ ] Pause/resume works.
- [ ] Seek works.
- [ ] Volume/mute works.
- [ ] Previous/Next work.
- [ ] Track-end auto-advance works.
- [ ] Shuffle works as intended.
- [ ] Spectrum analyzer still works and remains visually correct.
- [ ] Application remains responsive.

The file browser still recognizes MP3, WAV, WMA, M4A, AAC, FLAC, and OGG. Actual decode support through Media Foundation should be recorded only after runtime testing.

## Next functional polish after backend validation

- Embedded title/artist metadata in the Now Playing area.
- Clean filename fallback/recentering when metadata is absent.
- Any analyzer adjustment only if actual use exposes a specific problem.

## Preserve

- folder = playlist
- normal Windows local/mapped/UNC filesystem access
- native Browse
- manual path + GO
- async folder enumeration
- current UI identity
- current 31-band analyzer
- VU-style app icon
- single-EXE distribution
