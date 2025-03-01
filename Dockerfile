FROM alpine:latest

WORKDIR /app/

# rclone params for cleanOldTags
ENV RCLONE_CONFIG_S3_TYPE=s3
ENV RCLONE_CONFIG_S3_PROVIDER=AWS
ENV RCLONE_CONFIG_S3_ACCESS_KEY_ID=change-it
ENV RCLONE_CONFIG_S3_SECRET_ACCESS_KEY=change-it
ENV RCLONE_CONFIG_S3_REGION=eu-central-1

COPY --from=minio/mc:latest /usr/bin/mc /usr/local/bin
COPY --from=registry:latest /bin/registry /usr/local/bin
COPY --from=registry:latest /etc/docker/registry/config.yml /etc/docker/registry/config.yml

RUN apk upgrade \
&& apk add rclone \
&& addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app \
&& chown -R 101:101 /app \
&& rm -rf /tmp/*

COPY --chown=101:101 ./gitlab-registry-cleaner /app/gitlab-registry-cleaner

USER 101

ENTRYPOINT [ "/app/gitlab-registry-cleaner" ]
