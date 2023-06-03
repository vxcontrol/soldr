FROM debian:buster-slim

RUN apt update && apt install -y libssl1.1 && \
    apt clean && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt/vxagent/bin && \
    mkdir -p /opt/vxagent/data && \
    mkdir -p /opt/vxagent/logs && \
    mkdir -p /opt/vxagent/deps

COPY build/package/agent/preparing.sh                      /opt/vxagent/bin/
COPY build/package/agent/_tmp/version                      /opt/vxagent/bin/
COPY build/package/agent/_tmp/linux/amd64/vxbundle         /opt/vxagent/bin/vxagent
COPY build/package/agent/_tmp/deps/libraries_amd64.tar.gz  /opt/vxagent/deps/

RUN mkdir -p /usr/lib/vxagent && \
    tar -xzf /opt/vxagent/deps/libraries_amd64.tar.gz -C /usr/lib/vxagent/ && \
    rm -rf /opt/vxagent/deps

WORKDIR /opt/vxagent/data

RUN chmod +x /opt/vxagent/bin/preparing.sh && \
    /opt/vxagent/bin/preparing.sh

ENTRYPOINT ["/opt/vxagent/bin/vxagent"]
