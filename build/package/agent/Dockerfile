FROM debian:buster-slim

RUN mkdir -p /opt/vxagent/{bin,data,logs}

COPY preparing.sh 			/opt/vxagent/bin/
COPY build/package/agent/_tmp/version 			/opt/vxagent/bin/
COPY build/package/agent/_tmp/linux/amd64/vxagent 	/opt/vxagent/bin/

WORKDIR /opt/vxagent

RUN chmod +x /opt/vxagent/bin/preparing.sh && \
    /opt/vxagent/bin/preparing.sh

ENTRYPOINT ["/opt/vxagent/bin/vxagent"]
