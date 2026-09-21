# Project Status — 2026-09-21

## Current version

**v0.3.1 — polish candidate**

## Confirmed v0.3.0 backend result

The Media Foundation backend solved the key MCI failure without losing the core network-use case.

Confirmed by runtime test:
- original unmodified `The Raconteurs - Level.mp3` plays
- `The Red Jumpsuit Apparatus - Face Down.mp3` plays
- both were played directly from the network drive
- user reported the version overall works great

## v0.3.1 changes

### Seek behavior

The native Windows trackbar page-jump behavior made clicks away from the thumb move in large blocks.

v0.3.1 intercepts trackbar page clicks, maps the mouse position directly to the seek range, and seeks to that exact percentage. Dragging the thumb remains continuous.

The **volume slider is intentionally unchanged**.

### Metadata

Embedded metadata is now loaded asynchronously with `github.com/dhowden/tag`.

Display priority:
1. embedded Title
2. optional Artist appended as `TITLE • ARTIST`
3. filename stem as fallback

Network metadata reads are generation-checked so a delayed read for a previous song cannot overwrite the current Now Playing display.

### Status lamp

The old decorative POWER lamp is now functional:

- **green** — actively playing
- **yellow** — loading or Media Foundation WAITING/STALLED buffering state
- **red** — ready/stopped, paused, or error

The text status also reports LOADING or BUFFERING where appropriate.

### Resume / recovery ramp

A smooth approximately **1.5 second** fade-in is applied:
- when manually resuming from pause
- when playback recovers from a Media Foundation WAITING/STALLED buffering event

The volume slider does not move during the fade. Its selected volume remains the target. If the user touches the volume control during a ramp, the ramp is cancelled and the user's selected volume takes control immediately.

## Preserve

- Media Foundation IMFMediaEngine playback
- folder = playlist
- mapped-drive / UNC filesystem behavior
- native Browse
- manual path + GO
- asynchronous folder enumeration
- Shuffle behavior
- current dark stereo-component UI
- current 31-band analyzer
- VU-style app icon
- single-EXE distribution

## v0.3.1 runtime checks

- [ ] Metadata displays correctly on tagged tracks.
- [ ] Filename fallback works on untagged tracks.
- [ ] Clicking empty space on Seek jumps directly to the clicked position.
- [ ] Thumb dragging still seeks normally.
- [ ] Volume slider behavior is unchanged.
- [ ] Lamp is green while playing.
- [ ] Lamp is red while paused/stopped.
- [ ] Lamp becomes yellow during observable loading/buffering.
- [ ] Resume fade is about 1.5 seconds and does not move the volume slider.
- [ ] Buffer-recovery fade behaves similarly.
- [ ] Network playback remains stable.
- [ ] Shuffle remains correct.
- [ ] Spectrum analyzer remains correct.
