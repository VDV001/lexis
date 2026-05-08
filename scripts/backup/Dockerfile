FROM postgres:18-alpine

RUN apk add --no-cache \
        age \
        aws-cli \
        bash \
        ca-certificates \
        tzdata

WORKDIR /backup
COPY backup.sh /backup/backup.sh
RUN chmod +x /backup/backup.sh

ENTRYPOINT []
CMD ["/backup/backup.sh"]
