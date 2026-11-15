package media

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TenacityMind/RokuVideoCaster/internal/proxy"
)

type WebStreamTarget struct {
	StreamURL        string
	Headers          map[string]string
	ResolvedVideoURL string
}

type Server struct {
	mu            sync.RWMutex
	fileRegistry  map[string]string           // token -> absolute file path
	webRegistry   map[string]*WebStreamTarget // token -> web stream info
	subRegistry   map[string]string           // token -> absolute subtitle path
	extractedSubs map[string]string           // cache key -> extracted srt path
	thumbCache    map[string][]byte           // cache key -> JPEG bytes
	thumbSem      chan struct{}               // concurrency limiter for thumbnail extraction
	port          int
}

func NewServer(port int) *Server {
	return &Server{
		fileRegistry:  make(map[string]string),
		webRegistry:   make(map[string]*WebStreamTarget),
		subRegistry:   make(map[string]string),
		extractedSubs: make(map[string]string),
		thumbCache:    make(map[string][]byte),
		thumbSem:      make(chan struct{}, 4),
		port:          port,
	}
}

func (s *Server) RegisterFile(absPath string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already registered
	for token, p := range s.fileRegistry {
		if p == absPath {
			return token
		}
	}

	token := randomToken(16)
	s.fileRegistry[token] = absPath
	return token
}

func (s *Server) RegisterWebStream(streamURL string, headers map[string]string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, ws := range s.webRegistry {
		if ws.StreamURL == streamURL {
			return token
		}
	}

	token := "web-" + randomToken(12)
	s.webRegistry[token] = &WebStreamTarget{
		StreamURL: streamURL,
		Headers:   headers,
	}
	return token
}

func (s *Server) RegisterSubtitle(absPath string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := randomToken(16)
	s.subRegistry[token] = absPath
	return token
}

func (s *Server) SaveUploadedSubtitle(filename string, r io.Reader) (string, string, error) {
	tempDir := filepath.Join(os.TempDir(), "roku_caster_subs")
	_ = os.MkdirAll(tempDir, 0755)

	token := randomToken(12)
	destPath := filepath.Join(tempDir, token+"_"+filename)

	f, err := os.Create(destPath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return "", "", err
	}

	s.mu.Lock()
	s.subRegistry[token] = destPath
	s.mu.Unlock()

	return token, destPath, nil
}

func (s *Server) ExtractEmbeddedSubtitle(ctx context.Context, filePath string, streamIndex int) (string, error) {
	cacheKey := fmt.Sprintf("%s:%d", filePath, streamIndex)

	s.mu.RLock()
	if cached, ok := s.extractedSubs[cacheKey]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg required to extract embedded subtitles: %w", err)
	}

	tempDir := filepath.Join(os.TempDir(), "roku_caster_subs")
	_ = os.MkdirAll(tempDir, 0755)
	token := randomToken(12)
	outPath := filepath.Join(tempDir, fmt.Sprintf("%s_stream_%d.srt", token, streamIndex))

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-y",
		"-i", filePath,
		"-map", fmt.Sprintf("0:%d", streamIndex),
		"-f", "srt",
		outPath,
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg extract subtitle: %s (%w)", string(out), err)
	}

	s.mu.Lock()
	s.extractedSubs[cacheKey] = outPath
	s.subRegistry[token] = outPath
	s.mu.Unlock()

	return token, nil
}

func (s *Server) ExtractThumbnail(ctx context.Context, filePath string, seconds int) ([]byte, error) {
	cacheKey := fmt.Sprintf("%s:%d", filePath, seconds)

	s.mu.RLock()
	if cached, ok := s.thumbCache[cacheKey]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-ss", strconv.Itoa(seconds),
		"-i", filePath,
		"-vframes", "1",
		"-q:v", "3",
		"-vf", "scale=280:158",
		"-f", "image2",
		"-",
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg thumbnail extract error: %w", err)
	}

	data := buf.Bytes()
	if len(data) == 0 {
		return nil, fmt.Errorf("no thumbnail data produced")
	}

	s.mu.Lock()
	if len(s.thumbCache) > 500 {
		// Evict map if it exceeds 500 thumbnails to bound memory usage
		s.thumbCache = make(map[string][]byte)
	}
	s.thumbCache[cacheKey] = data
	s.mu.Unlock()

	return data, nil
}

func (s *Server) ExtractWebThumbnail(ctx context.Context, token string, ws *WebStreamTarget, seconds int) ([]byte, error) {
	cacheKey := fmt.Sprintf("%s:%d", token, seconds)

	s.mu.RLock()
	if cached, ok := s.thumbCache[cacheKey]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	select {
	case s.thumbSem <- struct{}{}:
		defer func() { <-s.thumbSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	s.mu.RLock()
	if cached, ok := s.thumbCache[cacheKey]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	resolved := ws.ResolvedVideoURL
	s.mu.RUnlock()

	if resolved == "" {
		res, err := proxy.ResolveHLSVideoPlaylist(ctx, ws.StreamURL, ws.Headers)
		if err != nil || res == "" {
			res = ws.StreamURL
		}
		s.mu.Lock()
		ws.ResolvedVideoURL = res
		resolved = res
		s.mu.Unlock()
	}

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	localIP := "127.0.0.1"
	localBase := proxy.ProxyBaseURL(localIP, s.port)
	proxyURL := proxy.MakeProxyURL(localBase, resolved, ws.Headers)

	cmdCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, ffmpegPath,
		"-ss", strconv.Itoa(seconds),
		"-i", proxyURL,
		"-vframes", "1",
		"-q:v", "3",
		"-vf", "scale=280:158",
		"-f", "image2",
		"-",
	)

	var buf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg web thumbnail extract error: %w (stderr: %s)", err, strings.TrimSpace(errBuf.String()))
	}

	data := buf.Bytes()
	if len(data) == 0 {
		return nil, fmt.Errorf("no thumbnail data produced")
	}

	s.mu.Lock()
	if len(s.thumbCache) > 500 {
		s.thumbCache = make(map[string][]byte)
	}
	s.thumbCache[cacheKey] = data
	s.mu.Unlock()

	return data, nil
}

func (s *Server) ServeHTTPMedia(w http.ResponseWriter, r *http.Request) {
	// e.g. /media/{token} or /media/{token}/video.mp4 or /media/{token}/thumb?t=120
	trimmed := strings.TrimPrefix(r.URL.Path, "/media/")
	parts := strings.Split(trimmed, "/")
	token := parts[0]

	s.mu.RLock()
	filePath, hasFile := s.fileRegistry[token]
	webStream, hasWeb := s.webRegistry[token]
	s.mu.RUnlock()

	if !hasFile && !hasWeb {
		http.NotFound(w, r)
		return
	}

	// Thumbnail request: /media/{token}/thumb?t=123
	if len(parts) > 1 && parts[1] == "thumb" {
		tSec, _ := strconv.Atoi(r.URL.Query().Get("t"))
		if tSec < 0 {
			tSec = 0
		}
		var thumbData []byte
		var err error
		if hasFile {
			thumbData, err = s.ExtractThumbnail(r.Context(), filePath, tSec)
		} else {
			thumbData, err = s.ExtractWebThumbnail(r.Context(), token, webStream, tSec)
		}
		if err != nil {
			http.Error(w, "thumbnail error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(thumbData)
		return
	}

	if !hasFile {
		http.NotFound(w, r)
		return
	}

	// Set permissive CORS for TV and browser
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeFile(w, r, filePath)
}

func (s *Server) ServeHTTPSubtitle(w http.ResponseWriter, r *http.Request) {
	// e.g. /subtitles/{token}
	token := strings.TrimPrefix(r.URL.Path, "/subtitles/")
	token = strings.TrimSuffix(token, ".srt")
	token = strings.TrimSuffix(token, ".vtt")

	s.mu.RLock()
	filePath, ok := s.subRegistry[token]
	s.mu.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, filePath)
}

func GetOutboundIP(destIP string) string {
	if destIP == "" {
		destIP = "8.8.8.8"
	}
	conn, err := net.DialTimeout("udp", destIP+":80", 2*time.Second)
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func randomToken(length int) string {
	bytes := make([]byte, length)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
