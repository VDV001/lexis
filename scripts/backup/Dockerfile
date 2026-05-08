FROM postgres:18-alpine

RUN apk add --no-cache \
        age \
        aws-cli \
        bash \
        ca-certificates \
        coreutils \
        tzdata

WORKDIR /backup
COPY backup.sh retention.sh /backup/
RUN chmod +x /backup/backup.sh /backup/retention.sh

ENTRYPOINT []
CMD ["/backup/backup.sh"]
