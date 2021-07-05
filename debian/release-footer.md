### Which file to download and install?

- Windows: `cambium-exporter_*_windows_amd64.tar.gz`

- Debian:
  - either `cambium-exporter_*_Debian_amd64.deb`, which depends on and
    auto-installs Chromium,
  - or `cambium-exporter_*_NoChromium_amd64.deb`, which does
    not.<sup>1</sup>

- Ubuntu: `cambium-exporter_*_NoChromium_amd64.deb`<sup>1, 2</sup>

- Other Linux variant: `cambium-exporter_*_linux_amd64.tar.gz`

Notes:

- <sup>1</sup>: you still need to provide your own Chromium or Google
  Chrome installation
- <sup>2</sup>: the Ubuntu `chromium-browser` is actually
  a Snap package, which **does not work with the systemd unit**; you
  need to install a "real" Chromium/Google Chrome via other means, or
  switch to Docker.
