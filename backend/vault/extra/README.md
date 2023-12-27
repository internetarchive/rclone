# Rclone Vault Test Environment (docker)

Assuming current working directory is `backend/vault/extra`, i.e. where this
`README.md` lives.

Stop any leftover containers with `docker-compose down` first.

1. If there are any leftover images, remove them first:

```
$ make clean
```

2. Checkout vault. You can use an already existing checkout, if available.

```
$ git clone --depth 1 git@git.lab.org:vault-site /tmp/vault-site
```

3. Build a vault image, pass your vault checkout directory via `VAULT`
   environment variable.

```
$ VAULT=/tmp/vault-site make image
```

4. Start vault and all components with docker-compose.

```
$ docker-compose up
```

5. Go to http://localhost:8000 on your host and log in to vault with
   `admin:admin` account.

6. To run the rclone test suite, run from the root rclone repo (replace "vo"
   with your local rclone name):

```
$ VAULT_TEST_REMOTE_NAME=vo: go test -v ./backend/vault/
```

## Limitations

* The `bootstrap.sh` script creates the superuser account, but executed on
  every `docker-compose up` - so it will fail the second time it run (since the
  user is already there)

