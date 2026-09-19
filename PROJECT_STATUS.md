# Project Status — 2026-09-19

## What is confirmed working

- Application launches and remains responsive.
- Native folder browsing works.
- Local folders enumerate correctly.
- Remote/network folders enumerate correctly.
- Local MP3 playback works.
- Folder-as-playlist behavior works.
- Transport controls, seek, volume, and auto-advance are implemented.
- VU meters move with real Windows output audio.
- VU flicker issue from v0.2 was fixed in v0.2.1.
- Vintage stereo/receiver visual direction is established.

## Current blocker

Remote audio playback is broken in v0.2.4 even though remote folder enumeration works.

### Reproduction

1. Browse to the remote/network music location.
2. Confirm files are listed normally.
3. Try **Play Folder**.
   - Result: `CAN'T OPEN` essentially immediately.
4. Alternatively double-click a single remote MP3 and wait.
   - Result: `CAN'T OPEN` after approximately 3–5 seconds.
5. Copy an MP3 to the local Desktop and play it.
   - Result: local playback succeeds.

### Regression boundary

- **v0.1.2:** confirmed remote playback works.
- **v0.2.4:** remote browsing works, remote playback fails.

The next technical step is to compare the v0.1.2 playback/open path against the current v0.2.4 path and isolate the change that caused remote MCI file opens to fail.

## UI direction

Keep the VU meters prominent; do not simply shrink them. Make them look more like real physical illuminated meters:

- realistic aspect ratio and scale geometry
- convincing warm backlight
- glass/bezel depth
- correct needle/pivot proportions
- stereo-style controls, labels, indicator lamps, and panel personality

The large dark center area is intentionally the file list, not unused space.
