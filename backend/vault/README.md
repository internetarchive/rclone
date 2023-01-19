
# Rclone with Vault Support

> [Rclone](https://rclone.org/) is a command-line program to manage files on cloud storage.

We are working on an Rclone backend for the [Vault Digital Preservation System](https://vault.archive-it.org/)
([Documentation](https://support.archive-it.org/hc/en-us/sections/7581093252628-Vault),
[Pilot](https://archive-it.org/blog/post/archive-it-partner-news-september-2021/)),
developed at the [Internet Archive](https://archive.org/). Currently, we
maintain this fork of Rclone and release versions here (but perspectively, we
would like to include this backend into the main Rclone project). We are basing
our releases on the latest version of the Rclone upstream project.

These releases are tested extensively, yet still prototypical and we are happy
about feedback: [vault@archive.org](mailto:vault@archive.org). There are also some [known limitations](#known-limitations).

With this version of Rclone, you can **list your collections** in Vault and
**upload files and folders** conveniently from **local disk** or other **cloud
providers** and **download files or folders**.

## Requirements

* An active [Vault](https://vault.archive-it.org/accounts/login/) account
* A macOS (both classic Intel-based Macs and the newer Apple Silicon Macs), Windows, or Linux machine
* Basic familiarity with the command line

## Install Rclone with Vault Support

We currently support macOS, Windows and Linux.

> You can find the latest releases under: [https://github.com/internetarchive/rclone/releases/latest](https://github.com/internetarchive/rclone/releases/latest)

Releases follow a versioning scheme that includes the Rclone version, timestamp
and commit, e.g. like: `v1.57.0-vault-20220627142057-e4798bf85` (where
`v1.57.0` is the latest version tag of rclone, `20220627142057` is the build
timestamp and `e4798bf85` is the commit hash).

* [Install on macOS](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#install-on-macos)
* [Install on Windows](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#install-on-windows)
* [Install on Linux](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#install-on-linux)

### Install on macOS

We support both classic Intel-based Macs and the newer Apple Silicon Macs
(Apple Support: [Mac computers with Apple
Silicon](https://support.apple.com/en-us/HT211814) -- the Apple Silicon chips
carry designations like [M1, M2,
...](https://en.wikipedia.org/wiki/Apple_silicon#M_series)).

We suggest you use
[Terminal.app](https://en.wikipedia.org/wiki/Terminal_(macOS)) (or any other
terminal emulator) and [curl](https://curl.se/) or
[wget](https://www.gnu.org/software/wget/) to download the binary (otherwise
you get warnings about unsigned software).
Also, for the following step, please make sure that you do not have a file
named "rclone" already in the folder where you are performing the download
(this can lead to cryptic *zsh: killed* kind of errors).

After download with `curl` or `wget` the file needs to be made executable with [chmod](https://man7.org/linux/man-pages/man1/chmod.1.html).

#### Intel-based Macs

```shell
$ curl --output rclone -L https://github.com/internetarchive/rclone/releases/download/v1.59.1-vault-20230119170251-b254084aa/rclone_1.59.1-vault-20230119170251-b254084aa_Darwin_x86_64
$ chmod +x rclone
```

#### Apple Silicon Macs

```shell
$ curl --output rclone -L https://github.com/internetarchive/rclone/releases/download/v1.59.1-vault-20230119170251-b254084aa/rclone_1.59.1-vault-20230119170251-b254084aa_Darwin_arm64
$ chmod +x rclone
```

### Install on Windows

Download the latest binary (e.g. with your browser):

* Rclone with Vault for Windows x64 64bit: [https://github.com/internetarchive/rclone/releases/download/v1.59.1-vault-20230119170251-b254084aa/rclone_1.59.1-vault-20230119170251-b254084aa_Windows_x86_64.exe](https://github.com/internetarchive/rclone/releases/download/v1.59.1-vault-20230119170251-b254084aa/rclone_1.59.1-vault-20230119170251-b254084aa_Windows_x86_64.exe)

In (the rare) case you have an ARM based computer running Windows, please
download: [https://github.com/internetarchive/rclone/releases/download/v1.59.1-vault-20230119170251-b254084aa/rclone_1.59.1-vault-20230119170251-b254084aa_Windows_arm64.exe](https://github.com/internetarchive/rclone/releases/download/v1.59.1-vault-20230119170251-b254084aa/rclone_1.59.1-vault-20230119170251-b254084aa_Windows_arm64.exe).

**Important**: We do not sign the executables, which is why Windows will issue
warnings about an untrusted source and will suggest that you delete the file.

To ensure the downloaded file is the same as the one we published, you can
compare the checksum of the file you downloaded against a list of checksums we
publish alongside each
[release](https://github.com/internetarchive/rclone/releases) (see Screenshot below). The filename
ends with `_checksums.txt` - on Windows you can generate various hash sums of a
file with
[certutil](https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/certutil),
a pre-installed command line utility;
[examples](https://superuser.com/a/898377))

![Using certutil to verify a SHA256 checksum](static/Windows_GitHub_Checksum.png)

Once downloaded, it may be convenient to rename the file. You can do this in your Explorer or with the Command Prompt with the
[ren](https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/ren)
command. Please make sure the file has an `.exe` extension, otherwise Windows
may not recognize it (an error you may see would be *rclone is not a
recognized internal or external command*).

### Install on Linux

Download the latest release depending on your architecture:

* [x64 64-bit](https://github.com/internetarchive/rclone/releases/download/v1.59.1-vault-20230119170251-b254084aa/rclone_1.59.1-vault-20230119170251-b254084aa_Linux_x86_64)
* [ARM64](https://github.com/internetarchive/rclone/releases/download/v1.59.1-vault-20230119170251-b254084aa/rclone_1.59.1-vault-20230119170251-b254084aa_Linux_arm64)

For convenience, you can rename the downloaded file to e.g. `rclone` with your
File Explorer or the [`mv`](https://man7.org/linux/man-pages/man1/mv.1.html)
command. Finally, set executable permissions:

```
$ chmod +x rclone
```

## Checkpoint: First Run


You can check if the binary works by printing out version information about the program (your output may vary depending on which system you are using).

To run the command you can either:

1. Stay in the directory where the binary is located and run it from there using:

	1. `./rclone version` on macOS and Linux
	2. `Rclone.exe version` on Windows

2. Put the binary (or a symlink to it) into your [PATH](https://en.wikipedia.org/wiki/PATH_(variable)).

```
$ rclone version
- os/version: ubuntu 20.04 (64 bit)
- os/kernel: 5.13.0-52-generic (x86_64)
- os/type: linux
- os/arch: amd64
- go/version: go1.18.3
- go/linking: dynamic
- go/tags: none
```

> All following examples in the documentation will demonstrate commands using the *PATH approach*.

## Configuring Rclone Vault Backend

To access Vault, Rclone will need to know your Vault credentials and the Vault
API endpoint.

> The current Vault API endpoint is at: [https://vault.archive-it.org/api](https://vault.archive-it.org/api)

You can configure your Vault username, password and API endpoint using the
`config` subcommand of rclone, like so (replace `alice` and `secret` to match
your credentials):

```
$ rclone config create vault vault username=alice password=secret endpoint=https://vault.archive-it.org/api
```

This will create a configuration file (or extend it, if if already existed) -
and will add a section for Vault. Rclone uses a single configuration file,
located by default under your [HOME
directory](https://en.wikipedia.org/wiki/Home_directory).

You can always ask Rclone to show you where your configuration file is located:

```
$ rclone config file
```

## First steps with the Rclone Vault backend

To check if everything works, you can e.g. run `rclone config userinfo` to
display information about the configured Vault user:

```shell
$ rclone config userinfo vault:
DefaultFixityFrequency: TWICE_YEARLY
             FirstName:
             LastLogin: 2022-07-02T00:29:20.364793Z
              LastName:
          Organization: SuperOrg
                  Plan: Basic
            QuotaBytes: 1099511627776
              Username: roosevelt
```

If you see a similar output, congratulations - your Rclone with Vault support
is now ready to use!

## Known Limitations

This is a working prototype and while continuously tested against our
development and QA Vault instances, a few limitations remain.

* ~~**uploaded files are currently not mutable** - that is, you cannot update a file with the same name but with different content (use `--ignore-existing` [global flag](https://rclone.org/flags/) to upload or synchronize files without considering existing files)~~ fixed in production since 10/2022 and currently testing
* read and write support **only on the command line** level (mount and serve are read only)
* currently, if you copy data from another cloud service to vault, **data will** be stored temporarily on the machine where rclone runs**, which means that if you want to transfer 10TB of data from a cloud service to vault, you will have to have at least 10TB of free disk space on the machine where rclone runs; if you want to upload files from the local filesystem to vault, this limitation does not apply

## Tasks

There are a few common tasks around Vault:

* [Creating a collection](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#creating-a-collection)
* [Depositing a single file](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#depositing-a-single-file-and-inspecting-the-result)
* [Depositing a directory](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#depositing-a-directory)
* [Listing a collection and folders](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#listing-collections-and-folders)
* [Syncing a local folder to Vault](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#syncing-a-folder-to-vault)
* [Download a single file](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#download-a-single-file)
* [Download a directory or collection](https://github.com/internetarchive/rclone/blob/ia-wt-1168/backend/vault/README.md#download-a-collection-or-directory)

For the example tasks we use an example local directory structure that looks
like this. We also assume for the moment that nothing has been uploaded so far.

```shell
$ tree data
data
├── a.pdf
├── b.pdf
├── c.pdf
├── d.png
└── extra
    ├── e.png
    └── examples
        └── f.png

2 directories, 6 files
```

### Creating a collection

A collection in Vault can be create explicitly with the `mkdir` command - every
top level directory corresponds to a collection in Vault.

For example, you can create a new collection called `TempSpace1` with the
following command:

```shell
$ rclone mkdir vault:/TempSpace1
```

Note that once created, collections cannot be deleted - they can only be renamed.

### Depositing a single file and inspecting the result

You can work with single files, e.g. if you want to deposit `a.pdf` into
`TempSpace1` collection, you can run:

```shell
$ rclone copy data/a.pdf vault:/TempSpace1
<5>NOTICE: vault batcher: preparing 1 file(s) for deposit
<5>NOTICE: vault (v1): deposit registered: 16
<5>NOTICE: depositing 100% |██████████████████████████████████████ ....| (341/341 kB, 3.396 MB/s)
<5>NOTICE: vault batcher: upload done (16), deposited 340.609Ki, 1 item(s)
```

To view the remote collection, you can use the `rclone ls ...` subcommand.

```shell
$ rclone ls vault:/TempSpace1
   348784 a.pdf
```

An extended listing can be show with the `lsl` subcommand:

```shell
$ rclone lsl vault:/TempSpace1
   348784 2022-08-11 22:25:27.000000000 a.pdf
```

### Depositing a directory

Often you want to deposit a whole directory and its content, recursively. Let's
deposit the whole `data` directory into a collection named `TempSpace2`.

The `data` folder looked like this:

```
$ tree data
data
├── a.pdf
├── b.pdf
├── c.pdf
├── d.png
└── extra
    ├── e.png
    └── examples
        └── f.png

2 directories, 6 files
```

Note that we do not need to create the collection first, rclone will do that for us.

```
$ rclone copy data vault:/TempSpace2
<5>NOTICE: vault batcher: preparing 5 file(s) for deposit
<5>NOTICE: vault (v1): deposit registered: 25
<5>NOTICE: depositing 100% |██████████████████████████████████████████████████████| (5.7/5.7 MB, 6.024 MB/s)
<5>NOTICE: vault batcher: upload done (25), deposited 5.698Mi, 5 item(s)
```

We can verify that the folder has been uploaded with the `tree` subcommand:


```shell
$ rclone tree vault:/TempSpace2
/
├── a.pdf
├── b.pdf
├── c.pdf
├── d.png
└── extra
    ├── e.png
    └── examples
        └── f.png

2 directories, 6 files
```

### Listing collections and folders

You can list all you collections or just specific ones, of just a folder or
subfolder.

```shell
$ rclone ls vault:/
   348784 TempSpace1/a.pdf
   348784 TempSpace2/a.pdf
  1021176 TempSpace2/b.pdf
   455725 TempSpace2/c.pdf
    55183 TempSpace2/d.png
  2235016 TempSpace2/extra/e.png
  2207711 TempSpace2/extra/examples/f.png
```

The `tree` subcommand renders the file system hierarchy as well.

```shell
$ rclone tree vault:/
/
├── TempSpace1
│   └── a.pdf
└── TempSpace2
    ├── a.pdf
    ├── b.pdf
    ├── c.pdf
    ├── d.png
    └── extra
        ├── e.png
        └── examples
            └── f.png
```

To only list everything under `TempSpace2/extra` we can specify the path in the command:

```shell
$ rclone tree vault:/TempSpace2/extra
/
├── e.png
└── examples
    └── f.png

1 directories, 2 files
```

### Syncing a folder to Vault

If you want to regularly synchronize a directory to Vault, you can use the
`sync` subcommand. By default, this will try to synchronize your local files
with the files in Vault.

As an example, let's sync our tree into a new collection `TempSpace3`.

```shell
$ rclone sync data vault:/TempSpace3
<5>NOTICE: vault batcher: preparing 6 file(s) for deposit
<5>NOTICE: vault (v1): deposit registered: 26
<5>NOTICE: depositing 100% |████████████████████████████████████████████| (6.0/6.0 MB, 5.449 MB/s)
<5>NOTICE: vault batcher: upload done (26), deposited 6.031Mi, 6 item(s)
```

Let's check if all went well:

```shell
$ rclone ls vault:/TempSpace3
   348784 a.pdf
  1021176 b.pdf
   455725 c.pdf
    55183 d.png
  2235016 extra/e.png
  2207711 extra/examples/f.png
```

Looks good. We can run `sync` again, in which case nothing should happen, since
all files are already in Vault.

```shell
$ rclone sync data vault:/TempSpace3
```

Indeed, there's no output - there was nothing to transfer.

Now let's add another file to our local directory tree:

```
$ echo "NEW" > data/new.file
```

Our local tree looks like this now:

```shell
$ tree data
data
├── a.pdf
├── b.pdf
├── c.pdf
├── d.png
├── extra
│   ├── e.png
│   └── examples
│       └── f.png
└── new.file

2 directories, 7 files
```

Now let's run `sync` again:

```shell
$ rclone sync data vault:/TempSpace3
<5>NOTICE: vault batcher: preparing 1 file(s) for deposit
<5>NOTICE: vault (v1): deposit registered: 27
<5>NOTICE: depositing 100% |█████████████████████████████████████████████████████████| ( 4/ 4B, 0.068 kB/s)
<5>NOTICE: vault batcher: upload done (27), deposited 4, 1 item(s)
```

Rclone noticed 1 new file and deposited it accordingly - let's see it in Vault:

```shell
$ rclone ls vault:/TempSpace3
   348784 a.pdf
  1021176 b.pdf
   455725 c.pdf
    55183 d.png
        4 new.file
  2235016 extra/e.png
  2207711 extra/examples/f.png
```

Indeed, `new.file` has been uploaded.

Note: A current limitation is that an already deposited file cannot be altered
- that is, you cannot upload a file with a existing name in Vault with
different content.

To workaround this issue, you can use the `--ignore-existing` [global
flag](https://rclone.org/flags/) which will skip files that exists on the
remote already (albeit content may differ).

```shell
$ rclone copy data vault:/TempSpace4 --ignore-existing
```

### Download a single file

You can download a single file from vault with the `copy` command. Note that
the last argument is the target directory, e.g. `.` for the current directory.

```shell
$ rclone copy vault:/TempSpace3/a.pdf .
```

### Download a collection or directory

Similarly, you can download a whole tree structure (e.g. a collection or a
specific folder from a collection) with the `copy` subcommand:

```
$ rclone copy vault:/TempSpace3 Downloads
```

Please note that Rclone use a convention when copying:

> Note that it is always the **contents of the directory that is synced, not the
> directory itself**. So when source:path is a directory, it's the contents of
> source:path that are copied, not the directory name and contents. --
> [https://rclone.org/commands/rclone_copy/](Rclone copy command documentation)

In the above example files and folders from `vault:/TempSpace3/` are put into
the `Downloads` directory (i.e. there is no `Downloads/TempSpace3` created).

### Note about Upload Latency

When Rclone runs commands, they are executed against the remote Vault service
and when Rclone exits without any error, your data has been successfully
transferred to Vault. Collection, file and folder *metadata* will be available
immediately after Rclone finished successfully. However, file *contents* will
be available only after a short delay, as data is processed by Vault (typically
in the range of minutes).

## Appendix: Example Commands

Rclone has [great docs on its own](https://rclone.org/docs/); the following are
a few more usage examples.

### Quick Tour

![static/506601.cast](static/506601.gif)

### Quota and Usage

```
$ rclone about vault:/
Total:   1 TiB
Used:    2.891 GiB
Free:    1021.109 GiB
Objects: 19.955k
```

Information about the user

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

### Listing Files

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

### Listing Files and Folders as a Tree

Similar to the linux [tree](https://en.wikipedia.org/wiki/Tree_(command))
command, rclone can render files and folder as a tree as well. Note that this
only starts to render the output when all the relevant files have been
inspected. Hence this command can take a while on large folders.

Options: `-d`, `-s`, ...

```shell
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
│   └── third.txt
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
    └── third.txt

12 directories, 18 files
```

### Creating Collections and Folder

Collections and folders are handled transparently (e.g. first path component
will be the name of the collection, and anything below: folders).

```shell
$ rclone mkdir vault:/X1
```

By default, behaviour is similar to `mkdir -p`, i.e. parents are created, if
they do not exist:

```shell
$ rclone mkdir vault:/X2/a/b/c
```

### Depositing / Uploading files and directories

Copy operations to vault will create directories as needed:

```shell
$ rclone copy ~/tmp/somedir vault:/ExampleCollection/somedir
```

If you configure other remotes, like Dropbox, Google Drive, Amazon S3, etc. you
can copy files directly from there to Vault (note that currently the
transferred files need to be stored temporarily on the machine that runs
vault).

```shell
$ rclone copy dropbox:/iris-data.csv vault:/C104
```

#### Resuming an Interrupted Deposit

It is possible to resume an interrupted deposit.

Assuming we want to copy local path "A" to vault "B" - we can start a deposit by
copying files. You'll see the deposit id logged to the terminal (e.g. 742):

```shell
$ rclone copy A vault:/B
<5>NOTICE: vault (v1): deposit registered: 742
...
```

You can interrupt the deposit e.g. with CTRL-C. To resume, add the
`--vault-resume-deposit-id` flag:

```shell
$ rclone copy A vault:/B --vault-resume-deposit-id 742
```

Note that resuming only makes sense when the source and destination path are the same.

### Sync

Sync is similar to copy, can be used to successively sync file to vault.

```
$ rclone sync ~/tmp/somedir vault:/ExampleCollection/somedir
```

### Downloading Files and Folders

Copy can be used to copy a file or folder from vault to local disk.

```
$ rclone copy vault:/ExampleCollection/somedir ~/tmp/somecopy
```

### Streaming Files

```
$ rclone cat vault:/ExampleCollection/somedir/f.txt
```

### Deleting Files and Folders

Note that currently, there are a few limitations around deletes: You currently
cannot reupload a file with the name of a file that has been deleted before.

```
$ rclone delete vault:/C123/a/f.txt
```

A whole folder or collection can be deleted as well.

```
$ rclone delete vault:/C123
```

### Show Disk Usage

Similar to [ncdu](https://en.wikipedia.org/wiki/Ncdu), rclone can show what
dirs consume most disk space.

```
$ rclone ncdu vault:/
```

Works for folders as well. Running this against large collections may take a while.

### Listing Hashes

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

### Vault Specific Commands

Backends can implement custom commands.

#### Deposit Status (ds, dst, deposit-status)

For vault we currently have a single command, that returns the deposit status,
given the deposit id (e.g. 742).

```shell
$ rclone backend ds vault:/ 742
```
