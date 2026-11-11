# 📺 Roku Video Caster (V2.0 PRO)

A dedicated, lightweight video casting system built **exclusively for Roku TV**. 

Cast personal local movies and online videos directly to your Roku TV with zero buffering, multi-track audio switching (Dolby Atmos / 5.1 passthrough), continuous playlists, and an interactive on-screen media player with real-time subtitle font sizing and sync adjustment.

---

## ✨ Features

- **📺 Roku TV Exclusive:** Zero DLNA or Chromecast overhead. Built specifically for Roku OS using the native SceneGraph framework.
- **⚡ Hardware Direct Passthrough:** Direct hardware streaming for 4K UHD, HDR, and Dolby Atmos/5.1 surround sound without local transcoding.
- **🎮 Redesigned Cinematic TV Media Player:**
  - **Discrete Seek vs. Menu Modes:** While playing, `Left`/`Right` directly skips backward/forward by 15s without menus getting in the way.
  - **Main Player Deck (Remote `OK` Key):** Pauses video and shows progress bar, elapsed/total time, and 3 focused action cards: `[ > Resume ]`, `[ [CC] Subtitles & Audio ]`, and `[ >> Next Video ]`.
  - **Dedicated Subtitle & Audio Drawer:** Sleek side panel with 6 easy vertical rows:
    - **Subtitle Track:** Toggle tracks (`English [CC]` / `OFF`).
    - **Subtitle Size:** Switch between `Normal (30pt)`, `Large (42pt - default)`, and `Huge (54pt)`.
    - **Font Color:** Choose between `White`, `Yellow`, or `Sky Blue`.
    - **Sync Timing:** Nudge timing by `±0.5s` with real-time zero-buffering response (`OK` resets to `0.0s`).
    - **Audio Track:** Toggle between available embedded audio streams.
    - **Live On-Screen Preview:** See your chosen font size and color live on screen before resuming playback!
- **📁 Personal Local File Streaming:** Drop any `.mkv`, `.mp4`, `.mov`, or `.webm` file. Automatically inspects audio tracks and embedded subtitles using `ffprobe`.
- **🌐 Headless Chrome Stream Scraper:** Paste streaming links or video pages (e.g. `primeshows.org`, `flixer.su`). The built-in browser engine automatically navigates iframes, bypasses Cloudflare Turnstile, and extracts direct `.m3u8` streams and subtitles.
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
