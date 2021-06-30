#!/bin/sh

case "$1" in
    remove)
        systemctl disable cambium-exporter || true
        systemctl stop cambium-exporter    || true
    ;;
esac
