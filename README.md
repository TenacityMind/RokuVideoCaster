# 📺 Roku Video Caster (V2.0 PRO)

A dedicated, lightweight video casting system built **exclusively for Roku TV**. 

Cast personal local movies and online videos directly to your Roku TV with zero buffering, multi-track audio switching (Dolby Atmos / 5.1 passthrough), continuous playlists, and an interactive on-screen media player with real-time subtitle sync adjustment.

---

## ✨ Features

- **📺 Roku TV Exclusive:** Zero DLNA or Chromecast overhead. Built specifically for Roku OS using the native SceneGraph framework.
- **⚡ Hardware Direct Passthrough:** Direct hardware streaming for 4K UHD, HDR, and Dolby Atmos/5.1 surround sound without local transcoding.
- **🎮 Interactive TV Player (Remote `OK` Key):**
  - **Remaining Time Countdown:** Always see exact time left (`-XX:XX`).
  - **Subtitle Controls:** Nudge sync timing `[-0.5s]` / `[+0.5s]` with instant zero-buffering response, toggle subtitle styles, or turn captions off.
  - **Audio Track Switcher:** Easily toggle between embedded audio streams on your TV.
- **📁 Personal Local File Streaming:** Drop any `.mkv`, `.mp4`, `.mov`, or `.webm` file. Automatically inspects audio tracks and embedded subtitles using `ffprobe`.
- **🌐 Online Video Stream Extractor:** Paste streaming links or video pages. Offers 1080p, 4K, and 720p stream quality selectors.
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

## 🎮 Remote Control Shortcuts

| Remote Button | Action When Menu Closed | Action When Menu Open (`OK`) |
|:---:|:---:|:---:|
| **`OK`** | Opens the Interactive OSD Menu | Toggles active setting / closes menu |
| **`▲ / ▼` (Up / Down)** | Quick Subtitle Sync Nudge (`±0.5s`) | Navigate control pills |
| **`◀ / ▶` (Left / Right)** | 15s Skip Backward / Forward | Adjust sync offset / cycle text color |
| **`Play / Pause`** | Play / Pause video | Play / Pause video |
| **`Back`** | Stop playback & return to dashboard | Close menu |

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

## 📄 License
MIT License. Built for seamless, ad-free streaming on Roku TV.
