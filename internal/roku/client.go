package roku

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ecpPort        = "8060"
	defaultTimeout = 5 * time.Second
)

type DeviceInfoXML struct {
	XMLName           xml.Name `xml:"device-info"`
	UserDeviceName    string   `xml:"user-device-name"`
	FriendlyModelName string   `xml:"friendly-model-name"`
	ModelName         string   `xml:"model-name"`
	ModelNumber       string   `xml:"model-number"`
	SoftwareVersion   string   `xml:"software-version"`
	SerialNumber      string   `xml:"serial-number"`
	Uptime            int64    `xml:"uptime"`
	PowerMode         string   `xml:"power-mode"`
}

type DeviceStatus struct {
	Online     bool   `json:"online"`
	DeviceName string `json:"device_name"`
	Model      string `json:"model"`
	Version    string `json:"version"`
	IP         string `json:"ip"`
	IsDevOpen  bool   `json:"is_dev_open"`
}

type ActiveAppXML struct {
	XMLName xml.Name `xml:"active-app"`
	App     struct {
		ID   string `xml:"id,attr"`
		Name string `xml:",chardata"`
	} `xml:"app"`
}

type LaunchParams struct {
	MediaURL    string  `json:"media_url"`
	SubURL      string  `json:"sub_url"`
	Title       string  `json:"title"`
	SubOffset   float64 `json:"sub_offset"`
	AudioTrack  int     `json:"audio_track"`
	CallbackURL string  `json:"callback_url"`
	WebURL      string  `json:"web_url"`
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) TestConnection(ctx context.Context, host string) (*DeviceStatus, error) {
	if host == "" {
		return nil, fmt.Errorf("roku host IP is required")
	}

	cleanHost := strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")
	if strings.Contains(cleanHost, ":") {
		cleanHost, _, _ = net.SplitHostPort(cleanHost)
	}

	queryURL := fmt.Sprintf("http://%s:%s/query/device-info", cleanHost, ecpPort)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &DeviceStatus{
			Online: false,
			IP:     cleanHost,
		}, fmt.Errorf("connect to Roku ECP port 8060: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &DeviceStatus{
			Online: false,
			IP:     cleanHost,
		}, fmt.Errorf("device-info returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info DeviceInfoXML
	if err := xml.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse device info: %w", err)
	}

	name := info.UserDeviceName
	if name == "" {
		name = info.FriendlyModelName
	}
	if name == "" {
		name = info.ModelName
	}
	if name == "" {
		name = "Roku TV"
	}

	// Check if dev channel is currently active
	isDevOpen := false
	if active, err := c.GetActiveApp(ctx, cleanHost); err == nil && active == "dev" {
		isDevOpen = true
	}

	return &DeviceStatus{
		Online:     true,
		DeviceName: name,
		Model:      info.ModelName + " (" + info.ModelNumber + ")",
		Version:    info.SoftwareVersion,
		IP:         cleanHost,
		IsDevOpen:  isDevOpen,
	}, nil
}

func (c *Client) GetActiveApp(ctx context.Context, host string) (string, error) {
	queryURL := fmt.Sprintf("http://%s:%s/query/active-app", host, ecpPort)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var app ActiveAppXML
	if err := xml.Unmarshal(body, &app); err != nil {
		return "", err
	}
	return app.App.ID, nil
}

func (c *Client) Launch(ctx context.Context, host string, params LaunchParams) error {
	// Build query params
	q := url.Values{}
	q.Set("url", params.MediaURL)
	q.Set("media_url", params.MediaURL)
	if params.SubURL != "" {
		q.Set("subtitles", params.SubURL)
		q.Set("sub_url", params.SubURL)
	}
	if params.Title != "" {
		q.Set("title", params.Title)
	}
	if params.SubOffset != 0 {
		q.Set("sub_offset", fmt.Sprintf("%.2f", params.SubOffset))
	}
	if params.AudioTrack > 0 {
		q.Set("audio_track", fmt.Sprintf("%d", params.AudioTrack))
	}
	if params.CallbackURL != "" {
		q.Set("callback_url", params.CallbackURL)
	}
	if params.WebURL != "" {
		q.Set("web_url", params.WebURL)
	}

	// 1. Launch dev channel (brings app forward & passes deep link args)
	launchEndpoint := fmt.Sprintf("http://%s:%s/launch/dev?%s", host, ecpPort, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, launchEndpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("launch on Roku TV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("launch returned HTTP %d", resp.StatusCode)
	}

	// 2. Also send /input event so if channel was already in message loop, it receives input instantly
	inputEndpoint := fmt.Sprintf("http://%s:%s/input?%s", host, ecpPort, q.Encode())
	if inReq, err := http.NewRequestWithContext(ctx, http.MethodPost, inputEndpoint, nil); err == nil {
		if inResp, err := c.httpClient.Do(inReq); err == nil {
			_ = inResp.Body.Close()
		}
	}

	return nil
}

func (c *Client) SendKey(ctx context.Context, host, key string) error {
	endpoint := fmt.Sprintf("http://%s:%s/keypress/%s", host, ecpPort, url.PathEscape(key))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("keypress: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("keypress returned HTTP %d", resp.StatusCode)
	}
	return nil
}
