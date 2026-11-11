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
	URL        string            `json:"url"`
	Quality    string            `json:"quality"`     // e.g. "1080p FHD", "4K UHD", "720p HD", "Auto"
	Format     string            `json:"format"`      // "Direct HLS", "Direct MP4", "DASH"
	Bitrate    string            `json:"bitrate"`     // e.g. "6.2 Mbps", "16.5 Mbps"
	IsSelected bool              `json:"is_selected"`
	Headers    map[string]string `json:"headers,omitempty"`
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
	srcRegex      = regexp.MustCompile(`(?i)<(?:source|video)[^>]+src=["']([^"']+)["']`)
	trackRegex    = regexp.MustCompile(`(?i)<track[^>]+src=["']([^"']+)["'][^>]*>`)
	langRegex     = regexp.MustCompile(`(?i)srclang=["']([^"']+)["']`)
	titleRegex    = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	hlsRegex      = regexp.MustCompile(`(?i)https?://[^\s"'<>]+\.m3u8[^\s"'<>]*`)
	mp4Regex      = regexp.MustCompile(`(?i)https?://[^\s"'<>]+\.mp4[^\s"'<>]*`)
	iframeRegex   = regexp.MustCompile(`(?i)<iframe[^>]+src=["']([^"']+)["']`)
	subFileRegex  = regexp.MustCompile(`(?i)https?://[^\s"'<>]+\.(?:vtt|srt)[^\s"'<>]*`)
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

	// Check if URL query contains an embedded video URL (e.g. ?url=https...m3u8 or ?file=...)
	for _, v := range parsed.Query() {
		for _, qv := range v {
			if strings.Contains(strings.ToLower(qv), ".m3u8") {
				result.Candidates = append(result.Candidates, StreamCandidate{
					URL:        qv,
					Quality:    "Auto Adaptive (Query)",
					Format:     "Direct HLS",
					Bitrate:    "Multi-rate",
					IsSelected: true,
				})
				return result, nil
			} else if strings.Contains(strings.ToLower(qv), ".mp4") {
				result.Candidates = append(result.Candidates, StreamCandidate{
					URL:        qv,
					Quality:    "Direct MP4 (Query)",
					Format:     "Direct MP4",
					Bitrate:    "Original",
					IsSelected: true,
				})
				return result, nil
			}
		}
	}

	// 2. Automated Headless Browser Scraper (ChromeDP Engine with Stealth & Turnstile bypass)
	// Opens the page in headless Chrome, executes client-side scripts, navigates player iframes,
	// clicks play buttons, and sniffs outgoing video requests (.m3u8, .mp4, DASH, subtitles).
	browserResult, browserErr := ExtractWithBrowser(ctx, targetURL)
	if browserErr == nil && len(browserResult.Candidates) > 0 {
		return browserResult, nil
	}

	// 3. Fallback: Static HTML / regex extraction (when Chrome is unavailable or page has raw media tags)
	seenUrls := make(map[string]bool)
	seenSubs := make(map[string]bool)

	html, err := fetchHTML(ctx, targetURL)
	if err == nil {
		// Extract Title
		if match := titleRegex.FindStringSubmatch(html); len(match) > 1 {
			result.Title = strings.TrimSpace(match[1])
		}

		extractMediaFromHTML(targetURL, html, result, seenUrls, seenSubs)

		// If no streams found on main page, scan iframes (e.g. video embed providers)
		if len(result.Candidates) == 0 {
			iframes := iframeRegex.FindAllStringSubmatch(html, -1)
			count := 0
			for _, m := range iframes {
				if count >= 3 {
					break
				}
				rawSrc := m[1]
				if strings.HasPrefix(rawSrc, "//") {
					rawSrc = "https:" + rawSrc
				}
				frameURL := resolveURL(targetURL, rawSrc)
				lFrame := strings.ToLower(frameURL)
				if strings.Contains(lFrame, "embed") || strings.Contains(lFrame, "player") || strings.Contains(lFrame, "video") || strings.Contains(lFrame, "stream") || strings.Contains(lFrame, "watch") {
					count++
					frameHTML, fErr := fetchHTML(ctx, frameURL)
					if fErr == nil {
						extractMediaFromHTML(frameURL, frameHTML, result, seenUrls, seenSubs)
						if len(result.Candidates) > 0 {
							break
						}
					}
				}
			}
		}
	}

	// If no video candidate was found on the page
	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no direct video stream (.m3u8 / .mp4) could be extracted from this page. If this site requires authentication or custom interaction, you can inspect the Network tab and paste the direct stream link")
	}
	result.Candidates[0].IsSelected = true

	return result, nil
}

func fetchHTML(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB max
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func extractMediaFromHTML(baseURL, html string, result *AnalysisResult, seenUrls map[string]bool, seenSubs map[string]bool) {
	// Normalize escaped slashes in JSON objects (e.g. "https:\/\/...")
	unescapedHTML := strings.ReplaceAll(html, `\/`, `/`)

	// Extract HLS links
	for _, hls := range hlsRegex.FindAllString(unescapedHTML, -1) {
		hls = strings.Trim(hls, `"' `)
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

	// Extract MP4 links
	for _, mp4 := range mp4Regex.FindAllString(unescapedHTML, -1) {
		mp4 = strings.Trim(mp4, `"' `)
		if !seenUrls[mp4] {
			seenUrls[mp4] = true
			quality := "1080p FHD"
			bitrate := "6.2 Mbps"
			lMp4 := strings.ToLower(mp4)
			if strings.Contains(lMp4, "2160") || strings.Contains(lMp4, "4k") {
				quality = "4K UHD"
				bitrate = "16.5 Mbps"
			} else if strings.Contains(lMp4, "720") {
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

	// Extract <source> and <video> tags
	for _, match := range srcRegex.FindAllStringSubmatch(unescapedHTML, -1) {
		rawSrc := match[1]
		fullSrc := resolveURL(baseURL, rawSrc)
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
	for _, match := range trackRegex.FindAllStringSubmatch(unescapedHTML, -1) {
		rawTrack := match[1]
		fullTrack := resolveURL(baseURL, rawTrack)
		if !seenSubs[fullTrack] {
			seenSubs[fullTrack] = true
			lang := "EN"
			if lMatch := langRegex.FindStringSubmatch(match[0]); len(lMatch) > 1 {
				lang = strings.ToUpper(lMatch[1])
			}
			result.Subtitles = append(result.Subtitles, SubtitleOption{
				URL:      fullTrack,
				Language: lang,
				Format:   "vtt",
			})
		}
	}

	// Extract direct subtitle file links (.vtt, .srt) found in scripts
	for _, sub := range subFileRegex.FindAllString(unescapedHTML, -1) {
		sub = strings.Trim(sub, `"' `)
		if !seenSubs[sub] {
			seenSubs[sub] = true
			ext := "vtt"
			if strings.HasSuffix(strings.ToLower(sub), ".srt") {
				ext = "srt"
			}
			result.Subtitles = append(result.Subtitles, SubtitleOption{
				URL:      sub,
				Language: "Auto",
				Format:   ext,
			})
		}
	}
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
