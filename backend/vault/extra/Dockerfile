# syntax = docker/dockerfile:1
#
# Containerfile for vault, build with "vault-site" repo as context. Please take
# a look at the Makefile for how this Dockerfile is used.
#
# TODO(martin): apt-get update makes this image non-reproducible; we assume
# that this does not affect vault
#
FROM python:3.12.3-slim-bookworm@sha256:afc139a0a640942491ec481ad8dda10f2c5b753f5c969393b12480155fe15a63
RUN apt-get update && apt-get install -y --no-install-recommends make build-essential git curl libmagic-dev
WORKDIR /app
COPY . .
# patch run workers script, so we run only a minimal set of workers we need for
# rclone testing
RUN patch --verbose 0001-minimal-workers.patch

RUN make clean
# we need uv for venv management and deps
RUN pip install uv
# https://docs.docker.com/build/cache/#use-the-dedicated-run-cache
RUN --mount=type=cache,target=/root/.cache make install

# https://github.com/ufoscout/docker-compose-wait, since we need a DB for bootstrap.sh
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.12.1/wait /wait
RUN echo "6c340afe7608744d099834972ca243ee5eaed6f1 /wait" | sha1sum --check && \
    chmod +x /wait

# this will be the bootstrap script that runs, when the vault containers runs
ADD dev/bootstrap.sh bootstrap.sh
RUN echo "a45490b064f32b3cbe61a0c9df08c951f9d5f1c1 ./bootstrap.sh" | sha1sum --check && \
    chmod +x ./bootstrap.sh

# TODO: we may remove these values here, because vault will need a docker
# compose file anyway
# ENV REDIS_HOST=redis
# ENV VAULT_POSTGRES_HOST=postgres
# ENV TEMPORAL_FRONTEND_URI=temporal:7233
# ENV CELERY_BROKER_URI=amqp://vault:vault@rabbitmq:5672/
# ENV PROMETHEUS_MULTIPROC_DIR=/tmp

# USCAS301 uses S3 minio by default; use USCAS301 to get S3 path included; use
# DEVNULL to discard contents (note: while using DEVNULL, temporal will still
# try to store data inside a bucket)
# ENV PRIMARY_STORAGE_MANAGER=USCAS301
# ENV GLOBAL_DEFAULT_REPLICA_LOCATIONS=USCAS301
# ENV PRIMARY_STORAGE_MANAGER=DEVNULL
# ENV S3_CHUNK_MANAGER_S3_ENDPOINT=http://minio:9000
# ENV S3_STORAGE_MANAGER_USCAS301_S3_ENDPOINT=http://minio:9000
# ENV S3_STORAGE_MANAGER_USCAS301_S3_ACCESS_KEY=accesskey
# ENV S3_STORAGE_MANAGER_USCAS301_S3_SECRET_KEY=secretkey
# set bucket explicitly here; we also need to instruct minio to prepare the bucket
# ENV S3_STORAGE_MANAGER_USCAS301_S3_BUCKET=s3storagemanagerdevstorage
# ENV CHUNK_MANAGER_TYPE=LOCALFS
# ENV ASYNC_MODE=TEMPORAL

# cf. WAIT_HOSTS, ... in docker-compose.yml
CMD /wait && ./bootstrap.sh

