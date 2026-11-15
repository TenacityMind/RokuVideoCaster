// Package proxy provides an HLS transparent proxy for Roku TV playback.
// It fetches upstream HLS playlists with the original browser-captured headers,
// rewrites all relative/absolute segment and sub-playlist URLs to route back
// through the local Mac server, so that Roku never contacts the CDN directly.
// This solves:
//   - CDN blocks on Referer / User-Agent
//   - Relative URL resolution failures in Roku's player
//   - CMAF fMP4 (#EXT-X-MAP + .m4s) segment proxying
package proxy

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var proxyHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// ServeHLS handles GET /proxy/hls?url=<encoded>&hdrs=<base64-json-headers>
// When the URL is a playlist (.m3u8), it fetches it, rewrites all URLs, and
// returns the rewritten playlist. For segment requests (.ts, .m4s, .mp4,
// audio playlists etc.) it byte-pipes the upstream response directly.
func ServeHLS(w http.ResponseWriter, r *http.Request, localBase string) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url param", http.StatusBadRequest)
		return
	}

	// Decode captured browser headers (optional)
	headers := decodeHeaders(r.URL.Query().Get("hdrs"))

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
	if err != nil {
		http.Error(w, "bad upstream url: "+err.Error(), http.StatusBadRequest)
		return
	}
	for k, v := range headers {
		upstreamReq.Header.Set(k, v)
	}
	// Always forward a sensible User-Agent so CDNs don't block
	if upstreamReq.Header.Get("User-Agent") == "" {
		upstreamReq.Header.Set("User-Agent",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}

	resp, err := proxyHTTPClient.Do(upstreamReq)
	if err != nil {
		http.Error(w, "upstream fetch failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	lower := strings.ToLower(rawURL)

	isPlaylist := strings.Contains(lower, ".m3u8") ||
		strings.Contains(ct, "mpegurl") ||
		strings.Contains(ct, "x-mpegURL")

	if isPlaylist {
		// Parse base URL for relative-URL resolution
		baseURL, _ := url.Parse(rawURL)
		hdrsParam := r.URL.Query().Get("hdrs") // pass same headers down to child requests

		rewritten, err := rewritePlaylist(resp.Body, baseURL, localBase, hdrsParam)
		if err != nil {
			http.Error(w, "playlist rewrite failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rewritten))
		return
	}

	// For segments: pipe upstream response directly
	w.Header().Set("Content-Type", ct)
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// rewritePlaylist reads the HLS playlist line-by-line and rewrites all URL
// references (both relative and absolute) to route through the local proxy.
func rewritePlaylist(body io.Reader, baseURL *url.URL, localBase, hdrsParam string) (string, error) {
	var sb strings.Builder
	scanner := bufio.NewScanner(body)

	for scanner.Scan() {
		line := scanner.Text()

		// EXT-X-MAP URI="..."
		if strings.HasPrefix(line, "#EXT-X-MAP:") {
			line = rewriteTagURI(line, "URI", baseURL, localBase, hdrsParam)
			sb.WriteString(line + "\n")
			continue
		}

		// EXT-X-MEDIA URI="..."
		if strings.HasPrefix(line, "#EXT-X-MEDIA:") {
			line = rewriteTagURI(line, "URI", baseURL, localBase, hdrsParam)
			sb.WriteString(line + "\n")
			continue
		}

		// EXT-X-KEY URI="..."
		if strings.HasPrefix(line, "#EXT-X-KEY:") {
			line = rewriteTagURI(line, "URI", baseURL, localBase, hdrsParam)
			sb.WriteString(line + "\n")
			continue
		}

		// Blank / comment lines that are not URLs
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			sb.WriteString(line + "\n")
			continue
		}

		// Bare URL line (segment or sub-playlist)
		abs := resolveURL(line, baseURL)
		proxied := buildProxyURL(localBase, abs, hdrsParam)
		sb.WriteString(proxied + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// rewriteTagURI rewrites the value of a named attribute (e.g. URI) inside an
// HLS tag line like `#EXT-X-MAP:URI="foo.mp4",BYTERANGE="..."`
func rewriteTagURI(line, attr string, baseURL *url.URL, localBase, hdrsParam string) string {
	prefix := attr + "=\""
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return line
	}
	start := idx + len(prefix)
	end := strings.Index(line[start:], "\"")
	if end < 0 {
		return line
	}
	original := line[start : start+end]
	abs := resolveURL(original, baseURL)
	proxied := buildProxyURL(localBase, abs, hdrsParam)
	return line[:start] + proxied + line[start+end:]
}

// resolveURL turns a potentially relative URL into an absolute one using base.
func resolveURL(raw string, base *url.URL) string {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	ref, err := url.Parse(raw)
	if err != nil || base == nil {
		return raw
	}
	return base.ResolveReference(ref).String()
}

// buildProxyURL constructs the local proxy URL for a given upstream URL.
func buildProxyURL(localBase, upstreamURL, hdrsParam string) string {
	params := url.Values{}
	params.Set("url", upstreamURL)
	if hdrsParam != "" {
		params.Set("hdrs", hdrsParam)
	}
	return localBase + "/proxy/hls?" + params.Encode()
}

// MakeProxyURL creates the proxied HLS URL to pass to Roku, encoding any
// captured browser headers so the proxy can forward them upstream.
func MakeProxyURL(localBase, upstreamURL string, headers map[string]string) string {
	hdrsParam := encodeHeaders(headers)
	return buildProxyURL(localBase, upstreamURL, hdrsParam)
}

func encodeHeaders(h map[string]string) string {
	if len(h) == 0 {
		return ""
	}
	b, err := json.Marshal(h)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func decodeHeaders(encoded string) map[string]string {
	if encoded == "" {
		return nil
	}
	b, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}
	var h map[string]string
	if err := json.Unmarshal(b, &h); err != nil {
		return nil
	}
	return h
}

// IsHLSURL returns true if the URL looks like an HLS master/media playlist.
func IsHLSURL(u string) bool {
	l := strings.ToLower(u)
	return strings.Contains(l, ".m3u8")
}

// IsDASHURL returns true if the URL looks like a DASH manifest.
func IsDASHURL(u string) bool {
	l := strings.ToLower(u)
	return strings.Contains(l, ".mpd")
}

// NeedsProxy returns true for stream types that benefit from proxying
// (HLS playlists, DASH manifests). Direct .mp4 files generally work
// without a proxy since Roku can send range requests directly.
func NeedsProxy(streamURL string) bool {
	return IsHLSURL(streamURL) || IsDASHURL(streamURL)
}

// ProxyBaseURL constructs the base proxy URL from localIP and port.
func ProxyBaseURL(localIP string, port int) string {
	return fmt.Sprintf("http://%s:%d", localIP, port)
}
