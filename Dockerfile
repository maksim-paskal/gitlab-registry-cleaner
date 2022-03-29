FROM alpine:latest

ADD https://downloads.rclone.org/v1.51.0/rclone-v1.51.0-linux-amd64.zip /tmp/rclone.zip

WORKDIR /app/

# rclone params for cleanOldTags
ENV RCLONE_CONFIG_S3_TYPE=s3
ENV RCLONE_CONFIG_S3_PROVIDER=AWS
ENV RCLONE_CONFIG_S3_ACCESS_KEY_ID=change-it
ENV RCLONE_CONFIG_S3_SECRET_ACCESS_KEY=change-it
ENV RCLONE_CONFIG_S3_REGION=eu-central-1

COPY --from=registry:2.7.1 /bin/registry /usr/local/bin
COPY --from=registry:2.7.1 /etc/docker/registry/config.yml /etc/docker/registry/config.yml

RUN apk upgrade \
&& cd /tmp \
&& unzip rclone.zip \
&& mv rclone-v1.51.0-linux-amd64/rclone /usr/local/bin/rclone \
&& touch /tmp/checksum \
&& echo "410f8e70f9e11b1abf55c13335bdeb36edfce9a97c7bbe8c91fa5de6f22f6031  /usr/local/bin/rclone" >> /tmp/checksum \
&& echo "d494c104bc9aa4b39dd473f086dbe0a5bdf370f1cb4a7b9bb2bd38b5e58bb106  /usr/local/bin/registry" >> /tmp/checksum \
&& cat /tmp/checksum \
&& sha256sum -c /tmp/checksum \
&& rm /tmp/checksum \
&& addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app \
&& chown -R 101:101 /app \
&& rm -rf /tmp/*

COPY --chown=101:101 ./gitlab-registry-cleaner /app/gitlab-registry-cleaner

USER 101

ENTRYPOINT [ "/app/gitlab-registry-cleaner" ]
