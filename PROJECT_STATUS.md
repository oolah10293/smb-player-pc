# Project Status — 2026-09-19

## Current version

**v0.2.7**

## What is confirmed working

- Application launches and remains responsive.
- Native folder browsing works.
- Local and remote/network folders enumerate correctly.
- Remote/network playback has been confirmed working for files MCI can open.
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

## Newly confirmed playback blocker

MCI cannot be trusted to open all normal MP3 files in the user's collection.

Observed spot test:
- clicking through tracks and giving each about five seconds produced roughly a 50/50 split between successful playback and `CAN'T OPEN`
- failures were reproducible by file rather than random
- `The Red Jumpsuit Apparatus - Face Down.mp3` played every time
- `The Raconteurs - Level.mp3` failed every time

Controlled A/B test on `Level.mp3`:
1. Original file failed from the music location.
2. The same original file copied to the local Desktop still failed, ruling out SMB/network access.
3. Original ID3 block was about 195,352 bytes, dominated by embedded PNG cover art.
4. A test copy had only the embedded cover-art frame removed; its ID3 block dropped to about 4,160 bytes.
5. The MP3 audio payload remained byte-for-byte identical.
6. The stripped copy played successfully.

Conclusion: the playback failure is in MCI's handling of the MP3 container/tag/header data, not the MP3 audio payload and not the network path. The exact internal MCI limit has not been proven, so do not encode a guessed tag-size threshold as the fix.

This is a deal breaker for a folder player: users must be able to trust that an ordinary MP3 in a folder will play without rewriting its tags.

## Backend decision

**v0.2.7 is the last planned MCI build.**

Next major task: replace MCI playback with a modern Windows playback backend, with Media Foundation as the current preferred direction.

Do not redesign the rest of the application as part of the backend swap.

Acceptance tests for the replacement backend:
- original, unmodified `The Raconteurs - Level.mp3` plays
- `The Red Jumpsuit Apparatus - Face Down.mp3` still plays
- local playback works
- mapped-drive / UNC playback works
- Previous / Play-Pause / Next work
- seek works
- volume works
- end-of-track auto-advance works
- current WASAPI spectrum analyzer still works
- application remains a simple Windows EXE without requiring a separate media-player installation

Also fix the current failure-path bugs during the swap:
- failed open must reset stale elapsed/total time instead of leaving the previous track's values visible
- PLAY after a failed open must not blindly resurrect playlist item 0

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

## Network-playback history

A remote playback failure was observed in v0.2.4 while remote directories still enumerated normally. Later:
- network playback worked again
- no intentional playback/network-code change explained the recovery
- VLC also buffered/chopped on the same remote path during the degraded session

Therefore the v0.2.4 incident remains historical evidence and is distinct from the newly confirmed MCI/MP3 compatibility problem above.

## Later requested functional tweaks

After the backend replacement is stable:
- Add Shuffle while preserving folder-as-playlist behavior.
- Add embedded track metadata to the Now Playing area when practical.
- Fall back cleanly to filename when metadata is missing.
- Keep the current analyzer behavior unless runtime use exposes a specific problem.

## Preserve

- working folder browser and folder-as-playlist model
- working mapped-drive / UNC behavior
- native Browse button
- manual path + GO
- async folder enumeration
- current dark stereo-component visual direction
- current spectrum analyzer concept and behavior
- VU-style app icon
