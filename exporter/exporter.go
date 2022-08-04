package exporter

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (c *Client) Start(listenAddress, version string) error {
	if err := c.login(); err != nil {
		return err
	}
	go c.startSessionRefresh()

	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		apGroups, err := c.fetchAPGroups(r.Context())
		if err != nil {
			log.Printf("fetching AP groups failed: %v", err)
		}

		portals, err := c.fetchGuestPortals(r.Context())
		if err != nil {
			log.Printf("fetching guest portals failed: %v", err)
		}

		err = tmpl.Execute(w, &indexVariables{
			Instance: c.instance.String(),
			Groups:   apGroups,
			Portals:  portals,
			Version:  version,
		})
		if err != nil {
			log.Printf("rendering index template failed: %v", err)
		}
	})

	router.GET("/apgroups", c.listAPGroups)
	router.GET("/apgroups/:ap_group/debug", c.apGroupDebugHandler)
	router.GET("/apgroups/:ap_group/metrics", c.apGroupMetricsHandler)

	router.GET("/portals", c.listPortals)
	router.GET("/portals/:portal_name/debug", c.portalDebugHandler)
	router.GET("/portals/:portal_name/metrics", c.portalMetricsHandler)

	var where string
	if host, port, err := net.SplitHostPort(listenAddress); err == nil && host == "" {
		where = fmt.Sprintf("http://0.0.0.0:%s/", port)
	} else {
		where = fmt.Sprintf("http://%s/", listenAddress)
	}

	c.log.Infof("Starting exporter on %s", where)
	return http.ListenAndServe(listenAddress, router)
}

func (c *Client) listAPGroups(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.Body.Close()
	result, err := c.fetchAPGroups(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&result)
}

func (c *Client) apGroupMetricsHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	apg := params.ByName("ap_group")

	reg := prometheus.NewRegistry()
	reg.MustRegister(&Collector{
		client:  c,
		apGroup: apg,
		ctx:     r.Context(),
	})

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func (c *Client) apGroupDebugHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	apg := params.ByName("ap_group")

	devices, err := c.fetchDevices(r.Context(), apg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	basic, err := c.fetchAPGroupData(r.Context(), apg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"apgroup": basic,
		"devices": devices,
	})
}

func (c *Client) listPortals(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.Body.Close()
	result, err := c.fetchGuestPortals(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&result)
}

func (c *Client) portalMetricsHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	name := params.ByName("portal_name")

	reg := prometheus.NewRegistry()
	reg.MustRegister(&PortalCollector{
		client: c,
		portal: name,
		ctx:    r.Context(),
	})

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func (c *Client) portalDebugHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	name := params.ByName("portal_name")

	sessions, total, err := c.fetchPortalSessions(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"total":    total,
		"sessions": sessions,
	})
}

type indexVariables struct {
	Instance string
	Groups   []string
	Portals  []string
	Version  string
}

//go:embed exporter.html
var tmplData string

var tmpl = template.Must(template.New("index").Option("missingkey=error").Parse(tmplData))
