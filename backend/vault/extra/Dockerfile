# syntax = docker/dockerfile:1
#
# Containerfile for vault, build with "vault-site" repo as context. Please take
# a look at the Makefile for how this Dockerfile is used.
#
# TODO(martin): apt-get update makes this image non-reproducible; we assume
# that this does not affect vault
#
FROM python:3.12.3-slim-bookworm@sha256:afc139a0a640942491ec481ad8dda10f2c5b753f5c969393b12480155fe15a63
RUN apt-get update && apt-get install -y --no-install-recommends make build-essential git curl
WORKDIR /app
COPY . .
RUN make clean

RUN pip install uv
# https://docs.docker.com/build/cache/#use-the-dedicated-run-cache
RUN --mount=type=cache,target=/root/.cache make install

# https://github.com/ufoscout/docker-compose-wait, since we need a DB for bootstrap.sh
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.12.1/wait /wait
RUN echo "6c340afe7608744d099834972ca243ee5eaed6f1 /wait" | sha1sum --check && \
    chmod +x /wait

ADD dev/bootstrap.sh bootstrap.sh
RUN echo "366e048a9f2831691b80336aac16a2a3bacc3cc2 ./bootstrap.sh" | sha1sum --check && \
    chmod +x ./bootstrap.sh

ENV REDIS_HOST=redis
ENV VAULT_POSTGRES_HOST=postgres
ENV TEMPORAL_FRONTEND_URI=temporal:7233
ENV CELERY_BROKER_URI=amqp://vault:vault@rabbitmq:5672/

# USCAS301 uses S3 minio by default; use USCAS301 to get S3 path included; use
# DEVNULL to discard contents
ENV PRIMARY_STORAGE_MANAGER=DEVNULL
# ENV PRIMARY_STORAGE_MANAGER=USCAS301
ENV S3_CHUNK_MANAGER_S3_ENDPOINT=http://minio:9000
ENV S3_STORAGE_MANAGER_USCAS301_S3_ENDPOINT=http://minio:9000
ENV S3_STORAGE_MANAGER_USCAS301_S3_ACCESS_KEY=accesskey
ENV S3_STORAGE_MANAGER_USCAS301_S3_SECRET_KEY=secretkey
ENV CHUNK_MANAGER_TYPE=LOCALFS
ENV ASYNC_MODE=TEMPORAL

# cf. WAIT_HOSTS, ... in docker-compose.yml
CMD /wait && ./bootstrap.sh

