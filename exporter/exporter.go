package exporter

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (c *Client) Start(listenAddress, version string) {
	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		tmpl.Execute(w, &indexVariables{
			Instance: c.instance.String(),
			Version:  version,
		})
	})

	router.GET("/apgroups", c.listAPGroups)
	router.GET("/apgroups/:ap_group/debug", c.debugHandler)
	// router.GET("/apgroups/:ap_group/metrics", cfg.debugHandler)

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
			<p>List of WiFi AP Group names:</p>
			<pre><a href="/apgroups">/apgroups</a></pre>
		</li>
		<li>
			<p>Metrics></p>
			<pre>/apgroups/<var>⟨ap-group-name⟩</var>/metrics</pre>
			<p>(replace <var>⟨ap-group-name⟩</var> with the name of one of your WiFi AP Groups)</p>
		</li>
	</ul>

</body>
</html>
`))
