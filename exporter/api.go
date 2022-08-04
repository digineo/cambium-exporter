package exporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0"

type responseBuffer struct {
	bytes.Buffer
}

var bpool = &sync.Pool{
	New: func() interface{} {
		return new(responseBuffer)
	},
}

func (rb *responseBuffer) Close() error {
	rb.Buffer.Reset()
	bpool.Put(rb)
	return nil
}

// newRequest creates a new HTTP request and prefills its header.
func (c *Client) fetch(ctx context.Context, method, path string, params url.Values) (*http.Response, error) {
	u2 := *c.instance // dup
	u2.Path = "/0/cn-srv"
	if len(path) > 0 && path[0] != '/' {
		u2.Path += "/"
	}
	u2.Path += path
	u2.RawQuery = params.Encode()
	url := u2.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	t0 := time.Now()
	res, err := c.client.Do(req)
	if err == nil {
		buf, _ := bpool.Get().(*responseBuffer)
		_, _ = io.Copy(buf, res.Body)
		res.Body.Close()
		res.Body = buf

		c.log.Debugf("fetch %s (status %d) %d bytes in %v",
			url,
			res.StatusCode,
			buf.Len(),
			time.Since(t0),
		)
	} else {
		c.log.Infof("error fetching %s: %v", url, err)
	}
	return res, err
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
	var data struct {
		Data struct {
			Profiles []APGroupAPIResponse `json:"profiles"`
		} `json:"data"`
	}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode device response data for AP group %q: %w", apGroup, err)
	}
	if len(data.Data.Profiles) == 0 {
		return nil, nil
	}

	apg := data.Data.Profiles[0]
	return &apg, nil
}

func (c *Client) fetchGuestPortals(ctx context.Context) ([]string, error) {
	res, err := c.fetch(ctx, http.MethodGet, "/services/guest/portal", url.Values{
		"limit":  {"10"},
		"offset": {"0"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guest portal list: %w", err)
	}

	defer res.Body.Close()
	var data struct {
		Data struct {
			Portals []PortalAPIResponse `json:"result"`
		} `json:"data"`
	}
	if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode guest portal response: %w", err)
	}
	if len(data.Data.Portals) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(data.Data.Portals))
	for _, portal := range data.Data.Portals {
		names = append(names, portal.Name)
	}
	return names, nil
}

const sessionsPerPage = 200

func (c *Client) fetchPortalSessions(ctx context.Context, name string) ([]*PortalSession, int, error) {
	count := make(map[string]int) // key = AP MAC address
	total := 0

	for page := 0; ; page++ {
		data, err := c.fetchPortalSessionsPage(ctx, name, page)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decode portal session data for portal %s (page %d): %w", name, page, err)
		}
		for _, s := range data.Sessions {
			count[s.DeviceMAC]++
		}
		if total = data.Meta.Total; total < sessionsPerPage*(page+1) {
			break // no more results on the next page
		}
	}

	sessions := make([]*PortalSession, 0, len(count))
	for mac, n := range count {
		sessions = append(sessions, &PortalSession{
			DeviceMAC:  mac,
			Sessions:   n,
			PortalName: name,
		})
	}
	return sessions, total, nil
}

func (c *Client) fetchPortalSessionsPage(ctx context.Context, name string, page int) (*SessionsAPIResponse, error) {
	res, err := c.fetch(ctx, http.MethodGet, "/services/guest/session/"+name, url.Values{
		"limit":  {strconv.Itoa(sessionsPerPage)},
		"offset": {strconv.Itoa(sessionsPerPage * page)},
	})
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	var data struct {
		Data SessionsAPIResponse `json:"data"`
	}

	err = json.NewDecoder(res.Body).Decode(&data)
	return &data.Data, err
}
