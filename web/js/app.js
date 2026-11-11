// State
let appConfig = { roku_ip: "", server_port: 8080, auto_advance: true };
let currentLocalFile = null;
let currentLocalProbe = null;
let currentLocalSubToken = "";
let currentWebAnalysis = null;
let currentWebSubToken = "";
let currentWebSubMode = "upload";
let playlist = [];

// Initialize
document.addEventListener("DOMContentLoaded", async () => {
  await loadConfig();
  await loadPlaylist();
  if (appConfig.roku_ip) {
    checkDeviceHealth(appConfig.roku_ip);
  }
});

// Config & Device
async function loadConfig() {
  try {
    const res = await fetch("/api/config");
    if (res.ok) {
      appConfig = await res.json();
      document.getElementById("modalIp").value = appConfig.roku_ip || "";
      document.getElementById("headerDeviceIp").textContent = appConfig.roku_ip || "No IP Set";
      document.getElementById("autoAdvanceCheckbox").checked = appConfig.auto_advance;
    }
  } catch (e) {
    console.error("Failed to load config", e);
  }
}

async function checkDeviceHealth(ip) {
  try {
    const res = await fetch("/api/device/test", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ip }),
    });
    const data = await res.json();
    const dot = document.getElementById("headerStatusDot");
    const name = document.getElementById("headerDeviceName");
    const ipLabel = document.getElementById("headerDeviceIp");

    if (data.online) {
      dot.className = "w-2 h-2 rounded-full bg-emerald-400 animate-pulse";
      name.textContent = data.device_name || "Roku TV";
      ipLabel.textContent = data.ip;
    } else {
      dot.className = "w-2 h-2 rounded-full bg-rose-500";
      name.textContent = "TV Offline";
      ipLabel.textContent = ip;
    }
  } catch (e) {
    console.error("Health check error", e);
  }
}

async function testDeviceConnection() {
  const ip = document.getElementById("modalIp").value.trim();
  const resBox = document.getElementById("modalPingResult");
  resBox.className = "p-3 rounded-xl text-xs flex items-center gap-2 bg-slate-800 text-slate-300";
  resBox.innerHTML = "<span>⏳</span><span>Pinging Roku TV ECP (Port 8060)...</span>";
  resBox.classList.remove("hidden");

  try {
    const res = await fetch("/api/device/test", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ip }),
    });
    const data = await res.json();
    if (data.online) {
      resBox.className = "p-3 rounded-xl text-xs flex items-center gap-2 bg-emerald-500/10 border border-emerald-500/20 text-emerald-400";
      resBox.innerHTML = `<span>✓</span><span>Online: ${data.device_name} • ${data.model}</span>`;
      checkDeviceHealth(ip);
    } else {
      resBox.className = "p-3 rounded-xl text-xs flex items-center gap-2 bg-rose-500/10 border border-rose-500/20 text-rose-400";
      resBox.innerHTML = `<span>✕</span><span>Offline: Could not connect to ${ip}:8060</span>`;
    }
  } catch (e) {
    resBox.className = "p-3 rounded-xl text-xs flex items-center gap-2 bg-rose-500/10 border border-rose-500/20 text-rose-400";
    resBox.innerHTML = `<span>✕</span><span>Network Error: ${e.message}</span>`;
  }
}

async function sideloadRokuChannel() {
  const ip = document.getElementById("modalIp").value.trim();
  const password = document.getElementById("modalPassword").value.trim();

  if (!ip) return alert("Please enter Roku TV IP address");
  if (!password) return alert("Please enter Roku Developer Password");

  const resBox = document.getElementById("modalPingResult");
  resBox.className = "p-3 rounded-xl text-xs flex items-center gap-2 bg-slate-800 text-sky-400";
  resBox.innerHTML = "<span>⏳</span><span>Uploading & installing channel via Digest Auth...</span>";
  resBox.classList.remove("hidden");

  try {
    const res = await fetch("/api/device/install", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ip, password }),
    });
    const data = await res.json();
    if (res.ok) {
      resBox.className = "p-3 rounded-xl text-xs flex items-center gap-2 bg-emerald-500/10 border border-emerald-500/20 text-emerald-400";
      resBox.innerHTML = `<span>✓</span><span>${data.message || "Channel installed successfully on Roku TV!"}</span>`;
    } else {
      resBox.className = "p-3 rounded-xl text-xs flex items-center gap-2 bg-rose-500/10 border border-rose-500/20 text-rose-400";
      resBox.innerHTML = `<span>✕</span><span>${data.error || "Installation failed"}</span>`;
    }
  } catch (e) {
    resBox.className = "p-3 rounded-xl text-xs flex items-center gap-2 bg-rose-500/10 border border-rose-500/20 text-rose-400";
    resBox.innerHTML = `<span>✕</span><span>Upload error: ${e.message}</span>`;
  }
}

async function saveDeviceConfig() {
  const ip = document.getElementById("modalIp").value.trim();
  const password = document.getElementById("modalPassword").value.trim();
  const autoAdvance = document.getElementById("autoAdvanceCheckbox").checked;

  try {
    await fetch("/api/config", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        roku_ip: ip,
        roku_password: password,
        auto_advance: autoAdvance,
      }),
    });
    appConfig.roku_ip = ip;
    checkDeviceHealth(ip);
    toggleDeviceModal();
  } catch (e) {
    alert("Failed to save config: " + e.message);
  }
}

function toggleDeviceModal() {
  document.getElementById("deviceModal").classList.toggle("hidden");
}

// Navigation Tabs
function selectTab(tabId) {
  const tabs = ["local", "web", "queue"];
  tabs.forEach(t => {
    const btn = document.getElementById("tab" + t.charAt(0).toUpperCase() + t.slice(1));
    const panel = document.getElementById("panel" + t.charAt(0).toUpperCase() + t.slice(1));
    if (t === tabId) {
      btn.classList.add("active-nav-tab", "border-sky-400");
      btn.classList.remove("border-transparent", "text-slate-400");
      panel.classList.remove("hidden");
    } else {
      btn.classList.remove("active-nav-tab", "border-sky-400");
      btn.classList.add("border-transparent", "text-slate-400");
      panel.classList.add("hidden");
    }
  });
}

// Local File Handling
function triggerFilePick() {
  document.getElementById("localFileInput").click();
}

async function onFileSelected(event) {
  const file = event.target.files[0];
  if (!file) return;

  currentLocalFile = file;
  document.getElementById("localFileName").textContent = file.name;
  document.getElementById("localFileSize").textContent = (file.size / (1024 * 1024)).toFixed(1) + " MB";
  document.getElementById("localMediaCard").classList.remove("hidden");

  // If local file object has webkitRelativePath or path (in desktop wrapper)
  const filePath = file.path || file.name;

  try {
    const res = await fetch("/api/probe/local", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ file_path: filePath }),
    });
    if (res.ok) {
      currentLocalProbe = await res.json();
      renderProbeResults(currentLocalProbe);
    }
  } catch (e) {
    console.warn("Probe local file:", e);
  }
}

function renderProbeResults(info) {
  if (info.resolution) {
    document.getElementById("localResolution").textContent = info.resolution;
  }

  // Audio Tracks
  const audioSel = document.getElementById("audioTrackSelect");
  audioSel.innerHTML = "";
  if (info.audio_tracks && info.audio_tracks.length > 0) {
    document.getElementById("audioTrackCount").textContent = `${info.audio_tracks.length} Tracks Found`;
    info.audio_tracks.forEach(t => {
      const opt = document.createElement("option");
      opt.value = t.index;
      opt.textContent = t.title;
      audioSel.appendChild(opt);
    });
  } else {
    audioSel.innerHTML = `<option value="0">Track 1: Primary Audio (Direct Passthrough)</option>`;
  }

  // Subtitles
  const subSel = document.getElementById("subTrackSelect");
  subSel.innerHTML = `
    <option value="none">None (Turn Off Subtitles)</option>
    <option value="upload">+ Upload Local .SRT / .VTT File...</option>
  `;
  if (info.sub_tracks && info.sub_tracks.length > 0) {
    info.sub_tracks.forEach((s, idx) => {
      const opt = document.createElement("option");
      opt.value = `embedded_${s.index}`;
      opt.textContent = `Embedded: ${s.title}`;
      subSel.appendChild(opt);
    });
  }
}

function onSubTrackChange() {
  const val = document.getElementById("subTrackSelect").value;
  const box = document.getElementById("subUploadBox");
  if (val === "upload") {
    box.classList.remove("hidden");
    pickLocalSub();
  } else {
    box.classList.add("hidden");
  }
}

function pickLocalSub() {
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".srt,.vtt";
  input.onchange = async (e) => {
    const file = e.target.files[0];
    if (file) {
      document.getElementById("subFileName").textContent = file.name;
      document.getElementById("subUploadBox").classList.remove("hidden");
      // Upload to local server
      const formData = new FormData();
      formData.append("subtitle", file);
      try {
        const res = await fetch("/api/subtitles/upload", { method: "POST", body: formData });
        const data = await res.json();
        if (data.token) {
          currentLocalSubToken = data.token;
        }
      } catch (err) {
        alert("Subtitle upload failed: " + err.message);
      }
    }
  };
  input.click();
}

async function castCurrentLocal() {
  if (!currentLocalFile) return alert("Please select a video file first");

  const filePath = currentLocalFile.path || currentLocalFile.name;
  const subVal = document.getElementById("subTrackSelect").value;
  let embeddedIdx = -1;
  let subToken = "";

  if (subVal.startsWith("embedded_")) {
    embeddedIdx = parseInt(subVal.replace("embedded_", ""));
  } else if (subVal === "upload") {
    subToken = currentLocalSubToken;
  }

  const audioTrack = parseInt(document.getElementById("audioTrackSelect").value) || 0;

  try {
    const res = await fetch("/api/cast/local", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        file_path: filePath,
        title: currentLocalFile.name,
        audio_track: audioTrack,
        subtitle_token: subToken,
        embedded_sub_index: embeddedIdx,
      }),
    });
    const data = await res.json();
    if (res.ok) {
      document.getElementById("dockTitle").textContent = currentLocalFile.name;
      document.getElementById("dockMeta").innerHTML = `<span class="px-1.5 py-0.2 rounded bg-slate-800 text-sky-400 font-mono">Playing on TV</span>`;
    } else {
      alert("Cast error: " + (data.error || "Unknown error"));
    }
  } catch (e) {
    alert("Network error: " + e.message);
  }
}

async function addCurrentLocalToQueue() {
  if (!currentLocalFile) return alert("Please select a video file first");
  const filePath = currentLocalFile.path || currentLocalFile.name;

  await fetch("/api/playlist/add", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      title: currentLocalFile.name,
      media_url: filePath,
      format: "local",
      duration: currentLocalProbe ? currentLocalProbe.duration_sec : 0,
    }),
  });
  loadPlaylist();
  alert("Added to playlist queue!");
}

// Online Video Link Handling
async function analyzeWebLink() {
  const url = document.getElementById("webStreamUrl").value.trim();
  if (!url) return alert("Please enter a video URL");

  const badge = document.getElementById("webSubStatusBadge");
  badge.innerHTML = "<span>⏳</span><span>Analyzing stream...</span>";

  try {
    const res = await fetch("/api/analyze/web", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url }),
    });
    const data = await res.json();
    if (res.ok) {
      currentWebAnalysis = data;
      renderWebAnalysis(data);
    } else {
      alert("Analysis failed: " + (data.error || "Unknown"));
    }
  } catch (e) {
    alert("Error analyzing: " + e.message);
  }
}

function renderWebAnalysis(data) {
  const grid = document.getElementById("webCandidatesGrid");
  const section = document.getElementById("webCandidatesSection");
  grid.innerHTML = "";

  if (data.candidates && data.candidates.length > 0) {
    section.classList.remove("hidden");
    data.candidates.forEach((c, idx) => {
      const card = document.createElement("label");
      card.className = `p-3 rounded-xl border flex items-center justify-between cursor-pointer ${
        idx === 0 ? "border-sky-500/40 bg-sky-500/10" : "border-slate-800 bg-slate-950/60 hover:border-slate-700"
      }`;
      card.innerHTML = `
        <div>
          <div class="text-xs font-bold text-white">${c.quality}</div>
          <div class="text-[10px] text-slate-400 font-mono">${c.format} • ${c.bitrate}</div>
        </div>
        <input type="radio" name="webCandidateRadio" value="${c.url || c.URL || ''}" ${idx === 0 ? "checked" : ""} class="text-sky-500 focus:ring-0">
      `;
      grid.appendChild(card);
    });
  }

  // Subtitles
  const badge = document.getElementById("webSubStatusBadge");
  const autoSelect = document.getElementById("webScrapedSubSelect");
  autoSelect.innerHTML = "";

  if (data.subtitles && data.subtitles.length > 0) {
    badge.className = "flex items-center gap-1.5 text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20";
    badge.innerHTML = `<span>✓</span> ${data.subtitles.length} Subtitles Discovered`;
    data.subtitles.forEach(s => {
      const opt = document.createElement("option");
      opt.value = s.url || s.URL || "";
      opt.textContent = `${s.language || s.Language || "Sub"} (${(s.format || s.Format || "SRT").toUpperCase()})`;
      autoSelect.appendChild(opt);
    });
    setWebSubMode("auto");
  } else {
    badge.className = "flex items-center gap-1.5 text-[10px] font-mono px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-400 border border-amber-500/20";
    badge.innerHTML = "<span>⚠️</span> No subtitles in stream";
    setWebSubMode("upload");
  }
}

function setWebSubMode(mode) {
  currentWebSubMode = mode;
  const modes = ["upload", "url", "auto", "off"];
  modes.forEach(m => {
    const btn = document.getElementById("subModeBtn" + m.charAt(0).toUpperCase() + m.slice(1));
    const box = document.getElementById("webSubBox" + m.charAt(0).toUpperCase() + m.slice(1));
    if (m === mode) {
      btn.className = "p-2.5 rounded-xl border border-sky-500/50 bg-sky-500/10 text-sky-300 font-medium text-left transition-all";
      box.classList.remove("hidden");
    } else {
      btn.className = "p-2.5 rounded-xl border border-slate-800 bg-slate-950/60 text-slate-400 font-medium text-left hover:border-slate-700 transition-all";
      box.classList.add("hidden");
    }
  });
}

function pickWebLocalSub() {
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".srt,.vtt";
  input.onchange = async (e) => {
    const file = e.target.files[0];
    if (file) {
      document.getElementById("webUploadedSubName").textContent = file.name;
      const formData = new FormData();
      formData.append("subtitle", file);
      try {
        const res = await fetch("/api/subtitles/upload", { method: "POST", body: formData });
        const data = await res.json();
        if (data.token) {
          currentWebSubToken = data.token;
          const badge = document.getElementById("webSubStatusBadge");
          badge.className = "flex items-center gap-1.5 text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20";
          badge.innerHTML = "<span>✓</span> Local Sub Ready";
        }
      } catch (err) {
        alert("Upload error: " + err.message);
      }
    }
  };
  input.click();
}

async function castCurrentWeb() {
  let streamURL = document.getElementById("webStreamUrl").value.trim();
  const selectedRadio = document.querySelector('input[name="webCandidateRadio"]:checked');
  if (selectedRadio && selectedRadio.value && selectedRadio.value !== "undefined") {
    streamURL = selectedRadio.value;
  }
  if (!streamURL || streamURL === "undefined") return alert("Please enter or select a valid video stream link");

  let subURL = "";
  let subToken = "";
  if (currentWebSubMode === "upload") {
    subToken = currentWebSubToken;
  } else if (currentWebSubMode === "url") {
    subURL = document.getElementById("webSubCustomUrl").value.trim();
  } else if (currentWebSubMode === "auto") {
    subURL = document.getElementById("webScrapedSubSelect").value;
  }

  const title = currentWebAnalysis ? (currentWebAnalysis.title || currentWebAnalysis.Title || "Online Stream") : "Online Stream";

  try {
    const res = await fetch("/api/cast/web", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        stream_url: streamURL,
        title: title,
        subtitle_url: subURL,
        subtitle_token: subToken,
      }),
    });
    const data = await res.json();
    if (res.ok) {
      document.getElementById("dockTitle").textContent = title;
      document.getElementById("dockMeta").innerHTML = `<span class="px-1.5 py-0.2 rounded bg-slate-800 text-sky-400 font-mono">Streaming to TV</span>`;
    } else {
      alert("Cast error: " + (data.error || "Unknown error"));
    }
  } catch (e) {
    alert("Network error: " + e.message);
  }
}

async function addCurrentWebToQueue() {
  let streamURL = document.getElementById("webStreamUrl").value.trim();
  const selectedRadio = document.querySelector('input[name="webCandidateRadio"]:checked');
  if (selectedRadio) streamURL = selectedRadio.value;
  if (!streamURL) return alert("Please enter a stream link first");

  const title = currentWebAnalysis ? currentWebAnalysis.Title : "Online Stream";
  await fetch("/api/playlist/add", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      title: title,
      media_url: streamURL,
      format: "web",
    }),
  });
  loadPlaylist();
  alert("Added to playlist queue!");
}

// Playlist Queue
async function loadPlaylist() {
  try {
    const res = await fetch("/api/playlist");
    if (res.ok) {
      playlist = await res.json();
      renderPlaylist(playlist);
    }
  } catch (e) {
    console.warn("Load playlist error", e);
  }
}

function renderPlaylist(items) {
  const container = document.getElementById("queueList");
  document.getElementById("queueBadge").textContent = items.length;

  if (!items || items.length === 0) {
    container.innerHTML = `
      <div class="p-6 text-center text-xs text-slate-500">
        Playlist queue is empty. Add videos from Local Files or Online Video Link.
      </div>
    `;
    return;
  }

  container.innerHTML = "";
  items.forEach((item, idx) => {
    const div = document.createElement("div");
    div.className = `p-3 rounded-xl border flex items-center justify-between ${
      item.is_current ? "bg-sky-500/10 border-sky-500/30" : "bg-slate-950 border-slate-800"
    }`;
    div.innerHTML = `
      <div class="flex items-center gap-3">
        <span class="w-6 h-6 rounded-lg ${
          item.is_current ? "bg-sky-500/20 text-sky-400" : "bg-slate-800 text-slate-400"
        } font-bold text-xs flex items-center justify-center">${idx + 1}</span>
        <div>
          <div class="text-xs font-bold text-white">${item.title}</div>
          <div class="text-[10px] text-slate-400">${item.format}</div>
        </div>
      </div>
      <div class="flex items-center gap-3">
        ${item.is_current ? `<span class="text-xs font-mono text-emerald-400 font-semibold">Now Playing</span>` : ""}
        <button onclick="removePlaylistItem('${item.id}')" class="text-xs text-rose-400 hover:text-rose-300 font-bold px-2 py-1">×</button>
      </div>
    `;
    container.appendChild(div);
  });
}

async function removePlaylistItem(id) {
  await fetch("/api/playlist/remove", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });
  loadPlaylist();
}

async function toggleAutoAdvance() {
  const enabled = document.getElementById("autoAdvanceCheckbox").checked;
  await fetch("/api/config", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ auto_advance: enabled }),
  });
}

// Remote Transport Control
async function sendRemoteKey(command) {
  try {
    await fetch("/api/cast/control", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ command }),
    });
  } catch (e) {
    console.error("Remote key error", e);
  }
}
