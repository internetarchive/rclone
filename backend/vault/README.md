# Rclone for Vault

Experimental CLI support for Internet Archive Vault Digital Preservation System
in [Rclone](https://rclone.org/). This is a private fork of rclone and
work-in-progress. Glad about any feedback (open or comment on issue, email to
[martin@archive.org](mailto:martin@archive.org), ...), especially on:

* bugs
* unintuitive behaviour

## Known limitations

* once an upload starts, it cannot be interrupted (or resumed)
* the *exit code* from a failed upload currently may be zero
* read and write support **only on the command line**
* partial read support (no write) for *mount* and *serve* commands

----

## Download the custom rclone binary

* [https://github.com/internetarchive/rclone/releases](https://github.com/internetarchive/rclone/releases)

Release will most likely follow a version scheme like:
`v1.57.0-vault-20220627142057-e4798bf85` (rclone version, "vault", timestamp,
commit).

## Building the custom rclone binary

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

## Configuration

There is a single configuration file for rclone, located by default under:

```
~/.config/rclone/rclone.conf
```

In you rclone config, add the following section for vault (the section name is
arbitrary; used to refer to the remote).

```ini
[vault]
type = vault
username = admin
password = admin
endpoint = http://localhost:8000/api
```

## Examples

Note: Most examples show abbreviated outputs. To show debug output, append `-v`
or `-vv` to the command.

### Quota and Usage

* [x] about

```
$ rclone about vault:/
Total:   1 TiB
Used:    2.891 GiB
Free:    1021.109 GiB
Objects: 19.955k
```

* [x] config userinfo

```
$ rclone config userinfo vault:/
DefaultFixityFrequency: TWICE_YEARLY
             FirstName:
             LastLogin: 2022-06-14T17:09:11.222339Z
              LastName:
          Organization: SuperOrg
                  Plan: Basic
            QuotaBytes: 1099511627776
              Username: admin
```

### Listing files

* [x] ls
* [x] lsl
* [x] lsf
* [x] lsd, lsd -R
* [x] lsjson

```shell
$ rclone ls vault:/
        8 C00/VERSION
        0 C1/abc.txt
     3241 C123/about.go
     4511 C123/backend.go
     3748 C123/bucket.go
     3683 C123/bucket_test.go
     2416 C123/cat.go
     2829 C123/copy.go
    10913 C123/help.go
      886 C123/ls.go
      ...

$ rclone lsl vault:/
        0 2022-06-08 23:49:10.000000000 C1/abc.txt
        8 2022-05-31 16:17:21.000000000 C00/VERSION
     3241 2022-05-31 17:13:45.000000000 C123/about.go
     4511 2022-05-31 17:17:10.000000000 C123/backend.go
     3748 2022-05-31 18:18:36.000000000 C123/bucket.go
     3683 2022-05-31 18:20:44.000000000 C123/bucket_test.go
     2416 2022-05-31 17:18:42.000000000 C123/cat.go
     2829 2022-05-31 17:09:35.000000000 C123/copy.go
    10913 2022-05-31 17:27:17.000000000 C123/help.go
      886 2022-05-31 17:06:59.000000000 C123/ls.go
      ...

$ rclone lsf vault:/
.Trash-1000/
C00/
C1/
C123/
C40/
C41/
C42/
C43/
C50/
C51/
...

$ rclone lsd vault:/
           0 2022-05-31 16:05:24         0 .Trash-1000
           0 2022-05-31 16:17:05         2 C00
           0 2022-06-08 23:49:06         1 C1
           0 2022-05-31 17:06:59        25 C123
           0 2022-06-07 15:27:55         1 C40
           0 2022-06-07 15:35:33         1 C41
           0 2022-06-07 15:44:15         1 C42
           0 2022-06-08 11:20:18         1 C43
           0 2022-06-08 13:09:09         1 C50
           0 2022-06-08 14:34:18         2 C51
           ...

$ rclone lsd -R vault:/C40
           0 2022-06-07 15:33:32         1 myblog
           0 2022-06-07 15:33:36         0 myblog/templates
           ...

$ rclone lsjson vault:/ | head -10
[
{"Path":".Trash-1000","Name":".Trash-1000","Size":0,"MimeType":"inode/directory","ModTime":"2022-05-31T14:05:24Z","IsDir":true,"ID":"1.12"},
{"Path":"C00","Name":"C00","Size":0,"MimeType":"inode/directory","ModTime":"2022-05-31T14:17:05Z","IsDir":true,"ID":"1.25"},
{"Path":"C1","Name":"C1","Size":0,"MimeType":"inode/directory","ModTime":"2022-06-08T21:49:06Z","IsDir":true,"ID":"1.38600"},
{"Path":"C123","Name":"C123","Size":0,"MimeType":"inode/directory","ModTime":"2022-05-31T15:06:59Z","IsDir":true,"ID":"1.48"},
{"Path":"C40","Name":"C40","Size":0,"MimeType":"inode/directory","ModTime":"2022-06-07T13:27:55Z","IsDir":true,"ID":"1.665"},
{"Path":"C41","Name":"C41","Size":0,"MimeType":"inode/directory","ModTime":"2022-06-07T13:35:33Z","IsDir":true,"ID":"1.674"},
{"Path":"C42","Name":"C42","Size":0,"MimeType":"inode/directory","ModTime":"2022-06-07T13:44:15Z","IsDir":true,"ID":"1.683"},
{"Path":"C43","Name":"C43","Size":0,"MimeType":"inode/directory","ModTime":"2022-06-08T09:20:18Z","IsDir":true,"ID":"1.698"},
{"Path":"C50","Name":"C50","Size":0,"MimeType":"inode/directory","ModTime":"2022-06-08T11:09:09Z","IsDir":true,"ID":"1.713"},
...
```

### Listing files and folders as tree

Similar to the linux [tree](https://en.wikipedia.org/wiki/Tree_(command)),
rclone can render a tree as well. Note that this only starts to render the
output when all the relevant files have been inspected. Hence this command can
take a while on large folders.

Options: `-d`, `-s`, ...

```
$ rclone tree vault:/C100
/
├── a
│   └── myblog
│       ├── content
│       │   └── blog
│       └── templates
├── b
│   ├── _index.md
│   ├── base.html
│   ├── blog-page.html
│   ├── blog.html
│   ├── config.toml
│   ├── first.md
│   ├── index.html
│   ├── second.md
│   └── xxxx.txt
├── c
│   └── myblog
│       ├── content
│       │   └── blog
│       └── templates
└── d
    ├── _index.md
    ├── base.html
    ├── blog-page.html
    ├── blog.html
    ├── config.toml
    ├── first.md
    ├── index.html
    ├── second.md
    └── xxxx.txt

12 directories, 18 files
```


### Creating Collections and Folder

Collections and folders are handled transparently (e.g. first path component
will be the name of the collection, and anything below: folders).

* [x] mkdir

```shell
$ rclone mkdir vault:/X1
```

By default, behaviour is similar to `mkdir -p`, i.e. parents are created, if
they do not exist:

```
$ rclone mkdir vault:/X2/a/b/c
```

### Uploading single files and directories

* [x] copy
* [x] copyto
* [x] copyurl

Copy operations to vault will create directories as needed:

```
$ rclone copy ~/tmp/somedir vault:/ExampleCollection/somedir
```

If you configure other remotes, like Dropbox, Google Drive, Amazon S3, etc. you can copy files directly from there to vault:

```
$ rclone copy dropbox:/iris-data.csv vault:/C104
```

### Sync

* [x] sync

Sync is similar to copy, can be used to successively sync file to vault.

```
$ rclone sync ~/tmp/somedir vault:/ExampleCollection/somedir
```

### Downloading files and folders

* [x] copy

Copy can be used to copy a file or folder from vault to local disk.

```
$ rclone copy vault:/ExampleCollection/somedir ~/tmp/somecopy
```

### Streaming files

* [x] cat

```
$ rclone cat vault:/ExampleCollection/somedir/f.txt
```

### Deleting files and folders

* [x] delete, deletefile

```
$ rclone delete vault:/C123/a/f.txt
```

A whole folder or collection can be deleted as well.

```
$ rclone delete vault:/C123
```

### Show disk usage

* [x] ncdu

Similar to [ncdu](https://en.wikipedia.org/wiki/Ncdu), rclone can show what dirs consume most disk space.

```
$ rclone ncdu vault:/
```

Works for folders as well. Running this against large collections may take a while.

### Listing hashes

* [x] md5sum
* [x] sha1sum
* [x] hashsum

Vault keeps track of MD5, SHA1 and SHA256 of objects and rclone is natively interested in those.

```
$ rclone md5sum vault:/
d41d8cd98f00b204e9800998ecf8427e  C103/testing-touch.txt
127a60cc6951b43d8ec9f2fbc566f53d  C102/base.org
d6c43639164bd159609fde47ae1477cc  C102/uuuu.txt
2b26b72ff91209f42e05a69c5bbff249  240/magentacloud.zip
c4b44a7043b45e0e03664827874739c9  240/Zwyns_Lbova_2018.pdf
275cc2f7f947d45c8a632dab13a8522c  240/midas.pdf
...

$ rclone sha1sum vault:/C100
a2b0031c595250996b9fd671ba543a5178a86c02  d/blog.html
e38c7b27a15bb492766686bc61cf27765f34f97e  d/base.html
785096246f488bce480e8fadcd7d4e7863c08773  d/config.toml
be3ad0ee54239c370d616b60f02736dd10137dc7  d/second.md
...

$ rclone hashsum sha256 vault:/C100
59739d7135e335183b260fa83428df7d2fba108f8398a4a21ef1da706315f2f1  d/blog.html
3aafe178d8e5bb9994bc2ecc0feb92eb63adc3023fdf0901c10bbe906542d05b  d/base.html
0d0a57a6ecb72d8f9fffb6b092037a99490d1fe20a30675b29caa0e24008dd28  d/blog-page.html
a6cfd6fc383e5856da20444a633ee5e4c23b603b27f807459186118035ed2441  d/first.md
4d3eedec138894549365ce82b54e39c79af7da4146d4d39855756623c5aad8e5  d/second.md
...
```

## TODO

* [ ] issue with `max-depth`
* [ ] ncdu performance
* [ ] resumable deposits
* [ ] cli access to various reports (fixity, ...)
* [ ] test harness


## Forum

Trying to move from "atexit" to "Shutdown", but that would require additional
changes, discussing it here:
[https://forum.rclone.org/t/support-for-returning-and-error-from-atexit-handlers-and-batch-uploads/31336](https://forum.rclone.org/t/support-for-returning-and-error-from-atexit-handlers-and-batch-uploads/31336)


## Review (MW) Notes

> ls --max-depth={1,2,..} are producing unexpected results: ...

The [ls](https://rclone.org/commands/rclone_ls/) subcommand lists size and path
of **objects only** - our root is the organization, the first level are
COLLECTION items, there is no FILE that has a an ORG as a parent
