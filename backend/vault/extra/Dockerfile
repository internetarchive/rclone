# syntax = docker/dockerfile:1
#
# Containerfile for vault, build with "vault-site" repo as context. Please take
# a look at the Makefile for how this Dockerfile is used.
#
# TODO: apt-get update makes this image non-reproducible; we assume that this does not affect vault
#
FROM python:3.8.16-slim-bullseye@sha256:13856bb0fadf6bc658125bae44d41ffd3cc42e423c399cfc7300d41a47f66237
RUN apt-get update && apt-get install -y --no-install-recommends make build-essential git
WORKDIR /app
COPY . .
RUN make clean

# https://docs.docker.com/build/cache/#use-the-dedicated-run-cache
RUN --mount=type=cache,target=/root/.cache make setup

# https://github.com/ufoscout/docker-compose-wait, since we need a DB for bootstrap.sh
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.12.1/wait /wait
RUN echo "6c340afe7608744d099834972ca243ee5eaed6f1 /wait" | sha1sum --check && \
    chmod +x /wait

ADD dev/bootstrap.sh bootstrap.sh
RUN echo "c1086e7b28092bc0b3f7be27f1add5526bfb82ab ./bootstrap.sh" | sha1sum --check && \
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

