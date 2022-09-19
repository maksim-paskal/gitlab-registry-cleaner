FROM alpine:latest
ARG TARGETARCH

ADD https://downloads.rclone.org/v1.51.0/rclone-v1.51.0-linux-${TARGETARCH}.zip /tmp/rclone.zip

WORKDIR /app/

# rclone params for cleanOldTags
ENV RCLONE_CONFIG_S3_TYPE=s3
ENV RCLONE_CONFIG_S3_PROVIDER=AWS
ENV RCLONE_CONFIG_S3_ACCESS_KEY_ID=change-it
ENV RCLONE_CONFIG_S3_SECRET_ACCESS_KEY=change-it
ENV RCLONE_CONFIG_S3_REGION=eu-central-1

COPY --from=registry:2.8.1 /bin/registry /usr/local/bin
COPY --from=registry:2.8.1 /etc/docker/registry/config.yml /etc/docker/registry/config.yml

RUN apk upgrade \
&& cd /tmp \
&& unzip rclone.zip \
&& mv rclone-v1.51.0-linux-${TARGETARCH}/rclone /usr/local/bin/rclone \
&& touch /tmp/checksum.amd64 \
&& echo "410f8e70f9e11b1abf55c13335bdeb36edfce9a97c7bbe8c91fa5de6f22f6031  /usr/local/bin/rclone" >> /tmp/checksum.amd64 \
&& echo "c8193513993708671bb413b1db61e80afb10de9bb7024ea7ae874ff6250d9ca3  /usr/local/bin/registry" >> /tmp/checksum.amd64 \
&& touch /tmp/checksum.arm64 \
&& echo "dbf5130f270400199ea97fe6b25849ef38c7d44f9e4e1c3812fff948ecba1242  /usr/local/bin/rclone" >> /tmp/checksum.arm64 \
&& echo "834b04a70c53aa8004c4fcae3dfb28e14e42d520e08bb8446383fa4c3a930ddb  /usr/local/bin/registry" >> /tmp/checksum.arm64 \
&& sha256sum -c /tmp/checksum.${TARGETARCH} \
&& rm /tmp/checksum.* \
&& addgroup -g 101 -S app \
&& adduser -u 101 -D -S -G app app \
&& chown -R 101:101 /app \
&& rm -rf /tmp/*

COPY --chown=101:101 ./gitlab-registry-cleaner /app/gitlab-registry-cleaner

USER 101

ENTRYPOINT [ "/app/gitlab-registry-cleaner" ]
