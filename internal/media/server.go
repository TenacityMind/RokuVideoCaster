package media

import (
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
	"strings"
	"sync"
	"time"
)

type Server struct {
	mu            sync.RWMutex
	fileRegistry  map[string]string // token -> absolute file path
	subRegistry   map[string]string // token -> absolute subtitle path
	extractedSubs map[string]string // cache key -> extracted srt path
	port          int
}

func NewServer(port int) *Server {
	return &Server{
		fileRegistry:  make(map[string]string),
		subRegistry:   make(map[string]string),
		extractedSubs: make(map[string]string),
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

func (s *Server) ServeHTTPMedia(w http.ResponseWriter, r *http.Request) {
	// e.g. /media/{token} or /media/{token}/video.mp4
	trimmed := strings.TrimPrefix(r.URL.Path, "/media/")
	parts := strings.Split(trimmed, "/")
	token := parts[0]

	s.mu.RLock()
	filePath, ok := s.fileRegistry[token]
	s.mu.RUnlock()

	if !ok {
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
