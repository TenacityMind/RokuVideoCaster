package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type StreamTrack struct {
	Index       int    `json:"index"`
	Type        string `json:"type"` // "audio" or "subtitle"
	Codec       string `json:"codec"`
	Title       string `json:"title"`
	Language    string `json:"language"`
	Channels    int    `json:"channels,omitempty"`
	ChannelDesc string `json:"channel_desc,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

type MediaInfo struct {
	FilePath    string        `json:"file_path"`
	FileName    string        `json:"file_name"`
	FileSize    int64         `json:"file_size"`
	DurationSec float64       `json:"duration_sec"`
	Resolution  string        `json:"resolution"`
	VideoCodec  string        `json:"video_codec"`
	AudioTracks []StreamTrack `json:"audio_tracks"`
	SubTracks   []StreamTrack `json:"sub_tracks"`
}

type ffprobeOutput struct {
	Streams []struct {
		Index     int               `json:"index"`
		CodecName string            `json:"codec_name"`
		CodecType string            `json:"codec_type"`
		Width     int               `json:"width"`
		Height    int               `json:"height"`
		Channels  int               `json:"channels"`
		Tags      map[string]string `json:"tags"`
		Dispo     map[string]int    `json:"disposition"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
}

func ResolveFilePath(path string) string {
	if path == "" {
		return ""
	}
	// 1. Direct path check
	if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}

	// 2. Expand home directory prefix (~/...)
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			expanded := filepath.Join(home, path[2:])
			if stat, err := os.Stat(expanded); err == nil && !stat.IsDir() {
				return expanded
			}
		}
	}

	// 3. Search common user directories for the filename
	base := filepath.Base(path)
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	candidates := []string{
		filepath.Join(home, "Downloads", base),
		filepath.Join(home, "Movies", base),
		filepath.Join(home, "Desktop", base),
		filepath.Join(home, "Videos", base),
		filepath.Join(home, "Documents", base),
	}
	for _, c := range candidates {
		if stat, err := os.Stat(c); err == nil && !stat.IsDir() {
			return c
		}
	}

	// 4. Also check immediate subdirectories in Downloads and Movies
	searchDirs := []string{
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Movies"),
	}
	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				subCandidate := filepath.Join(dir, entry.Name(), base)
				if stat, err := os.Stat(subCandidate); err == nil && !stat.IsDir() {
					return subCandidate
				}
			}
		}
	}

	return path
}

func ProbeMediaFile(ctx context.Context, filePath string) (*MediaInfo, error) {
	filePath = ResolveFilePath(filePath)
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat media file: %w", err)
	}

	info := &MediaInfo{
		FilePath: filePath,
		FileName: filepath.Base(filePath),
		FileSize: stat.Size(),
	}

	// Find ffprobe in PATH
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		// Fallback to basic info if ffprobe isn't installed
		info.AudioTracks = []StreamTrack{
			{Index: 0, Type: "audio", Codec: "Direct Passthrough", Title: "Track 1: Primary Audio", Channels: 2, ChannelDesc: "Stereo / Surround", IsDefault: true},
		}
		return info, nil
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxTimeout, ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	out, err := cmd.Output()
	if err != nil {
		// If probe failed, return basic info without failing
		return info, nil
	}

	var parsed ffprobeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return info, nil
	}

	if dur, err := strconv.ParseFloat(parsed.Format.Duration, 64); err == nil {
		info.DurationSec = dur
	}

	for _, s := range parsed.Streams {
		lang := s.Tags["language"]
		if lang == "" {
			lang = "und"
		}
		title := s.Tags["title"]

		switch s.CodecType {
		case "video":
			if info.Resolution == "" && s.Width > 0 && s.Height > 0 {
				info.VideoCodec = s.CodecName
				resTag := fmt.Sprintf("%dx%d", s.Width, s.Height)
				if s.Width >= 3800 || s.Height >= 2100 {
					resTag += " (4K UHD)"
				} else if s.Width >= 1900 || s.Height >= 1000 {
					resTag += " (1080p FHD)"
				} else if s.Width >= 1200 || s.Height >= 700 {
					resTag += " (720p HD)"
				}
				info.Resolution = resTag
			}

		case "audio":
			isDef := s.Dispo["default"] == 1
			chDesc := "Stereo"
			if s.Channels == 6 {
				chDesc = "5.1 Surround"
			} else if s.Channels == 8 {
				chDesc = "7.1 Surround / Atmos"
			} else if s.Channels == 1 {
				chDesc = "Mono"
			}

			trackTitle := title
			if trackTitle == "" {
				trackTitle = fmt.Sprintf("Track %d: %s (%s, %s)", len(info.AudioTracks)+1, strings.ToUpper(lang), strings.ToUpper(s.CodecName), chDesc)
			}

			info.AudioTracks = append(info.AudioTracks, StreamTrack{
				Index:       s.Index,
				Type:        "audio",
				Codec:       s.CodecName,
				Title:       trackTitle,
				Language:    lang,
				Channels:    s.Channels,
				ChannelDesc: chDesc,
				IsDefault:   isDef,
			})

		case "subtitle":
			isDef := s.Dispo["default"] == 1
			subTitle := title
			if subTitle == "" {
				subTitle = fmt.Sprintf("Sub %d: %s (%s)", len(info.SubTracks)+1, strings.ToUpper(lang), s.CodecName)
			}
			info.SubTracks = append(info.SubTracks, StreamTrack{
				Index:     s.Index,
				Type:      "subtitle",
				Codec:     s.CodecName,
				Title:     subTitle,
				Language:  lang,
				IsDefault: isDef,
			})
		}
	}

	return info, nil
}
