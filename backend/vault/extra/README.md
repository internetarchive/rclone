# Rclone Vault Test Environment

Assuming current working directory is `backend/vault/extra`, i.e. where this
`README.md` lives.

If there are any leftover images, remove them first:

```
$ make clean # ~ docker rmi vault-base vault-bootstrap
```

1. Checkout vault (somewhere else) and build a base image.

```
$ docker build -t vault-base -f Dockerfile.base /path/to/git.archive.org/dps/vault-site
```

2. Build a vault bootstrap container.

```
$ docker build --no-cache -t vault-bootstrap -f Dockerfile.bootstrap .
```

3. Start vault and all components with docker-compose.

```
$ docker-compose up
```

4. Go to http://localhost:8000 on your host and log in to vault.

5. To run the rclone test suite, run from the root rclone repo:

```
$ VAULT_TEST_REMOTE_NAME=vo: go test -v ./backend/vault/
```

