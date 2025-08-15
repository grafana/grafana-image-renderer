FROM alpine:edge@sha256:115729ec5cb049ba6359c3ab005ac742012d92bbaa5b8bc1a878f1e8f62c0cb8 AS builder

WORKDIR /src
RUN apk add --no-cache openssl

COPY ./rootCA.pem ca.pem
RUN openssl x509 -inform PEM -in ca.pem -out ca.crt

FROM grafana/grafana-enterprise:latest@sha256:1645b1867318ef7697c86c1f5e43a3c6c410124307486c1d6e8b77ea7ee9e563

COPY --from=builder /src/ca.crt /usr/local/share/ca-certificates/ca.crt
COPY --chown=grafana ./grafana.localhost+1.pem /grafana.localhost.pem
COPY --chown=grafana ./grafana.localhost+1-key.pem /grafana.localhost-key.pem

USER root
RUN update-ca-certificates --fresh
USER grafana

ENV GF_SERVER_PROTOCOL=https
ENV GF_SERVER_CERT_FILE=/grafana.localhost.pem
ENV GF_SERVER_CERT_KEY=/grafana.localhost-key.pem
