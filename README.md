# Cambium cnMaestro Cloud Exporter

This is a [Prometheus](https://prometheus.io/) exporter for the
[Cambium cnMaestro Cloud controller](https://cloud.cambiumnetworks.com).

Currently, it exports the following metrics:

- AP group status:
  - number of devices (APs), config sync status, client count
  - per AP:
    - up- and downtime
    - per radio:
      - channel, channel width, power, quality, transfer rate
- Guest access portals:
  - number of active sessions per portal and per AP

(This list might be outdated. The authoritative list is defined in the
sources code, see [`collector.go`](./exporter/collector.go)).

## Installation

### Pre-built binary releases

Head to the [release page](https://github.com/digineo/cambium-exporter/releases)
for downloads.

There, you'll find packages for Debian/Ubuntu (for systemd), and archives containing
a pre-built binary for other Linux distributions and Windows.

Please note that you;re going to need a relatively recent version of either Chromium
or Google Chrome installed on the machine where you want to run the exporter.

On Debian, this can be achieved through an `apt install chromium`.

On Ubuntu, the version installed through `apt install chromium-browser` is actually
a Snap packege, which does not work properly with the way  Please install a
"real" package instead.

<details><summary>Why do I need a browser? (click to expand)</summary>

There's currently no officially supported API for cnMaestro cloud instances.

To get the metrics, the exporter can however simply issue HTTP requests to their
internal API, which is protected by a session cookie and CSRF token.

To get the session cookie and CSRF token, we sadly cannot issue plain HTTP
requests to their SSO, because they do some of JS crypto shenanigans to mangle
and obfuscate their login procedure.

We could reverse-engineer that, but that'll always only be a temporary solution.
A much simpler workaround is to leverage a remote-controlled browser to do the
login dance. After all, their web UI is made for browsers, isn't it?

The exporter uses Chromium/Google Chrome only to retrieve the session cookie,
and will shutdown the browser afterwards. Every so often (pre-emptively before
the cookie expires), the exporter will try to refresh the session cookie and
login once more, requiring another browser instance.

This all happens automatically, and without the need for user-interaction,
hence the browser starts without UI ("headless").

If you want to see what the browser does, you try this:

```console
$ HEADLESS=0 cambium-exporter --login --verbose --config ./config.toml
```

</details>

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
and restart the exporter by running `systemctl restart cambium-exporter`.
Modify the start parameters in `/etc/defaults/cambium-exporter` if you want
the controller to bind on other addresses than localhost.

After starting the controller, just visit http://localhost:9836/.
You will see a list of all configured WiFi AP groups and links to the
corresponding metrics endpoints.

### Prometheus

Add a scrape config to your Prometheus configuration and reload Prometheus.

This template sets up Prometheus to scrape two AP groups ("Default" and
"another-group") and one guest access portal ("VisitorPortal"). Omit one
or the other, if you don't use the portal feature, for example.

In general, you should only need to fill in the `static_configs.targets`
array:

```yml
scrape_configs:
  - job_name: cambium_aps
    static_configs:
      - targets:
        # Select the AP groups you care about.
        # Visit http://localhost:9836 to view a list of available AP groups.
        - Default
        - another-group
    relabel_configs:
      - # convert "NAME" to "/apgroups/NAME/metrics"
        source_labels: [__address__]
        regex:         (.+)
        target_label:  __metrics_path__
        replacement:   /apgroups/$1/metrics
      - # override "instance" label with address, i.e. the target name
        source_labels: [__address__]
        target_label: instance
      - # overrride address (point to the exporter's real hostname:port)
        target_label: __address__
        replacement: 127.0.0.1:9836

  - job_name: cambium_portals
    static_configs:
      - targets:
        # Select the guest access portals you care about.
        # Visit http://localhost:9836 to view a list of available portals.
        - VisitorPortal
    relabel_configs: # similar to the above
      - source_labels: [__address__]
        regex:         (.+)
        target_label:  __metrics_path__
        replacement:   /portals/$1/metrics
      - source_labels: [__address__]
        target_label: instance
      - target_label: __address__
        replacement: 127.0.0.1:9836
```

<details><summary>Using a combined jobs for AP groups and guest portals</summary>

You can also combine the scrape config above. In this case, the naming of
`static_configs.targets` changes, and you need to prefix either `apgroups/`
or `portals/` to the target name.

```yaml
scrape_configs:
  - job_name: cambium
    static_configs:
      - targets:
        # Select the AP groups and portals you care about.
        # Visit http://localhost:9836 to view a list of available entries.
        - apgroups/Default
        - apgroups/another-group
        - portals/VisitorPortal
    relabel_configs:
      - # convert "TYPE/NAME" to "/TYPE/NAME/metrics"
        source_labels: [__address__]
        regex:         (.+)
        target_label:  __metrics_path__
        replacement:   /$1/metrics
      - # override "instance" label with address, i.e. the target name
        source_labels: [__address__]
        target_label: instance
      - # overrride address (point to the exporter's real hostname:port)
        target_label: __address__
        replacement: 127.0.0.1:9836
```

</details>

## License

This exporter is available as open soure under the terms of the
[MIT License](https://opensource.org/licenses/MIT).

For more details, see the [LICENSE](./LICENSE) file.

## Notice

"Cambium", "cnMaestro" and "cnMaestro Cloud" are trademarks of Cambium
Networks, Ltd., <https://www.cambiumnetworks.com/>.
