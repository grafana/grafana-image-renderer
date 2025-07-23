FROM alpine:edge AS builder

WORKDIR /src
RUN apk add --no-cache openssl

COPY ./rootCA.pem ca.pem
RUN openssl x509 -inform PEM -in ca.pem -out ca.crt

FROM grafana/grafana-enterprise:latest

COPY --from=builder /src/ca.crt /usr/local/share/ca-certificates/ca.crt
COPY --chown=grafana ./grafana.localhost+1.pem /grafana.localhost.pem
COPY --chown=grafana ./grafana.localhost+1-key.pem /grafana.localhost-key.pem

USER root
RUN update-ca-certificates --fresh
USER grafana

ENV GF_SERVER_PROTOCOL=https
ENV GF_SERVER_CERT_FILE=/grafana.localhost.pem
ENV GF_SERVER_CERT_KEY=/grafana.localhost-key.pem
