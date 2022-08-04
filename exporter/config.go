package exporter

import (
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
	log      logger
}

const (
	sessionRefreshInterval      = 6 * time.Hour    // how often to refresh session cookie
	sessionRefreshRetries       = 24               // number of retries, if session refresh failed (24*30min = 12h)
	sessionRefershRetryInterval = 30 * time.Minute // interval between failed sesion refresh attempts
)

// LoadClientConfig loads the configuration from a file and initializes
// the client.
func LoadClientConfig(file string, verbose bool) (*Client, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %w", file, err)
	}
	defer f.Close()

	c := Client{log: logger(verbose)}
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
	c.log.Infof("performing login")

	info, err := auth.Login(c.Username, c.Password)
	if err != nil {
		c.log.Errorf("login failed: %v", err)
		return err
	}

	xsrfCookie := &http.Cookie{Name: "XSRF-TOKEN"}
	if info.XSRFToken == "" {
		xsrfCookie.MaxAge = -1
	} else {
		c.log.Debugf("login: got xsrf token")
		xsrfCookie.Value = info.XSRFToken
	}

	c.log.Debugf("login: got session token")
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
			c.log.Errorf("session refresh failed: %v", err)
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
