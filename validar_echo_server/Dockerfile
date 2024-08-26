FROM ubuntu:latest

RUN apt-get update && \
apt-get install -y netcat-traditional

WORKDIR /scripts

COPY send-message-to-server.sh /scripts
RUN chmod +x /scripts/send-message-to-server.sh

ENTRYPOINT ["/scripts/send-message-to-server.sh"]