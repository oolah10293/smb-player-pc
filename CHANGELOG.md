# Changelog

## v0.3.1

- Added asynchronous embedded **Title / Artist** metadata display with filename fallback.
- Added metadata support through `github.com/dhowden/tag` for MP3 ID3, MP4/M4A, OGG, and FLAC tags.
- Added generation checking so slow network metadata reads cannot overwrite a newer track's display.
- Added ellipsis handling for long Now Playing metadata strings.
- Changed **Seek** channel clicks from the native trackbar's wide page jumps to direct click-to-position seeking.
- Intentionally left the **Volume** slider's existing click/drag behavior unchanged.
- Converted the decorative POWER lamp into a transport/network-status lamp:
  - green while actively playing
  - yellow while loading or waiting/stalled for data
  - red while stopped, ready, paused, or in error
- Added Media Foundation WAITING/STALLED buffering-state handling.
- Added an approximately 1.5-second smooth fade-in when resuming from pause.
- Added the same fade-in when playback resumes after a detected buffering interruption.
- The fade does not move the volume slider and yields immediately if the user adjusts volume.
- Preserved the Media Foundation backend, Shuffle, network filesystem behavior, and 31-band WASAPI spectrum analyzer.

## v0.3.0

- Replaced the Windows MCI playback backend with Windows Media Foundation **IMFMediaEngine** in audio-only mode.
- Kept the existing browser, mapped-drive / UNC filesystem behavior, UI, and WASAPI spectrum analyzer separate from the playback-backend change.
- Added explicit Media Foundation startup/shutdown and an IMFMediaEngineNotify callback that posts media events back to the Win32 window.
- Added file-URL conversion for local, mapped-drive, and UNC media paths.
- Added Media Engine playback, pause/resume, seek, duration/position, volume/mute, and end-of-track handling.
- Added a visible LOADING state while a Media Engine source is being prepared.
- Added **Shuffle** with an ON/OFF button.
- Shuffle Play Folder starts at a random track, visits the folder without repeats within a cycle, avoids an immediate repeat across cycle boundaries, and preserves actual history for Previous/Next navigation.
- Fixed failed-open UI state so stale elapsed/total time is cleared.
- Fixed the previous behavior where PLAY after a failed open could restart playlist item 0.
- Added guards against stale PAUSE/ENDED events while a replacement Media Engine source is still loading.
- Preserved the v0.2.7 31-band spectrum analyzer and VU-style app icon.
- GitHub Actions successfully builds the v0.3.0 64-bit Windows executable.
- Runtime validation of the new backend is still required before the migration is considered complete.

## v0.2.7

- Replaced the dual analog VU meters with one wide **31-band segmented spectrum analyzer**.
- Preserved the existing Windows MCI playback path; the analyzer is a separate visualization branch.
- Added WASAPI shared-mode endpoint loopback capture of the default Windows render device.
- Analyzer intentionally responds to all audio on that Windows output device, not only SMB Player.
- Added a 4096-point FFT and logarithmically spaced bands covering roughly 35 Hz to 16 kHz.
- Added 16 LED-style segments per band with green, amber, and red regions.
- Added fast bar response and short peak-hold markers.
- Added one common slow display AGC so the spectrum stays visually useful across quiet and loud material without altering the playback audio or independently normalizing each band.
- Runtime test confirmed the spectrum is genuinely frequency-dependent and visually tracks the music.
- Runtime test also confirmed the analyzer is effectively unaffected by the SMB Player volume control until mute, which is desirable for this display.
- Retained the existing VU-style application icon as an homage to the original meter design.
- Network/playback behavior otherwise intentionally unchanged.

## v0.2.6

- Reworked VU needle motion for **fast but continuous** analog-style travel.
- Increased meter update cadence from 25 Hz to approximately 100 Hz with 1 ms Windows timer resolution requested.
- Replaced heavy smoothing with a lightly damped spring model so the needle visibly traverses intermediate positions instead of appearing to jump.
- Expanded the middle of the display response so typical mastered music produces more visible needle travel.
- Corrected VU scale proportions and kept LEFT/RIGHT/VU markings clear of the pivot.
- Moved SEEK and VOLUME captions into a dedicated bottom control strip and centered them over their controls.
- Preserved the VU-style application icon.
- Playback/network code intentionally unchanged.

## v0.2.5

- Further meter styling and UI polish.
- Added the custom VU-style application icon.
- Confirmed network playback working on a different computer/network-drive setup.
- Identified that smooth VU movement still needed more responsive continuous travel.

## v0.2.4

- Continued vintage receiver/stereo visual redesign.
- Reworked VU meters toward more realistic proportions.
- Added warm backlit meter treatment and deeper bezels.
- Added POWER indicator lamp.
- Added visible SEEK and VOLUME labels.
- Added more faceplate/detail treatment around the controls and file-list area.
- A remote-playback failure was observed during a degraded network session; later testing did not reproduce it and VLC also buffered on the same network path.

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
