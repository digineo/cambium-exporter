### Which file to download and install?

- Windows:
  - `cambium-exporter_*_windows_amd64.tar.gz`
- Debian and derivates (Ubuntu):
  - `cambium-exporter_*_amd64.deb` (requires systemd)
- Other Linux variant:
  - `cambium-exporter_*_linux_amd64.tar.gz`

Notes:

These packages don't ship with a hard-coded depencency to Chromium or Google Chrome, however, one of these is required for the operation.

On Ubuntu, the `chromium-browser` package is actually a Snap package, which **does not work with the systemd unit**. You need to install a "real" Chromium/Google Chrome via other means, or switch to Docker.
