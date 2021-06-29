package exporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0"

// newRequest creates a new HTTP request and prefills its header.
func (c *Client) fetch(ctx context.Context, method, path string, params url.Values) (*http.Response, error) {
	var u2 url.URL
	u2 = *c.instance // dup
	u2.Path = "/0/cn-srv"
	if len(path) > 0 && path[0] != '/' {
		u2.Path += "/"
	}
	u2.Path += path
	u2.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u2.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	// bypass canonicalization
	req.Header["x-cidx"] = []string{"0"}
	if token := c.getCsrfToken(); token != "" {
		req.Header["X-XSRF-TOKEN"] = []string{token}
	}

	return c.client.Do(req)
}

// fetchCSRFToken is needed to get the initial CSRF cookie, needed for
// most operations.
func (c *Client) fetchCSRFToken(ctx context.Context) error {
	res, err := c.fetch(ctx, http.MethodGet, "/user/me", nil)
	if err != nil {
		return fmt.Errorf("failed to fetch CSRF token: %w", err)
	}
	res.Body.Close()

	return nil
}

// fetchAPGroups returns the list of WiFi AP group names.
func (c *Client) fetchAPGroups(ctx context.Context) ([]string, error) {
	if c.getCsrfToken() == "" {
		if err := c.fetchCSRFToken(ctx); err != nil {
			return nil, err
		}
	}

	res, err := c.fetch(ctx, http.MethodGet, "/config/profiles", url.Values{
		"fields": {"name,hasDevices"},
		"limit":  {"0"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AP group list: %w", err)
	}

	defer res.Body.Close()
	var data struct {
		Data struct {
			Profiles []struct {
				Name string `json:"name"`
			} `json:"profiles"`
		} `json:"data"`
	}

	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode AP group response data: %w", err)
	}

	groups := make([]string, 0, len(data.Data.Profiles))
	for _, p := range data.Data.Profiles {
		groups = append(groups, p.Name)
	}

	return groups, nil
}

var fetchDevFields = []string{
	"$inventory",
	"model", "mac", "tid", "sn", "sys.online", "sys.upTime", "sys.dnTime", "sys.lastRbt.uTs",
	"sys.lastRbt.code", "mgmt.actSw", "cfg.name", "lstUpd", "radio.mac",
	"radio.MIRTName", "config.profile:%s",

	"$radios",
	"id", "rxAvg", "txAvg", "band", "radios.mac", "channel", "chWidth", "rfqlt", "pow",
}

func (c *Client) fetchDevices(ctx context.Context, apGroup string) ([]*Device, error) {
	path := fmt.Sprintf("/stats/profiles/%s/devices", apGroup)
	fields := fmt.Sprintf(strings.Join(fetchDevFields, ","), apGroup)

	res, err := c.fetch(ctx, http.MethodGet, path, url.Values{
		"all":      {"true"},
		"fields":   {fields},
		"limit":    {"0"},
		"offset":   {"0"},
		"sortedBy": {"cfg.name"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch devices for AP group %q: %w", apGroup, err)
	}

	defer res.Body.Close()
	var data struct {
		Data struct {
			Profiles struct {
				Devices []DeviceAPIResponse `json:"devices"`
			} `json:"profiles"`
		} `json:"data"`
	}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode device response data for AP group %q: %w", apGroup, err)
	}

	devs := make([]*Device, 0, len(data.Data.Profiles.Devices))
	for _, dev := range data.Data.Profiles.Devices {
		devs = append(devs, dev.Normalize())
	}

	return devs, nil
}

func (c *Client) fetchAPGroupData(ctx context.Context, apGroup string) (*APGroupAPIResponse, error) {
	fields := fmt.Sprintf("name,deviceCount,offlineCount,clientCount,clientCount24h,name:%s", apGroup)
	res, err := c.fetch(ctx, http.MethodGet, "/config/profiles", url.Values{"fields": {fields}})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch devices for AP group %q: %w", apGroup, err)
	}

	defer res.Body.Close()
	var buf bytes.Buffer
	tee := io.TeeReader(res.Body, &buf)

	var data struct {
		Data struct {
			Profiles []APGroupAPIResponse `json:"profiles"`
		} `json:"data"`
	}
	if err = json.NewDecoder(tee).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode device response data for AP group %q: %w", apGroup, err)
	}
	if len(data.Data.Profiles) == 0 {
		return nil, nil
	}

	apg := data.Data.Profiles[0]

	return &apg, nil
}
