# 📺 Roku Video Caster (V2.0 PRO)

[![Release](https://img.shields.io/badge/Release-v2.0.1-brightgreen.svg)](https://github.com/TenacityMind/RokuVideoCaster/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Windows%20%7C%20Android-lightgrey.svg)]()
[![Roku OS](https://img.shields.io/badge/Roku%20OS-10.0+-662D91.svg?logo=roku)](https://developer.roku.com)

A dedicated, high-performance streaming & video casting system built **exclusively for Roku TV**. 

Cast personal local movies and online videos directly to your Roku TV with zero buffering, multi-track audio switching (Dolby Atmos / 5.1 passthrough), continuous playlist queues, and an interactive on-screen media player with real-time subtitle font sizing and sync adjustment.

---

## ✨ Features

- **📺 Roku TV Exclusive:** Zero DLNA or Chromecast overhead. Built specifically for Roku OS using the native SceneGraph framework.
- **⚡ Hardware Direct Passthrough:** Direct hardware streaming for 4K UHD, HDR, and Dolby Atmos/5.1 surround sound without local transcoding.
- **🛡️ Built-in Transparent HLS Proxy:**
  - Circumvents CDN anti-hotlink protection (`403 Forbidden` / Referer checking) on online streaming hosts (e.g. `flixer.gd`, `flixer.su`).
  - Seamlessly tunnels browser-captured `Referer` and `User-Agent` headers upstream for every segment.
  - Rewrites both master and sub-playlist manifests in real time, converting relative URLs and CMAF fMP4 (`.m4s` + `#EXT-X-MAP`) chunks into fully playable local streams for Roku TV.
- **📁 Intelligent Local File Streaming:**
  - Drop or pick any video file (`.mp4`, `.mkv`, `.m4v`, `.mov`) in the Web UI.
  - **Smart Disk Discovery:** Automatically locates files in your user folders (`~/Downloads`, `~/Movies`, `~/Desktop`, `~/Videos`) even when standard browser sandboxes omit absolute file paths.
  - **Full Byte-Range Seeking:** HTTP `206 Partial Content` streaming enables immediate scrubbing and instant seek on Roku's native hardware decoder.
  - Inspects resolution, duration, audio tracks, and embedded subtitles via `ffprobe`.
- **🌐 Headless Chrome Stream Scraper:**
  - Paste streaming links or video pages (e.g. `primeshows.org`, `flixer.su`, `flixer.gd`).
  - Automated headless Chrome engine navigates iframes, bypasses Cloudflare Turnstile, and intercepts direct `.m3u8` master and variant playlists.
  - Robust false-positive filtering prevents web manifest stalls and suppresses raw CMAF segment clutter.
- **🎮 Netflix-Style Cinematic TV Player UI & Remote Controls:**
  - **🎞️ Scene Slider Preview (Trick Play Thumbnails):** Pressing or holding `Left` or `Right` glides a floating 16:9 thumbnail preview card along the red timeline showing the scene and timestamp. Playback is not interrupted until you press `OK` to commit the seek!
  - **Instant 140ms Hardware Extraction & In-Memory Caching:** Dynamic thumbnail generation powered by `ffmpeg` on your host machine, serving cached frames in `10ms`.
  - **Uniform Netflix Action Deck:** Clean, equal-height action buttons: `[|| Pause / > Play]`, `[<< 10s]`, `[10s >>]`, `[[CC] Subtitles & Audio]`, and `[Next Video >>|]`.
  - **Netflix Remote Control Key Behavior:**
    - `OK` while playing: Pauses video and displays options deck.
    - `Down` while playing: Shows controls deck without pausing video (plays in background).
    - `Up`: Direct one-click jump into Subtitles & Audio modal.
    - `Back`: Stepwise dismiss (Slider Preview -> Options Deck -> TV Home Screen).
    - `5-Second Auto-Hide`: OSD automatically fades out after 5s of inactivity when playing.
  - **Netflix Centered Dual-Column Subtitles & Audio Modal:**
    - Left Column: Audio stream switcher (`English 5.1`, `Stereo`, etc.) with active checkmark.
    - Right Column: Subtitle tracks, sizing (`Normal / Large / Huge`), colors (`White / Yellow / Sky Blue`), and `±0.5s` sync nudging.
  - **Pure ASCII UI:** 100% ASCII indicators eliminate missing rectangular glyph boxes across all Roku OS system fonts.
- **🏠 Cinematic TV Home Screen / Dashboard:**
  - Displays a clean emerald green `* READY TO CAST` status on startup.
  - Features a high-contrast glowing pill displaying your computer's actual local network IP (e.g. `http://10.0.0.70:8080`) so you can open the dashboard instantly on your phone, tablet, or laptop.
- **💬 Flexible Subtitle Fallback:** If an online video lacks subtitles, pair it instantly with your own uploaded `.srt`/`.vtt` file or direct subtitle link.
- **📑 Continuous Playlist Queue:** Queue multiple episodes or movies with auto-advance to the next video when finished.
- **🚀 Standalone Single Binary:** macOS and Windows single-file executables with embedded web dashboard. No Node.js, Go, or Python needed to run!
- **📱 Android Companion App:** System share target ("Share to Roku TV") from Chrome, YouTube, Brave, or Firefox.

---

## 🛠️ Roku TV Quick Setup (One-Time)

To allow sideloading the native channel onto your Roku TV:

1. **Enable Developer Mode on Roku TV:**
   - On your physical Roku remote, press in sequence:
     ```
     Home 3x  ➜  Up 2x  ➜  Right  ➜  Left  ➜  Right  ➜  Left  ➜  Right
     ```
   - Select **Enable installer and restart**.
   - Note down the **Roku TV IP Address** displayed on screen (e.g., `192.168.1.50`).
   - Set a developer password (e.g., `your-password`).
2. **Done!** You only ever need to do this once.

---

## 🚀 Running the App

### macOS
Download `roku-caster` and double-click, or run in Terminal:
```bash
./bin/roku-caster
```
*The web dashboard opens automatically at `http://localhost:8080`.*

### Windows
Download `roku-caster-win.exe` and double-click to launch.

### Sideloading the Channel to TV (One-Click)
1. Open the dashboard at `http://localhost:8080`.
2. Click the TV connection badge in the top-right corner.
3. Enter your **Roku TV IP** and **Developer Password**.
4. Click **"Sideload / Update Channel on TV"**. The channel will install and appear on your Roku TV home screen!

### CLI Sideload / Update Option
You can also sideload or update the channel directly from the command line:
```bash
./bin/roku-caster -install -ip 192.168.1.50 -password your-password
```

---

## 🎮 Remote Navigation Shortcuts

| Mode | Remote Key | Action |
| :--- | :--- | :--- |
| **Video Playing** | **`Left` / `Right`** | **Skip -15s / +15s** (No menus) |
| | **`OK` / `Pause` / `Down`** | **Pause video** & open Main Deck |
| | **`Back`** | Stop playback & exit to Dashboard |
| **Main Deck (Paused)** | **`Left` / `Right`** | Select `[Resume]`, `[Subtitles & Audio]`, or `[Next Video]` |
| | **`OK`** on Subtitles | Opens Subtitle & Audio Side Drawer |
| | **`Play` / `Back`** | Resume playback |
| **Subtitle Drawer** | **`Up` / `Down`** | Move between setting rows (Track, Size, Color, Sync, Audio, Done) |
| | **`Left` / `Right`** | Change or cycle value for selected row |
| | **`OK` on Sync** | Reset sync timing to `0.0s (In Sync)` |
| | **`OK` on Done** / **`Back`** | Return to Main Deck / resume video |

---

## 🏗️ Building from Source

Requirements: Go 1.22+ and `zip`.

```bash
# Clone the repository
git clone https://github.com/TenacityMind/RokuVideoCaster.git
cd RokuVideoCaster

# Build macOS binary
make build-mac

# Build Windows .exe
make build-win

# Build all binaries
make build-all
```

---

## 📱 Android Companion App

The `android/` directory contains an Android Studio project with an `ACTION_SEND` intent filter.
When you tap **Share** in mobile Chrome or Brave on any video link, choose **"Roku Video Caster"** to instantly send the video to your Roku TV!

---

## 🙏 Acknowledgements & Credits

Special thanks to the [Castor](https://github.com/stupside/castor) project by [stupside](https://github.com/stupside) for:
- The automated headless Chrome scraping architecture and stealth browser techniques.
- Turnstile challenge evasion and automated video player evaluation methods.
- Foundational direct media streaming and subtitle synchronization concepts.

---

## 📄 License
MIT License. Built for seamless, ad-free streaming on Roku TV.
