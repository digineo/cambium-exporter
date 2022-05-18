FROM debian:bullseye

RUN set -ex \
 && apt-get update \
 && apt-get upgrade --yes \
 && apt-get install --yes --no-install-recommends \
      ca-certificates \
      chromium \
 && rm -rf /var/lib/apt/lists/*

COPY dist/cambium-exporter_linux_amd64/cambium-exporter /usr/bin

EXPOSE 9836

CMD ["/usr/bin/cambium-exporter", "--web.listen-address=:9836", "--config=/config.toml"]
