# Notes

Integration test (local):

```
$ VAULT_TEST_REMOTE_NAME=vault: go test -v ./backend/vault/...
```

* 11 PASS
* 6 SKIP
* 7 FAIL (plus subtests)

Mostly, `duplicate key` ... -- if a file exists, we just return that or nil error?

```
dev-postgres-1  | 2022-08-04 13:26:47.197 UTC [3434] ERROR:  duplicate key value violates unique constraint "vault_collection_org_and_name"
dev-postgres-1  | 2022-08-04 13:26:47.197 UTC [3434] DETAIL:  Key (organization_id, name)=(1, trailing dot.) already exists.
dev-postgres-1  | 2022-08-04 13:26:47.197 UTC [3434] STATEMENT:  INSERT INTO "vault_collection" ("name", "organization_id", "target_replication", "fixity_frequency", "tree_node_id") VALUES ('trailing dot.', 1, 2, 'TWICE_YEARLY', NULL) RETURNING "vault_collection"."id"
```


```shell
--- FAIL: TestIntegration (109.41s)
    --- SKIP: TestIntegration/FsCheckWrap (0.00s)
    --- PASS: TestIntegration/FsCommand (0.00s)
    --- PASS: TestIntegration/FsRmdirNotFound (0.13s)
    --- PASS: TestIntegration/FsString (0.00s)
    --- PASS: TestIntegration/FsName (0.00s)
    --- PASS: TestIntegration/FsRoot (0.00s)
    --- FAIL: TestIntegration/FsRmdirEmpty (0.07s)
    --- FAIL: TestIntegration/FsMkdir (108.54s)
        --- PASS: TestIntegration/FsMkdir/FsMkdirRmdirSubdir (2.68s)
        --- PASS: TestIntegration/FsMkdir/FsListEmpty (0.07s)
        --- PASS: TestIntegration/FsMkdir/FsListDirEmpty (0.12s)
        --- SKIP: TestIntegration/FsMkdir/FsListRDirEmpty (0.00s)
        --- PASS: TestIntegration/FsMkdir/FsListDirNotFound (0.09s)
        --- SKIP: TestIntegration/FsMkdir/FsListRDirNotFound (0.00s)
        --- FAIL: TestIntegration/FsMkdir/FsEncoding (89.63s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/control_chars (5.26s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot (5.22s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot_dot (5.22s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/punctuation (0.26s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_space (5.36s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_tilde (5.22s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_CR (5.19s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_LF (5.21s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_HT (5.25s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_VT (5.25s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_dot (5.20s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_space (5.27s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_CR (5.28s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_LF (5.26s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_HT (5.21s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_VT (5.24s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_dot (5.22s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/invalid_UTF-8 (0.18s)
            --- FAIL: TestIntegration/FsMkdir/FsEncoding/URL_encoding (5.23s)
        --- PASS: TestIntegration/FsMkdir/FsNewObjectNotFound (0.16s)
        --- FAIL: TestIntegration/FsMkdir/FsPutError (0.00s)
        --- FAIL: TestIntegration/FsMkdir/FsPutZeroLength (5.02s)
        --- SKIP: TestIntegration/FsMkdir/FsOpenWriterAt (0.00s)
        --- SKIP: TestIntegration/FsMkdir/FsChangeNotify (0.00s)
        --- FAIL: TestIntegration/FsMkdir/FsPutFiles (5.08s)
        --- SKIP: TestIntegration/FsMkdir/FsPutChunked (0.00s)
        --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize (5.14s)
            --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsPutUnknownSize (0.10s)
            --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsUpdateUnknownSize (5.03s)
        --- PASS: TestIntegration/FsMkdir/FsRootCollapse (0.36s)
    --- FAIL: TestIntegration/FsShutdown (0.14s)
FAIL
FAIL    github.com/rclone/rclone/backend/vault  109.415s
?       github.com/rclone/rclone/backend/vault/api      [no test files]
?       github.com/rclone/rclone/backend/vault/cache    [no test files]
?       github.com/rclone/rclone/backend/vault/extra    [no test files]
FAIL
```
