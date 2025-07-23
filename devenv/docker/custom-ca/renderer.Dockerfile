FROM mariell/test

USER root
COPY ./rootCA.pem rootCA.pem
RUN mkdir -p /usr/share/ca-certificates/
RUN openssl x509 -inform PEM -in rootCA.pem -out /usr/share/ca-certificates/rootCA.crt
RUN update-ca-certificates --fresh

USER nonroot
