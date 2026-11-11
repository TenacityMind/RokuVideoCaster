package scraper

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/TenacityMind/RokuVideoCaster/internal/scraper/stealth"
)

//go:embed js/player_pos.js
var playerPosJS string

//go:embed js/iframe_src.js
var iframeSrcJS string

//go:embed js/turnstile_iframe_pos.js
var turnstileIframePosTemplate string

//go:embed js/turnstile_gone.js
var turnstileGoneTemplate string

const turnstileSelector = ".cf-turnstile"

var (
	turnstileIframePosJS = strings.ReplaceAll(turnstileIframePosTemplate, "__TURNSTILE__", turnstileSelector)
	turnstileGoneJS      = strings.ReplaceAll(turnstileGoneTemplate, "__TURNSTILE__", turnstileSelector)
	consoleURLRegex      = regexp.MustCompile(`https?://[^\s"'<>]+`)
)

type frameCandidate struct {
	Src  string  `json:"src"`
	Area float64 `json:"area"`
}

func ExtractWithBrowser(parentCtx context.Context, targetURL string) (*AnalysisResult, error) {
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	profile := stealth.New()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("webrtc-ip-handling-policy", "disable_non_proxied_udp"),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.WindowSize(1920, 1080),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	result := &AnalysisResult{
		PageURL: targetURL,
		Title:   "Online Stream",
	}

	var mu sync.Mutex
	seenMedia := make(map[string]bool)
	seenSubs := make(map[string]bool)
	capturedCandidates := make([]StreamCandidate, 0)
	capturedSubtitles := make([]SubtitleOption, 0)
	streamFound := make(chan struct{}, 1)

	// Stream collection listener
	chromedp.ListenTarget(taskCtx, func(ev any) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			reqURL := e.Request.URL
			if isMediaURL(reqURL) && !isAdURL(reqURL) {
				mu.Lock()
				if !seenMedia[reqURL] {
					seenMedia[reqURL] = true
					format := "Direct HLS"
					quality := "1080p FHD"
					if strings.Contains(strings.ToLower(reqURL), ".mp4") {
						format = "Direct MP4"
					} else if strings.Contains(strings.ToLower(reqURL), ".mpd") {
						format = "DASH"
					}
					headers := make(map[string]string)
					for k, v := range e.Request.Headers {
						if strVal, ok := v.(string); ok {
							headers[k] = strVal
						}
					}
					capturedCandidates = append(capturedCandidates, StreamCandidate{
						URL:        reqURL,
						Quality:    quality,
						Format:     format,
						Bitrate:    "Multi-rate Adaptive",
						IsSelected: len(capturedCandidates) == 0,
						Headers:    headers,
					})
					select {
					case streamFound <- struct{}{}:
					default:
					}
				}
				mu.Unlock()
			} else if isSubtitleURL(reqURL) {
				mu.Lock()
				if !seenSubs[reqURL] {
					seenSubs[reqURL] = true
					ext := "vtt"
					if strings.HasSuffix(strings.ToLower(reqURL), ".srt") {
						ext = "srt"
					}
					capturedSubtitles = append(capturedSubtitles, SubtitleOption{
						URL:      reqURL,
						Language: "Auto",
						Format:   ext,
					})
				}
				mu.Unlock()
			}

		case *network.EventResponseReceived:
			respURL := e.Response.URL
			mime := strings.ToLower(e.Response.MimeType)
			if isMediaMIME(mime) && !isAdURL(respURL) {
				mu.Lock()
				if !seenMedia[respURL] {
					seenMedia[respURL] = true
					format := "Direct HLS"
					if strings.Contains(mime, "mp4") {
						format = "Direct MP4"
					} else if strings.Contains(mime, "dash") {
						format = "DASH"
					}
					capturedCandidates = append(capturedCandidates, StreamCandidate{
						URL:        respURL,
						Quality:    "1080p FHD",
						Format:     format,
						Bitrate:    "Adaptive Stream",
						IsSelected: len(capturedCandidates) == 0,
					})
					select {
					case streamFound <- struct{}{}:
					default:
					}
				}
				mu.Unlock()
			}

		case *runtime.EventConsoleAPICalled:
			for _, arg := range e.Args {
				var text string
				if json.Unmarshal(arg.Value, &text) != nil {
					continue
				}
				for _, raw := range consoleURLRegex.FindAllString(text, -1) {
					if isMediaURL(raw) && !isAdURL(raw) {
						mu.Lock()
						if !seenMedia[raw] {
							seenMedia[raw] = true
							capturedCandidates = append(capturedCandidates, StreamCandidate{
								URL:        raw,
								Quality:    "1080p FHD",
								Format:     "Direct HLS",
								Bitrate:    "Adaptive Stream",
								IsSelected: len(capturedCandidates) == 0,
							})
							select {
							case streamFound <- struct{}{}:
							default:
							}
						}
						mu.Unlock()
					}
				}
			}
		}
	})

	// Initial browser setup with stealth profile
	if err := chromedp.Run(taskCtx,
		runtime.Enable(),
		network.Enable(),
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorDeny),
		profile.Apply(),
	); err != nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	// Navigate to target URL
	navCtx, navCancel := context.WithTimeout(taskCtx, 15*time.Second)
	_ = chromedp.Run(navCtx, chromedp.Navigate(targetURL))
	navCancel()

	// Retrieve page title
	var pageTitle string
	_ = chromedp.Run(taskCtx, chromedp.Title(&pageTitle))
	if pageTitle != "" {
		result.Title = pageTitle
	}

	// Action sequence to trigger video playback
	actions := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"click player", func(c context.Context) error { return clickPlayer(c) }},
		{"click center", func(c context.Context) error { return clickAt(c, 960, 540) }},
		{"solve turnstile", func(c context.Context) error { return solveTurnstile(c) }},
		{"navigate iframes", func(c context.Context) error { return navigateIframes(c, targetURL) }},
		{"click player again", func(c context.Context) error { return clickPlayer(c) }},
	}

	for _, act := range actions {
		mu.Lock()
		foundCount := len(capturedCandidates)
		mu.Unlock()
		if foundCount > 0 {
			break
		}

		actCtx, actCancel := context.WithTimeout(taskCtx, 6*time.Second)
		_ = act.fn(actCtx)
		actCancel()

		select {
		case <-streamFound:
			// Stream captured!
			break
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			break
		}
	}

	// Wait up to 5 seconds for any final stream manifest if none captured yet
	mu.Lock()
	initialFound := len(capturedCandidates) > 0
	mu.Unlock()

	if !initialFound {
		select {
		case <-streamFound:
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
		}
	}

	mu.Lock()
	result.Candidates = capturedCandidates
	result.Subtitles = capturedSubtitles
	mu.Unlock()

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no stream captured by browser automation")
	}

	result.Candidates[0].IsSelected = true
	return result, nil
}

func isMediaURL(raw string) bool {
	l := strings.ToLower(raw)
	return strings.Contains(l, ".m3u8") ||
		strings.Contains(l, ".mpd") ||
		strings.Contains(l, ".mp4") ||
		strings.Contains(l, ".m4v") ||
		strings.Contains(l, ".webm") ||
		strings.Contains(l, ".mkv")
}

func isSubtitleURL(raw string) bool {
	l := strings.ToLower(raw)
	return strings.Contains(l, ".vtt") || strings.Contains(l, ".srt")
}

func isMediaMIME(mime string) bool {
	l := strings.ToLower(mime)
	return strings.Contains(l, "mpegurl") ||
		strings.Contains(l, "dash+xml") ||
		strings.Contains(l, "video/mp4") ||
		strings.Contains(l, "video/webm") ||
		strings.Contains(l, "video/x-matroska")
}

func isAdURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	p := strings.ToLower(u.Path)
	return strings.Contains(p, "preroll") ||
		strings.Contains(p, "midroll") ||
		strings.Contains(p, "postroll") ||
		strings.Contains(p, "/ad/") ||
		strings.Contains(p, "/ads/")
}

func clickAt(ctx context.Context, x, y float64) error {
	return chromedp.Run(ctx, chromedp.MouseClickXY(x, y, chromedp.ButtonLeft))
}

func clickPlayer(ctx context.Context) error {
	var pos *struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(playerPosJS, &pos)); err != nil || pos == nil {
		return err
	}
	return clickAt(ctx, pos.X, pos.Y)
}

func solveTurnstile(ctx context.Context) error {
	var present bool
	if err := chromedp.Run(ctx, chromedp.Evaluate("!("+turnstileGoneJS+")", &present)); err != nil || !present {
		return nil
	}
	var pos struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := chromedp.Run(ctx, chromedp.Poll(turnstileIframePosJS, &pos, chromedp.WithPollingTimeout(0))); err == nil && (pos.X > 0 || pos.Y > 0) {
		return clickAt(ctx, pos.X, pos.Y)
	}
	return nil
}

func navigateIframes(ctx context.Context, parentURL string) error {
	var frames []frameCandidate
	pollCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = chromedp.Run(pollCtx, chromedp.Poll(iframeSrcJS, &frames, chromedp.WithPollingTimeout(0)))
	if len(frames) == 0 {
		return nil
	}

	// Try largest player iframe
	best := frames[0]
	if best.Src == "" {
		return nil
	}

	iframeCtx, ifCancel := context.WithTimeout(ctx, 8*time.Second)
	defer ifCancel()

	_ = chromedp.Run(iframeCtx, chromedp.ActionFunc(func(c context.Context) error {
		_, _, _, _, err := page.Navigate(best.Src).
			WithReferrer(parentURL).
			WithReferrerPolicy(page.ReferrerPolicyStrictOriginWhenCrossOrigin).
			Do(c)
		return err
	}))

	_ = clickPlayer(iframeCtx)
	_ = clickAt(iframeCtx, 960, 540)
	return nil
}
