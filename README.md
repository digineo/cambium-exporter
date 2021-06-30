# Cambium cnMaestro Cloud Exporter

This is a [Prometheus](https://prometheus.io/) exporter for the
[Cambium cnMaestro Cloud controller](https://cloud.cambiumnetworks.com).

## Installation

Use one oth the pre-built [binary releases](https://github.com/digineo/cambium-exporter),
or compile and install it using the Go toolchain:

```console
$ go install github.com/digineo/cambium-exporter@latest
```

## Configuration

### Exporter

You need a `config.toml` with the following content:

```toml
Instance = "https://<your-instance>.cloud.cambiumnetworks.com/"
SessionID = "<sid cookie>"
```

The `Instance` is the URL to your cloud controller. Authentication (for
this exporter) is a bit cumbersome, since the cloud controller does not
provide a publicy accessible API.

<details><summary>Obtaining the session cookie</summary>

You need to undertake the following steps to obtain the session cookie
(assuming you already have a login to the controller, and permissions to
create new users):

0. Log into your controller instance.
1. Under "Administration" → "Users", click on "Add User", and fill in an
   email address and select "Monitor" from "Role". Then click on "Send"
   to send an invitation to that email address.
2. When receiving the invitation, open the link in a new private browser
   window. You need to register a new, dedicated account for the email
   address entered above.
3. Upon completing the registration, you should see an outstanding
   invitation in your account dashboard. Click on "accept". You will be
   redirected to the cloud controller dashboard.
4. Open the Browser developer tools (<kbd>Ctrl+Shift+I</kbd> or
   <kbd>F12</kbd> on Firefox), and you will find the `sid` cookie in the
   "Storage" tab. Copy only its value into the `config.toml` file.
5. Close the private window - DO NOT LOGOUT (this invalidates the session
   cookie).

If you find the exporter to stop working, you probably need to refresh
the `SessionID` configuration. To do so, open a private browser window,
and log into your controller instance with the Monitor account. Then
repeat the last two steps.

</details>

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
