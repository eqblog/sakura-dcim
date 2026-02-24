FROM postgres:16-alpine

RUN apk add --no-cache bash dcron

COPY backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

# Default: daily at 2 AM
RUN echo "0 2 * * * /usr/local/bin/backup.sh >> /var/log/backup.log 2>&1" > /etc/crontabs/root

VOLUME ["/backups"]

CMD ["crond", "-f", "-l", "2"]
