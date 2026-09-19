# Project Status — 2026-09-19

## Current version

**v0.2.7**

## What is confirmed working

- Application launches and remains responsive.
- Native folder browsing works.
- Local and remote/network folders enumerate correctly.
- Local playback works.
- Remote/network playback has been confirmed working in later testing.
- Folder-as-playlist behavior works.
- Transport controls, seek, volume, and auto-advance are implemented.
- The Win32 GUI remains pinned to one OS thread to avoid the original freeze.
- Directory/network enumeration remains asynchronous.
- Startup does not automatically scan the remembered folder.
- Wide 31-band spectrum analyzer is working from real Windows output audio.
- Analyzer bands visibly respond differently across the frequency spectrum rather than acting like duplicated level meters.
- WASAPI endpoint loopback captures the default Windows render mix, so other PC audio also appears by design.
- Analyzer visual level is effectively independent of SMB Player's volume control until mute.
- The analyzer uses common adaptive display gain so it stays visually active without altering the actual audio.
- The existing VU-style app icon is retained intentionally as an homage to the original dual-meter design.

## Current UI direction

The spectrum analyzer is now the preferred visual centerpiece over the dual analog VU meters.

Design goal: **visually interesting without offending**.

That means:
- motion must correlate with the actual song
- bass/mid/treble regions should behave independently
- the display should use a satisfying amount of its range
- it should not sit pegged or dead
- literal calibration is not important
- no visualization processing may alter the playback audio

The current 31-band / 16-segment green-amber-red analyzer is a successful first implementation and should be preserved while it is used for a while before unnecessary tuning.

## Playback architecture

Playback remains Windows MCI.

The spectrum analyzer does **not** replace the media backend. It is a separate side path:

```text
MCI playback -> Windows audio -> speakers

Windows render endpoint -> WASAPI loopback -> PCM -> FFT -> 31 display bands
```

This deliberately avoids destabilizing the currently working player just to obtain visualization data.

## Network-playback history

A remote playback failure was observed in v0.2.4 while remote directories still enumerated normally. Later:
- network playback worked again
- no intentional playback/network-code change explained the recovery
- VLC also buffered/chopped on the same remote path during the degraded session

Therefore the v0.2.4 incident is preserved as historical evidence but is not considered an active reproducible player regression.

## Next requested functional tweaks

- Add Shuffle while preserving folder-as-playlist behavior.
- Add embedded track metadata to the Now Playing area when practical.
- Fall back cleanly to filename when metadata is missing.
- Keep the current analyzer behavior unless runtime use exposes a specific problem.

## Preserve

- MCI playback architecture
- working network playback behavior
- native Browse button
- manual path + GO
- async folder enumeration
- current dark stereo-component visual direction
- current spectrum analyzer concept
- VU-style app icon
