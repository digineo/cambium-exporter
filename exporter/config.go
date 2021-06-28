package exporter

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"github.com/pelletier/go-toml"
)

type Config struct {
	Username  string // currently not used
	Password  string // currently not used
	SessionID string // used instead of username/password
	Instance  string
	APGroups  []string // list of WiFi AP Group names to export

	instance *url.URL
	client   *http.Client
}

// LoadConfig loads the configuration from a file.
func LoadConfig(file string) (*Config, error) {
	cfg := Config{}
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

func (cfg *Config) getCsrfToken() string {
	for _, c := range cfg.client.Jar.Cookies(cfg.instance) {
		if c.Name == "XSRF-TOKEN" {
			return c.Value
		}
	}

	return ""
}

const userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0"

func (cfg *Config) FetchCSRFToken() {
	var u2 url.URL
	u2 = *cfg.instance // dup
	u2.Path = "/0/cn-srv/user/me"

	req, err := http.NewRequest(http.MethodGet, u2.String(), nil)
	if err != nil {
		panic(err)
	}

	// for _, c := range cfg.client.Jar.Cookies(cfg.instance) {
	// 	fmt.Printf("add-cookie: %v\n", c)
	// 	req.AddCookie(c)
	// }

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	// bypass canonicalization
	req.Header["x-cidx"] = []string{"0"}
	if token := cfg.getCsrfToken(); token != "" {
		req.Header["X-XSRF-TOKEN"] = []string{token}
	}

	fmt.Println("token:", cfg.getCsrfToken())
	res, err := cfg.client.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
	var buf bytes.Buffer
	io.Copy(&buf, res.Body)

	fmt.Println("response:", res.StatusCode, buf.String())
	if token := cfg.getCsrfToken(); token != "" {
		fmt.Println("token:", token)
	}
}
