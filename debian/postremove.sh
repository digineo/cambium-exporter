#!/bin/sh

case "$1" in
    remove)
        systemctl daemon-reload
        userdel  cambium-exporter || true
        groupdel cambium-exporter 2>/dev/null || true
    ;;
esac
