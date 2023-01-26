FROM debian:buster-slim

RUN mkdir -p /opt/api/logs && \
    apt update && apt install -y openssl && \
    apt clean && rm -rf /var/lib/apt/lists/*

COPY build/bin/vxapi                   /opt/api/bin/
COPY db/api/migrations                 /opt/api/db/api/migrations
COPY build/package/api/templates       /opt/api/templates
COPY build/ssl                         /opt/api/ssl
COPY assets/web                        /opt/api/static
COPY security/certs/api                /opt/api/security/certs/api
COPY build/package/api/entrypoint.sh   /opt/api/bin/

RUN chmod +x /opt/api/bin/entrypoint.sh

WORKDIR /opt/api

ENTRYPOINT ["/opt/api/bin/entrypoint.sh", "/opt/api/bin/vxapi"]
