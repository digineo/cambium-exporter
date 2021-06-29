package exporter

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"github.com/pelletier/go-toml"
)

type Client struct {
	Username  string // currently not used
	Password  string // currently not used
	SessionID string // used instead of username/password
	Instance  string
	APGroups  []string // list of WiFi AP Group names to export

	instance *url.URL
	client   *http.Client
}

// LoadClientConfig loads the configuration from a file and initializes
// the client.
func LoadClientConfig(file string) (*Client, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %w", file, err)
	}
	defer f.Close()

	cfg := Client{}
	if err := toml.NewDecoder(f).Strict(true).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("loading config file %q failed: %w", file, err)
	}

	uri, err := url.Parse(cfg.Instance)
	if err != nil {
		return nil, fmt.Errorf("invalid instance url: %w", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("invalid cookies: %w", err)
	}
	jar.SetCookies(uri, []*http.Cookie{{
		Name:  "sid",
		Value: cfg.SessionID,
	}})

	cfg.instance = uri
	cfg.client = &http.Client{Jar: jar}

	return &cfg, nil
}

func (c *Client) getCsrfToken() string {
	for _, cookie := range c.client.Jar.Cookies(c.instance) {
		if cookie.Name == "XSRF-TOKEN" {
			return cookie.Value
		}
	}

	return ""
}
