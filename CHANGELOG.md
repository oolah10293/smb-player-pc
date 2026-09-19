# Changelog

## v0.2.6

- Reworked VU needle motion for **fast but continuous** analog-style travel.
- Increased meter update cadence from 25 Hz to approximately 100 Hz with 1 ms Windows timer resolution requested.
- Replaced heavy smoothing with a lightly damped spring model so the needle visibly traverses intermediate positions instead of appearing to jump.
- Expanded the middle of the display response so typical mastered music produces more visible needle travel.
- Corrected VU scale proportions and kept LEFT/RIGHT/VU markings clear of the pivot.
- Moved SEEK and VOLUME captions into a dedicated bottom control strip and centered them over their controls.
- Preserved the VU-style application icon.
- Playback/network code intentionally unchanged.


## v0.2.4 — current

- Continued vintage receiver/stereo visual redesign.
- Reworked VU meters toward more realistic proportions.
- Added warm backlit meter treatment and deeper bezels.
- Added POWER indicator lamp.
- Added visible SEEK and VOLUME labels.
- Added more faceplate/detail treatment around the controls and file-list area.
- **Known regression discovered:** remote/network files enumerate but fail to open for playback.

## v0.2.3

- Meter scale/label cleanup.
- Proper major/minor graduations and red-zone markings.
- Improved needle/pivot treatment.
- Recessed meter bezels.
- Fluorescent-style Now Playing display.
- More deliberate button depth and typography.

## v0.2.2

- Restored the native Windows **BROWSE** folder picker.
- Manual path entry retained as an additional option.

## v0.2.1

- Fixed VU-area flashing/flicker.
- Double-buffered the VU panel.
- Limited repaint invalidation to the meter/status regions instead of repainting the full upper window on every meter tick.

## v0.2

- First vintage visual identity pass.
- Dual analog LEFT/RIGHT VU meters.
- VU needles driven from actual Windows output-channel peak data rather than fake animation.
- Dark receiver-style faceplate.
- Amber/green display treatment.

## v0.1.2

- Fixed immediate application freeze by pinning the Win32 GUI/message-loop goroutine to one OS thread with `runtime.LockOSThread()`.
- **Confirmed known-good network playback:** successfully played a song from a remote hard drive over the network from another PC several hours away.

## v0.1.1

- Moved directory/network enumeration off the GUI thread.
- Stopped automatically scanning the remembered folder during startup.
- Remembered path is shown at startup but waits for explicit navigation.

## v0.1

- Initial Windows proof-of-concept.
- Folder-first browsing and playback.
- Play Folder.
- Previous / Play-Pause / Next.
- Seek and volume.
- Sorting.
- Last-folder memory.
- Initial build froze shortly after startup.
