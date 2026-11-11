package roku

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/icholy/digest"
)

//go:embed channel.zip
var embeddedChannelZip []byte

const (
	devUser        = "rokudev"
	installTimeout = 60 * time.Second
)

type SideloadResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func SideloadChannel(ctx context.Context, host, password string) (*SideloadResult, error) {
	if host == "" {
		return nil, fmt.Errorf("roku host IP is required")
	}
	if password == "" {
		return nil, fmt.Errorf("roku developer password is required")
	}

	cleanHost := strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")
	if strings.Contains(cleanHost, ":") {
		cleanHost, _, _ = net.SplitHostPort(cleanHost)
	}
	installURL := "http://" + net.JoinHostPort(cleanHost, "80") + "/plugin_install"

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("mysubmit", "Install"); err != nil {
		return nil, err
	}
	part, err := w.CreateFormFile("archive", "roku_video_caster.zip")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(embeddedChannelZip); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	client := &http.Client{
		Transport: &digest.Transport{
			Username: devUser,
			Password: password,
		},
	}

	// Pre-flight GET request to establish digest nonce before uploading large payload
	preReq, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, installURL, nil)
	if preResp, err := client.Do(preReq); err == nil {
		preResp.Body.Close()
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, installURL, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect to roku dev portal (port 80): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("developer password rejected by Roku TV")
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("plugin_install returned HTTP %d", resp.StatusCode)
	}

	page, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	text := strings.ToLower(string(page))
	if strings.Contains(text, "install success") || strings.Contains(text, "identical to previous version") {
		return &SideloadResult{
			Success: true,
			Message: "Channel installed successfully on Roku TV",
		}, nil
	}

	snippet := strings.TrimSpace(string(page))
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	return nil, fmt.Errorf("roku rejected package: %s", snippet)
}
