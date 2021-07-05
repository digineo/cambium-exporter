# Cambium cnMaestro Cloud Exporter

This is a [Prometheus](https://prometheus.io/) exporter for the
[Cambium cnMaestro Cloud controller](https://cloud.cambiumnetworks.com).

## Installation

### Pre-built binary releases

Head to the [release page](https://github.com/digineo/cambium-exporter)
for download.

### Docker

Alternatively, you can use Docker:

```console
$ docker run -v /path/to/config.toml:config.toml:ro -p 9836:9836 digineode/cambium-exporter
```

### Build it yourself

Last but not least, you can compile and install the exporter using the
Go toolchain:

```console
$ go install github.com/digineo/cambium-exporter@latest
```

## Configuration

### Exporter

You need a `config.toml` with the following content:

```toml
Username = "<login email address>"
Password = "<login password>"
Instance = "https://<your instance>.cloud.cambiumnetworks.com/"
```

It is **strongly recommended**, that you create a separate user for the
exporter (with role "Monitor").

If you use the Debian package, just edit `/etc/cambium-exporter/config.toml`
and restart the exporter by running `systemctl restart `cambium-exporter`.
Modify the start parameters in `/etc/defaults/cambium-exporter` if you want
the controller to bind on other addresses than localhost.

After starting the controller, just visit http://localhost:9836/.
You will see a list of all configured WiFi AP groups and links to the
corresponding metrics endpoints.

### Prometheus

Add a scrape config to your Prometheus configuration and reload Prometheus.

```yml
scrape_configs:
  - job_name: cambium
    relabel_configs:
      - source_labels: [__address__]
        regex:         (.+)
        target_label:  __metrics_path__
        replacement:   /apgroups/$1/metrics
      - source_labels: [__address__]
        target_label: instance
      - target_label: __address__
        replacement: 127.0.0.1:9836 # The exporter's real hostname:port
    static_configs:
      - targets: # select the AP groups you care about
        - Default
        - another-group
```

## License

This exporter is available as open soure under the terms of the
[MIT License](https://opensource.org/licenses/MIT).

For more details, see the [LICENSE](./LICENSE) file.

## Notice

"Cambium", "cnMaestro" and "cnMaestro Cloud" are trademarks of Cambium
Networks, Ltd., <https://www.cambiumnetworks.com/>.
