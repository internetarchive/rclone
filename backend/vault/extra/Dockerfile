# syntax = docker/dockerfile:experimental
#
# Containerfile for vault, build with vault-site as context
#
FROM python:3.8.16-slim-bullseye
RUN apt-get update && apt-get install -y --no-install-recommends make build-essential git
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/root/.cache make clean && make setup

# wait script, TODO(martin): make a copy
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.9.0/wait /wait
RUN echo "189a4a2b04b072a1fd5e7c346877ba50d565a2c0  /wait" | sha1sum --check
RUN chmod +x /wait

# copy bootstrap script, TODO(martin): make a copy
ADD https://raw.githubusercontent.com/internetarchive/rclone/ia-wt-1168/backend/vault/extra/bootstrap.sh bootstrap.sh
RUN echo "f5c74f6547ef9578fad12d9cb67d57c689eedb2c" | sha1sum --check
RUN chmod +x ./bootstrap.sh

# components, settings
ENV REDIS_HOST=redis
ENV VAULT_POSTGRES_HOST=postgres
ENV TEMPORAL_FRONTEND_URI=temporal:7233
ENV CELERY_BROKER_URI=amqp://vault:vault@rabbitmq:5672/
ENV S3_CHUNK_MANAGER_S3_ENDPOINT=http://minio:9000
ENV PRIMARY_STORAGE_MANAGER=USCAS301
ENV S3_STORAGE_MANAGER_USCAS301_S3_ENDPOINT=http://minio:9000
ENV S3_STORAGE_MANAGER_USCAS301_S3_ACCESS_KEY=accesskey
ENV S3_STORAGE_MANAGER_USCAS301_S3_SECRET_KEY=secretkey
ENV CHUNK_MANAGER_TYPE=LOCALFS
ENV ASYNC_MODE=TEMPORAL

CMD /wait && ./bootstrap.sh

