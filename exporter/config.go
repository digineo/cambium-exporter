package exporter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	"github.com/digineo/cambium-exporter/auth"
	"github.com/pelletier/go-toml"
)

type Client struct {
	Username string
	Password string
	Instance string

	instance *url.URL
	client   *http.Client
	authInfo *auth.AuthInfo
}

const (
	loginTimeout                = 5 * time.Minute  // max time for a login attempt
	sessionRefreshInterval      = 6 * time.Hour    // how often to refresh session cookie
	sessionRefreshRetries       = 24               // number of retries, if session refresh failed (24*30min = 12h)
	sessionRefershRetryInterval = 30 * time.Minute // interval between failed sesion refresh attempts
)

// LoadClientConfig loads the configuration from a file and initializes
// the client.
func LoadClientConfig(file string) (*Client, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %w", file, err)
	}
	defer f.Close()

	c := Client{}
	if err := toml.NewDecoder(f).Strict(true).Decode(&c); err != nil {
		return nil, fmt.Errorf("loading config file %q failed: %w", file, err)
	}

	uri, err := url.Parse(c.Instance)
	if err != nil {
		return nil, fmt.Errorf("invalid instance url: %w", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("invalid cookies: %w", err)
	}

	c.instance = uri
	c.client = &http.Client{Jar: jar}

	return &c, nil
}

func (c *Client) login() error {
	ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	defer cancel()

	info, err := auth.Login(ctx, c.Instance, c.Username, c.Password)
	if err != nil {
		return err
	}

	xsrfCookie := &http.Cookie{Name: "XSRF-TOKEN"}
	if info.XSRFToken == "" {
		xsrfCookie.MaxAge = -1
	} else {
		xsrfCookie.Value = info.XSRFToken
	}

	sidCookie := &http.Cookie{
		Name:  "sid",
		Value: info.SessionID,
	}

	c.client.Jar.SetCookies(c.instance, []*http.Cookie{sidCookie, xsrfCookie})

	return nil
}

func (c *Client) getCsrfToken() string {
	for _, cookie := range c.client.Jar.Cookies(c.instance) {
		if cookie.Name == "XSRF-TOKEN" {
			return cookie.Value
		}
	}

	return ""
}

func (c *Client) startSessionRefresh() {
	t := time.NewTicker(sessionRefreshInterval)
	failures := 0

	for range t.C {
		if err := c.login(); err != nil {
			log.Printf("session refresh failed: %v", err)
			failures++
			if failures > sessionRefreshRetries {
				log.Fatal("could not refresh session for 12+ hours, aborting")
			}

			t.Reset(sessionRefershRetryInterval)
		} else {
			t.Reset(sessionRefreshInterval)
		}
	}
}
