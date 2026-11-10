package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/TenacityMind/RokuVideoCaster/internal/config"
	"github.com/TenacityMind/RokuVideoCaster/internal/media"
	"github.com/TenacityMind/RokuVideoCaster/internal/playlist"
	"github.com/TenacityMind/RokuVideoCaster/internal/roku"
	"github.com/TenacityMind/RokuVideoCaster/internal/scraper"
	"github.com/TenacityMind/RokuVideoCaster/internal/subtitle"
)

type Server struct {
	cfg         *config.Config
	rokuClient  *roku.Client
	mediaServer *media.Server
	playlistMgr *playlist.Manager
	mux         *http.ServeMux
	webFS       http.FileSystem
}

func NewServer(cfg *config.Config, webFS http.FileSystem) *Server {
	_, _, port, autoAdvance := cfg.Get()
	s := &Server{
		cfg:         cfg,
		rokuClient:  roku.NewClient(),
		mediaServer: media.NewServer(port),
		playlistMgr: playlist.NewManager(autoAdvance),
		mux:         http.NewServeMux(),
		webFS:       webFS,
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	// API Endpoints
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/device/test", s.handleDeviceTest)
	s.mux.HandleFunc("/api/device/install", s.handleDeviceInstall)
	s.mux.HandleFunc("/api/probe/local", s.handleProbeLocal)
	s.mux.HandleFunc("/api/analyze/web", s.handleAnalyzeWeb)
	s.mux.HandleFunc("/api/subtitles/upload", s.handleSubtitleUpload)
	s.mux.HandleFunc("/api/cast/local", s.handleCastLocal)
	s.mux.HandleFunc("/api/cast/web", s.handleCastWeb)
	s.mux.HandleFunc("/api/cast/control", s.handleCastControl)
	s.mux.HandleFunc("/api/playlist", s.handlePlaylist)
	s.mux.HandleFunc("/api/playlist/add", s.handlePlaylistAdd)
	s.mux.HandleFunc("/api/playlist/remove", s.handlePlaylistRemove)
	s.mux.HandleFunc("/api/roku/event", s.handleRokuEvent)

	// Media and Subtitle direct streaming routes
	s.mux.HandleFunc("/media/", s.mediaServer.ServeHTTPMedia)
	s.mux.HandleFunc("/subtitles/", s.mediaServer.ServeHTTPSubtitle)

	// Embedded Web UI
	if s.webFS != nil {
		fileServer := http.FileServer(s.webFS)
		s.mux.Handle("/", fileServer)
	}
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ip, _, port, autoAdvance := s.cfg.Get()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"roku_ip":      ip,
			"server_port":  port,
			"auto_advance": autoAdvance,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			RokuIP       string `json:"roku_ip"`
			RokuPassword string `json:"roku_password"`
			ServerPort   int    `json:"server_port"`
			AutoAdvance  bool   `json:"auto_advance"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
		if err := s.cfg.Update(req.RokuIP, req.RokuPassword, req.ServerPort, req.AutoAdvance); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
			return
		}
		s.playlistMgr.SetAutoAdvance(req.AutoAdvance)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleDeviceTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	targetIP := req.IP
	if targetIP == "" {
		targetIP, _, _, _ = s.cfg.Get()
	}

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	status, err := s.rokuClient.TestConnection(ctx, targetIP)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"online":  false,
			"ip":      targetIP,
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleDeviceInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IP       string `json:"ip"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	targetIP := req.IP
	if targetIP == "" {
		targetIP, _, _, _ = s.cfg.Get()
	}
	password := req.Password
	if password == "" {
		_, password, _, _ = s.cfg.Get()
	}

	res, err := roku.SideloadChannel(r.Context(), targetIP, password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleProbeLocal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath string `json:"file_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	info, err := media.ProbeMediaFile(r.Context(), req.FilePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleAnalyzeWeb(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	result, err := scraper.AnalyzeVideoURL(r.Context(), req.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSubtitleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "parse form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("subtitle")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing subtitle file: "+err.Error())
		return
	}
	defer file.Close()

	// Normalize subtitle to UTF-8 SRT
	normalized, err := subtitle.NormalizeToSRT(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to normalize subtitle: "+err.Error())
		return
	}

	token, _, err := s.mediaServer.SaveUploadedSubtitle(header.Filename, strings.NewReader(string(normalized)))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save subtitle: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":     token,
		"filename":  header.Filename,
		"sub_url":   fmt.Sprintf("/subtitles/%s.srt", token),
		"message":   "Subtitle uploaded and normalized successfully",
	})
}

func (s *Server) handleCastLocal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FilePath         string  `json:"file_path"`
		Title            string  `json:"title"`
		AudioTrack       int     `json:"audio_track"`
		SubtitleToken    string  `json:"subtitle_token"`
		EmbeddedSubIndex int     `json:"embedded_sub_index"`
		SubOffset        float64 `json:"sub_offset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	targetIP, _, port, _ := s.cfg.Get()
	if targetIP == "" {
		writeError(w, http.StatusBadRequest, "Roku TV IP is not configured")
		return
	}

	localIP := media.GetOutboundIP(targetIP)
	fileToken := s.mediaServer.RegisterFile(req.FilePath)
	mediaURL := fmt.Sprintf("http://%s:%d/media/%s", localIP, port, fileToken)

	var subURL string
	if req.SubtitleToken != "" {
		subURL = fmt.Sprintf("http://%s:%d/subtitles/%s.srt", localIP, port, req.SubtitleToken)
	} else if req.EmbeddedSubIndex >= 0 {
		subTok, err := s.mediaServer.ExtractEmbeddedSubtitle(r.Context(), req.FilePath, req.EmbeddedSubIndex)
		if err == nil {
			subURL = fmt.Sprintf("http://%s:%d/subtitles/%s.srt", localIP, port, subTok)
		}
	}

	title := req.Title
	if title == "" {
		title = filepath.Base(req.FilePath)
	}

	callbackURL := fmt.Sprintf("http://%s:%d/api/roku/event", localIP, port)

	err := s.rokuClient.Launch(r.Context(), targetIP, roku.LaunchParams{
		MediaURL:    mediaURL,
		SubURL:      subURL,
		Title:       title,
		SubOffset:   req.SubOffset,
		AudioTrack:  req.AudioTrack,
		CallbackURL: callbackURL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "casting",
		"media_url": mediaURL,
		"sub_url":   subURL,
		"title":     title,
	})
}

func (s *Server) handleCastWeb(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		StreamURL     string  `json:"stream_url"`
		Title         string  `json:"title"`
		SubtitleURL   string  `json:"subtitle_url"`
		SubtitleToken string  `json:"subtitle_token"`
		SubOffset     float64 `json:"sub_offset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	targetIP, _, port, _ := s.cfg.Get()
	if targetIP == "" {
		writeError(w, http.StatusBadRequest, "Roku TV IP is not configured")
		return
	}

	localIP := media.GetOutboundIP(targetIP)
	subURL := req.SubtitleURL
	if req.SubtitleToken != "" {
		subURL = fmt.Sprintf("http://%s:%d/subtitles/%s.srt", localIP, port, req.SubtitleToken)
	}

	title := req.Title
	if title == "" {
		title = "Online Stream"
	}

	callbackURL := fmt.Sprintf("http://%s:%d/api/roku/event", localIP, port)

	err := s.rokuClient.Launch(r.Context(), targetIP, roku.LaunchParams{
		MediaURL:    req.StreamURL,
		SubURL:      subURL,
		Title:       title,
		SubOffset:   req.SubOffset,
		CallbackURL: callbackURL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "casting",
		"media_url": req.StreamURL,
		"sub_url":   subURL,
		"title":     title,
	})
}

func (s *Server) handleCastControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Command string  `json:"command"` // "play", "pause", "rev", "fwd", "up", "down", "left", "right", "select", "back"
		Offset  float64 `json:"offset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	targetIP, _, _, _ := s.cfg.Get()
	if targetIP == "" {
		writeError(w, http.StatusBadRequest, "Roku TV IP is not configured")
		return
	}

	keyMap := map[string]string{
		"play":   "Play",
		"pause":  "Play",
		"rev":    "Rev",
		"fwd":    "Fwd",
		"up":     "Up",
		"down":   "Down",
		"left":   "Left",
		"right":  "Right",
		"ok":     "Select",
		"select": "Select",
		"back":   "Back",
	}

	rokuKey, ok := keyMap[strings.ToLower(req.Command)]
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown command: "+req.Command)
		return
	}

	if err := s.rokuClient.SendKey(r.Context(), targetIP, rokuKey); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "key": rokuKey})
}

func (s *Server) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.playlistMgr.List())
}

func (s *Server) handlePlaylistAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Title    string  `json:"title"`
		MediaURL string  `json:"media_url"`
		SubURL   string  `json:"sub_url"`
		Format   string  `json:"format"`
		Duration float64 `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	item := s.playlistMgr.Add(req.Title, req.MediaURL, req.SubURL, req.Format, req.Duration)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handlePlaylistRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	removed := s.playlistMgr.Remove(req.ID)
	writeJSON(w, http.StatusOK, map[string]bool{"removed": removed})
}

func (s *Server) handleRokuEvent(w http.ResponseWriter, r *http.Request) {
	event := r.URL.Query().Get("event")

	if (event == "finished" && s.playlistMgr.IsAutoAdvance()) || event == "next" {
		nextItem := s.playlistMgr.Next()
		if nextItem != nil {
			targetIP, _, _, _ := s.cfg.Get()
			if targetIP != "" {
				_ = s.rokuClient.Launch(r.Context(), targetIP, roku.LaunchParams{
					MediaURL:    nextItem.MediaURL,
					SubURL:      nextItem.SubURL,
					Title:       nextItem.Title,
					CallbackURL: r.URL.String(),
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "event processed"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
