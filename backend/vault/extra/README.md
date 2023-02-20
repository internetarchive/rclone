# Rclone Vault Test Environment

Assuming current working directory is `backend/vault/extra`, i.e. where this
`README.md` lives.

If there are any leftover images, remove them first:

```
$ make clean # ~ docker rmi vault-base vault-bootstrap
```

1. Checkout vault. You can use an already existing checkout, if available.

```
$ git clone git@git.lab.org:vault-site /tmp/vault-site
```

2. Build a vault base and vault bootstrap images, pass your vault checkout
   directory via `VAULT` environment variable.

```
$ VAULT=/tmp/vault-site make images
```

3. Start vault and all components with docker-compose.

```
$ docker-compose up
```

4. Go to http://localhost:8000 on your host and log in to vault with
   `admin:admin` account.

5. To run the rclone test suite, run from the root rclone repo:

```
$ VAULT_TEST_REMOTE_NAME=vo: go test -v ./backend/vault/
```

