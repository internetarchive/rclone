# Notes

## Overview

For rclone tests, the idea is to have a fully ephemeral, fresh-off-the-source
docker image ready to run.

```shell
$ cd backend/vault/extra
$ make # and make clean, to remove the docker image again
```

This does roughly the following:

```shell
cp ./bootstrap.sh $(VAULT)/dev
cp ./0001-minimal-workers.patch $(VAULT)
DOCKER_BUILDKIT=1 docker build -t vault:latest -f Dockerfile $(VAULT)
rm -f $(VAULT)/dev/bootstrap.sh
rm -f $(VAULT)/0001-minimal-workers.patch
```

Vault needs a little bit of manual setup, e.g. creating and "org", and this is
done in bootstrap.sh with the help of fixtures currently.

One the docker image is built, one should be able to run vault plus all its adjacent components with:

```shell
$ docker-compose up
```





## Integration tests

Integration test (local):

```
$ VAULT_TEST_REMOTE_NAME=vault: go test -v ./backend/vault/...
```

* 10 PASS
* 6 SKIP
* 7 FAIL (plus subtests)

Mostly, `duplicate key` ... -- if a file exists, we just return that or nil error?

```
dev-postgres-1  | 2022-08-04 13:26:47.197 UTC [3434] ERROR:  duplicate key value violates unique constraint "vault_collection_org_and_name"
dev-postgres-1  | 2022-08-04 13:26:47.197 UTC [3434] DETAIL:  Key (organization_id, name)=(1, trailing dot.) already exists.
dev-postgres-1  | 2022-08-04 13:26:47.197 UTC [3434] STATEMENT:  INSERT INTO "vault_collection" ("name", "organization_id", "target_replication", "fixity_frequency", "tree_node_id") VALUES ('trailing dot.', 1, 2, 'TWICE_YEARLY', NULL) RETURNING "vault_collection"."id"
```


```shell
--- FAIL: TestIntegration (106.86s)
    --- SKIP: TestIntegration/FsCheckWrap (0.00s)
    --- PASS: TestIntegration/FsCommand (0.00s)
    --- PASS: TestIntegration/FsRmdirNotFound (0.11s)
    --- PASS: TestIntegration/FsString (0.00s)
    --- PASS: TestIntegration/FsName (0.00s)
    --- PASS: TestIntegration/FsRoot (0.00s)
    --- FAIL: TestIntegration/FsRmdirEmpty (0.12s)
    --- FAIL: TestIntegration/FsMkdir (106.02s)
        --- FAIL: TestIntegration/FsMkdir/FsMkdirRmdirSubdir (0.36s)
        --- PASS: TestIntegration/FsMkdir/FsListEmpty (0.07s)
        --- PASS: TestIntegration/FsMkdir/FsListDirEmpty (0.12s)
        --- SKIP: TestIntegration/FsMkdir/FsListRDirEmpty (0.00s)
        --- PASS: TestIntegration/FsMkdir/FsListDirNotFound (0.09s)
        --- SKIP: TestIntegration/FsMkdir/FsListRDirNotFound (0.00s)
        --- FAIL: TestIntegration/FsMkdir/FsEncoding (89.27s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/control_chars (5.26s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot (5.20s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot_dot (5.23s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/punctuation (0.19s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_space (5.21s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_tilde (5.25s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_CR (5.23s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_LF (5.27s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_HT (5.31s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_VT (5.23s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_dot (5.19s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_space (5.31s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_CR (5.17s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_LF (5.24s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_HT (5.26s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_VT (5.19s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_dot (5.24s)
            --- SKIP: TestIntegration/FsMkdir/FsEncoding/invalid_UTF-8 (0.00s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/URL_encoding (5.21s)
        --- PASS: TestIntegration/FsMkdir/FsNewObjectNotFound (0.18s)
        --- FAIL: TestIntegration/FsMkdir/FsPutError (0.00s)
        --- FAIL: TestIntegration/FsMkdir/FsPutZeroLength (5.06s)
        --- SKIP: TestIntegration/FsMkdir/FsOpenWriterAt (0.00s)
        --- SKIP: TestIntegration/FsMkdir/FsChangeNotify (0.00s)
        --- FAIL: TestIntegration/FsMkdir/FsPutFiles (5.07s)
        --- SKIP: TestIntegration/FsMkdir/FsPutChunked (0.00s)
        --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize (5.21s)
            --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsPutUnknownSize (0.12s)
            --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsUpdateUnknownSize (5.09s)
        --- PASS: TestIntegration/FsMkdir/FsRootCollapse (0.39s)
    --- FAIL: TestIntegration/FsShutdown (0.23s)
FAIL
```

----

# Development Notes

## Building the custom rclone binary from source

Building requires the Go toolchain installed.

```
$ git clone git@github.com:internetarchive/rclone.git
$ cd rclone
$ git checkout ia-wt-1168
$ make
$ ./rclone version
rclone v1.59.0-beta.6244.66b9ef95f.sample
- os/version: ubuntu 20.04 (64 bit)
- os/kernel: 5.13.0-48-generic (x86_64)
- os/type: linux
- os/arch: amd64
- go/version: go1.18.3
- go/linking: dynamic
- go/tags: none
```

## Debug output

To show debug output, append `-v` or `-vv` to the command.

## Valid Vault Path Rules

As per `assert_key_valid_for_write` method from PetaBox.

> bucket is the pbox identifier

> key is the file path not including the bucket

* [x] key cannot be empty
* [x] name cannot be bucket + `_files.xml`
* [x] name cannot be bucket + `_meta.xml`
* [x] name cannot be bucket + `_meta.sqlite`
* [x] name cannot be bucket + `_reviews.xml`
* [x] key cannot start with a slash
* [x] key cannot contain consecutive slashes, e.g. `//`
* [x] cannot exceed `PATH_MAX`
* [x] when key is split on `/` it cannot contain `.` or `..`
* [x] components cannot be longer than `NAME_MAX`
* [x] key cannot contain NULL byte
* [x] key cannot contain newline
* [x] key cannot contain carriage return
* [x] key must be valid unicode
* [x] `contains_xml_incompatible_characters` must be false

## TODO

A few issues to address.

* [x] issue with `max-depth`
* [ ] ncdu performance
* [x] resumable deposits
* [ ] cli access to various reports (fixity, ...)
* [ ] test harness
* [ ] full read-write support for "mount" and "serve" mode
* [ ] when a deposit is interrupted, a few stale files may remain, leading to unexpected results

## Forum

* [x] Trying to move from "atexit" to "Shutdown", but that would require additional
changes, discussing it here:
[https://forum.rclone.org/t/support-for-returning-and-error-from-atexit-handlers-and-batch-uploads/31336](https://forum.rclone.org/t/support-for-returning-and-error-from-atexit-handlers-and-batch-uploads/31336)

## Testing on Windows

Report of a failed upload from Windows, 2023-04-15.

```
Looks like there is a path issue (or a docs issue) depositing files on Windows. Running .\rclone.exe copy 2023-04-07_18-36-11_scan_00001.tif vault:/TempSpace1
I get:
023/04/14 16:34:11 Failed to copy: open /?/C:/Users/gw234478/projects/vault/2023-04-07_18-36-11_scan_00001.tif: The filename, directory name, or volume label syntax is incorrect.
It’s the same if I try different prefixes with the path (\, .\) or give it the full path, put it in quotes, etc. Not sure if this is just an rclone thing or something that needs better docs. Look like it does actually upload the file, but oddly I can’t download it though the UI.
It also looks like my path experiments threw a bunch of stuff in there. Works fine on WSL or native CentOS. Everything not in TempSpace2 or ua746 was from a windows upload.
I also put in a minor PR for the docs: https://github.com/internetarchive/rclone/pull/1. Happy to write the path issue up as an issue, but looks like they’re disabled.
```

* [https://stackoverflow.com/questions/24782826/the-filename-directory-name-or-volume-label-syntax-is-incorrect-inside-batch](https://stackoverflow.com/questions/24782826/the-filename-directory-name-or-volume-label-syntax-is-incorrect-inside-batch)
