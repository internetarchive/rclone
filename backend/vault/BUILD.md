# Build instructions

Tag, Build, Release with [goreleaser](https://goreleaser.com/). Get a GitHub Personal Access Token
from [settings/tokens](https://github.com/settings/tokens) (need repo write
access).

```shell
$ git tag $(git tag -l | grep -E '^v[0-9\.]+$' | tail -1)-vault-$(date +"%Y%m%d%H%M%S")-$(git rev-parse --short HEAD)
$ # will result in something like: v1.59.1-vault-20221010201333-6eb36a82f
$ export GITHUB_TOKEN="ghp_mv4gc3lqnrssa5dpnnsw4idgojxw2idemv3gk3dpobsxeidtmv2hi2lom5zqu"
$ goreleaser release --rm-dist
```

A new release should become available under
[https://github.com/internetarchive/rclone/releases](https://github.com/internetarchive/rclone/releases)
within a minute.

## Post-Release Task

The
[README](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md)
contains various link, which need to be updated after a new release has been
published. There is a `README.gen.sh` generator script, that will inspect the
latest release on GitHub and will print a README to stdout (the generator
script uses the same `GITHUB_TOKEN` environment variable to access the GitHub
API).

```
$ cd vault/backend
$ ./README.gen.sh > README.md
$ git commit -m "vault: update readme"
$ git push
```

# Maintenance

A long as we are developing Vault in a fork, we need to regularly include commit from the master branch.
