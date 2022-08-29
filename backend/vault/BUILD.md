# Build instructions

Tag, Build, Release. Get a token from [settings/tokens](https://github.com/settings/tokens) (need repo write access).

```shell
$ git tag $(git tag -l | grep -E '^v[0-9\.]+$' | tail -1)-vault-$(date +"%Y%m%d%H%M%S")-$(git rev-parse --short HEAD)
$ export GITHUB_TOKEN="ghp_mv4gc3lqnrssa5dpnnsw4idgojxw2idemv3gk3dpobsxeidtmv2hi2lom5zqu"
$ goreleaser release --rm-dist
```

A new release should be available under [https://github.com/internetarchive/rclone/releases](https://github.com/internetarchive/rclone/releases).

# Maintenance

A long as we are developing Vault in a fork, we need to regularly include commit from the master branch.
