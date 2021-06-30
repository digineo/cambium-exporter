#!/bin/sh

groupadd --system cambium-exporter || true
useradd --system -d /nonexistent -s /usr/sbin/nologin -g cambium-exporter cambium-exporter || true

chown cambium-exporter /etc/cambium-exporter/*.toml

systemctl daemon-reload
systemctl enable cambium-exporter
systemctl restart cambium-exporter
