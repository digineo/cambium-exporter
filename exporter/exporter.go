package exporter

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (c *Client) Start(listenAddress, version string) {
	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		apGroups, err := c.fetchAPGroups(r.Context())
		if err != nil {
			log.Printf("fetching AP groups failed: %v", err)
		}

		err = tmpl.Execute(w, &indexVariables{
			Instance: c.instance.String(),
			Groups:   apGroups,
			Version:  version,
		})
		if err != nil {
			log.Printf("rendering index template failed: %v", err)
		}
	})

	router.GET("/apgroups", c.listAPGroups)
	router.GET("/apgroups/:ap_group/debug", c.debugHandler)
	router.GET("/apgroups/:ap_group/metrics", c.metricsHandler)

	log.Printf("Starting exporter on http://%s/", listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, router))
}

func (c *Client) listAPGroups(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.Body.Close()
	result, err := c.fetchAPGroups(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)

		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&result)
}

func (c *Client) metricsHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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

func (c *Client) debugHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"apgroup": basic,
		"devices": devices,
	})
}

type indexVariables struct {
	Instance string
	Groups   []string
	Version  string
}

var tmpl = template.Must(template.New("index").Option("missingkey=error").Parse(`<!doctype html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Cambium cnMaestro Cloud Exporter (Version {{.Version}})</title>
</head>
<body>
	<h1>Cambium cnMaestro Cloud Exporter</h1>
	<p>Version: {{.Version}}</p>
	<p><a href="{{ .Instance }}" target="_blank">Open controller in new tab.</a></p>

	<h2>Endpoints</h2>
	<ul>
		<li>
			<a href="/apgroups">List of WiFi AP Group names</a> (JSON)
		</li>
		<li>
			<strong>Metrics</strong>
			<ul>{{ range .Groups }}
				<li>
					<a href="/apgroups/{{ . }}/metrics">{{ . }}</a> &bull;
					<a href="/apgroups/{{ . }}/debug">debug data</a> (JSON)
				</li>
			{{ end }}</ul>
		</li>
	</ul>
</body>
</html>
`))
