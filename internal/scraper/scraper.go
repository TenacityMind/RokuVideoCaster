package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type StreamCandidate struct {
	URL        string `json:"url"`
	Quality    string `json:"quality"`     // e.g. "1080p FHD", "4K UHD", "720p HD", "Auto"
	Format     string `json:"format"`      // "Direct HLS", "Direct MP4", "DASH"
	Bitrate    string `json:"bitrate"`     // e.g. "6.2 Mbps", "16.5 Mbps"
	IsSelected bool   `json:"is_selected"`
}

type SubtitleOption struct {
	URL      string `json:"url"`
	Language string `json:"language"`
	Format   string `json:"format"` // "srt", "vtt"
}

type AnalysisResult struct {
	PageURL    string            `json:"page_url"`
	Title      string            `json:"title"`
	Candidates []StreamCandidate `json:"candidates"`
	Subtitles  []SubtitleOption  `json:"subtitles"`
}

var (
	srcRegex   = regexp.MustCompile(`(?i)<(?:source|video)[^>]+src=["']([^"']+)["']`)
	trackRegex = regexp.MustCompile(`(?i)<track[^>]+src=["']([^"']+)["'][^>]*>`)
	langRegex  = regexp.MustCompile(`(?i)srclang=["']([^"']+)["']`)
	titleRegex = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	hlsRegex   = regexp.MustCompile(`(?i)https?://[^\s"'<>]+\.m3u8[^\s"'<>]*`)
	mp4Regex   = regexp.MustCompile(`(?i)https?://[^\s"'<>]+\.mp4[^\s"'<>]*`)
)

func AnalyzeVideoURL(ctx context.Context, targetURL string) (*AnalysisResult, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	result := &AnalysisResult{
		PageURL: targetURL,
		Title:   parsed.Host,
	}

	// 1. Direct media link check (.m3u8, .mp4, .mkv, .webm)
	lower := strings.ToLower(parsed.Path + "?" + parsed.RawQuery)
	if strings.Contains(lower, ".m3u8") {
		result.Candidates = append(result.Candidates, StreamCandidate{
			URL:        targetURL,
			Quality:    "Auto Adaptive (1080p/4K)",
			Format:     "Direct HLS",
			Bitrate:    "Multi-rate",
			IsSelected: true,
		})
		return result, nil
	} else if strings.Contains(lower, ".mp4") || strings.Contains(lower, ".m4v") || strings.Contains(lower, ".mkv") || strings.Contains(lower, ".webm") {
		result.Candidates = append(result.Candidates, StreamCandidate{
			URL:        targetURL,
			Quality:    "Direct Video Source",
			Format:     "Direct Stream",
			Bitrate:    "Original",
			IsSelected: true,
		})
		return result, nil
	}

	// 2. Fetch web page content
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB max
	if err != nil {
		return nil, err
	}
	html := string(body)

	// Extract Title
	if match := titleRegex.FindStringSubmatch(html); len(match) > 1 {
		result.Title = strings.TrimSpace(match[1])
	}

	// Extract HLS and MP4 links
	seenUrls := make(map[string]bool)

	for _, hls := range hlsRegex.FindAllString(html, -1) {
		if !seenUrls[hls] {
			seenUrls[hls] = true
			result.Candidates = append(result.Candidates, StreamCandidate{
				URL:     hls,
				Quality: "1080p FHD",
				Format:  "Direct HLS",
				Bitrate: "Adaptive",
			})
		}
	}

	for _, mp4 := range mp4Regex.FindAllString(html, -1) {
		if !seenUrls[mp4] {
			seenUrls[mp4] = true
			quality := "1080p FHD"
			bitrate := "6.2 Mbps"
			if strings.Contains(strings.ToLower(mp4), "2160") || strings.Contains(strings.ToLower(mp4), "4k") {
				quality = "4K UHD"
				bitrate = "16.5 Mbps"
			} else if strings.Contains(strings.ToLower(mp4), "720") {
				quality = "720p HD"
				bitrate = "2.8 Mbps"
			}
			result.Candidates = append(result.Candidates, StreamCandidate{
				URL:     mp4,
				Quality: quality,
				Format:  "Direct MP4",
				Bitrate: bitrate,
			})
		}
	}

	// Extract <source> tags
	for _, match := range srcRegex.FindAllStringSubmatch(html, -1) {
		rawSrc := match[1]
		fullSrc := resolveURL(targetURL, rawSrc)
		if !seenUrls[fullSrc] {
			seenUrls[fullSrc] = true
			format := "Direct Video"
			if strings.Contains(fullSrc, ".m3u8") {
				format = "Direct HLS"
			} else if strings.Contains(fullSrc, ".mp4") {
				format = "Direct MP4"
			}
			result.Candidates = append(result.Candidates, StreamCandidate{
				URL:     fullSrc,
				Quality: "Source Stream",
				Format:  format,
				Bitrate: "Default",
			})
		}
	}

	// Extract <track> subtitles
	for _, match := range trackRegex.FindAllStringSubmatch(html, -1) {
		rawTrack := match[1]
		fullTrack := resolveURL(targetURL, rawTrack)
		lang := "en"
		if lMatch := langRegex.FindStringSubmatch(match[0]); len(lMatch) > 1 {
			lang = lMatch[1]
		}
		result.Subtitles = append(result.Subtitles, SubtitleOption{
			URL:      fullTrack,
			Language: strings.ToUpper(lang),
			Format:   "vtt",
		})
	}

	// If no video candidate was found on the page
	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no direct video stream (.m3u8 / .mp4) found on this page. If this site uses a custom embedded player, please copy the direct stream link (or inspect the Network tab for .m3u8), or stream the video file using 'Personal Local Files'")
	}
	result.Candidates[0].IsSelected = true

	return result, nil
}

func resolveURL(baseURL, refURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return refURL
	}
	ref, err := url.Parse(refURL)
	if err != nil {
		return refURL
	}
	return base.ResolveReference(ref).String()
}
