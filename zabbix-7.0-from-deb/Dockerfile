FROM ubuntu:22.04

LABEL maintainer="mv"
ENV DEBIAN_FRONTEND=noninteractive

COPY init-schema.sh /usr/local/bin/init-schema.sh
COPY sql/ /sql/
RUN chmod +x /usr/local/bin/init-schema.sh

RUN echo 'Acquire::AllowInsecureRepositories "true";' > /etc/apt/apt.conf.d/99insecure && \
    apt-get update --allow-unauthenticated && \
    apt-get install -y --no-install-recommends ca-certificates

RUN sed -i 's|http://|https://|g' /etc/apt/sources.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        libsnmp-dev libevent-dev libpcre2-dev libcurl4-openssl-dev fping \
        mysql-client gnupg lsb-release wget net-tools iputils-ping vim nano systemd \
        libodbc2 libopenipmi0 libxml2 gettext-base && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY zabbix-server-mysql_7.0.0~alpha9-4+ubuntu22.04_amd64.deb /tmp/
RUN dpkg -i /tmp/zabbix-server-mysql_7.0.0~alpha9-4+ubuntu22.04_amd64.deb || apt-get install -f -y

RUN mkdir -p /run/zabbix /var/log/zabbix /var/lib/zabbix /etc/zabbix && \
    chown -R zabbix:zabbix /run/zabbix /var/log/zabbix /var/lib/zabbix

EXPOSE 10051

ENTRYPOINT ["/usr/local/bin/init-schema.sh"]
