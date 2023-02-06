# Rclone Vault Test Environment

1. Checkout vault (somewhere else) and build a base image.

```
$ docker build --no-cache --network "host" -t vault-base -f Dockerfile.base /path/to/git.archive.org/dps/vault-site
```

2. Start components from scratch.

```
$ docker-compose down # if necessary
$ docker-compose up
```

3. Run initial database setup and derive an image.

```
$ docker build --no-cache --network "host" -t vault -f Dockerfile.db .
```

4. Run server so rclone tests can use it.

```
$ docker run --network "host" --rm -it vault make run
```

After that, an image `vault` is setup for testing and ready to run.

## TODO

* use ephemeral volumes, so we can get rid of the `rm v` stuff




----


* remove any previous local volumes under `./v`
* build a vault-base image from git checkout, use vault commit id as tag
* run `docker-compose up` for the components





Goal: We want to have E2E tests for rclone and vault.

    $ cd backend/vault/dev
    $ rm -rf ./v # container volumes
    $ docker-compose up

Checkout the vault repo and build an image:

    $ time docker build --network "host" -t vault -f .../rclone/backend/vault/dev/Dockerfile .

In the future we may use testcontainers.

## Setup

We can start a complete vault environment with docker compose.

TODO(martin):

* [ ] build vanilla vault image from checkout

```shell
$ docker-compose --file backend/vault/dev/docker-compose.yml up
```

TODO:

* document the steps
* Dockerfile.base
* Dockerfile.db
* dumpdata

```
$ docker run --network="host" --rm vault ./venv/bin/python manage.py dumpdata  --natural-foreign > fixture.json
```

To load the data:

```
$ cat /home/tir/code/git.archive.org/martin/rclone/backend/vault/dev/fixture.json | docker run --network="host" -i -a stdin --rm vault ./venv/bin/python manage.py loaddata --format json -
```
