module github.com/rclone/rclone

go 1.21

require (
	bazil.org/fuse v0.0.0-20230120002735-62a210ff1fd5
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.13.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.7.0
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.3.2
	github.com/Azure/azure-sdk-for-go/sdk/storage/azfile v1.2.2
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358
	github.com/Max-Sum/base32768 v0.0.0-20230304063302-18e6ce5945fd
	github.com/Unknwon/goconfig v1.0.0
	github.com/a8m/tree v0.0.0-20240104212747-2c8764a5f17e
	github.com/aalpar/deheap v0.0.0-20210914013432-0cc84d79dec3
	github.com/abbot/go-http-auth v0.4.0
	github.com/anacrolix/dms v1.5.0
	github.com/anacrolix/log v0.13.1
	github.com/antchfx/htmlquery v1.3.0
	github.com/atotto/clipboard v0.1.4
	github.com/aws/aws-sdk-go v1.54.19
	github.com/buengese/sgzip v0.1.1
	github.com/cloudsoda/go-smb2 v0.0.0-20231124195312-f3ec8ae2c891
	github.com/colinmarc/hdfs/v2 v2.4.0
	github.com/coreos/go-semver v0.3.1
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/deepmap/oapi-codegen v1.12.4
	github.com/dop251/scsu v0.0.0-20220106150536-84ac88021d00
	github.com/dropbox/dropbox-sdk-go-unofficial/v6 v6.0.5
	github.com/gabriel-vasile/mimetype v1.4.3
	github.com/gdamore/tcell/v2 v2.6.0
	github.com/getkin/kin-openapi v0.114.0
	github.com/go-chi/chi/v5 v5.0.8
	github.com/go-git/go-billy/v5 v5.5.0
	github.com/google/uuid v1.6.0
	github.com/hanwen/go-fuse/v2 v2.2.1-0.20230410213758-80c1c8221982
	github.com/henrybear327/Proton-API-Bridge v1.0.0
	github.com/henrybear327/go-proton-api v1.0.0
	github.com/jcmturner/gokrb5/v8 v8.4.4
	github.com/jlaffaye/ftp v0.2.0
	github.com/josephspurrier/goversioninfo v1.4.0
	github.com/jzelinskie/whirlpool v0.0.0-20201016144138-0675e54bb004
	github.com/klauspost/compress v1.17.9
	github.com/koofr/go-httpclient v0.0.0-20240520111329-e20f8f203988
	github.com/koofr/go-koofrclient v0.0.0-20221207135200-cbd7fc9ad6a6
	github.com/mattn/go-colorable v0.1.13
	github.com/mattn/go-runewidth v0.0.15
	github.com/minio/minio-go/v7 v7.0.74
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/sys/mountinfo v0.6.2
	github.com/ncw/swift/v2 v2.0.1
	github.com/oracle/oci-go-sdk/v65 v65.32.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/sftp v1.13.6
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2
	github.com/prometheus/client_golang v1.19.1
	github.com/putdotio/go-putio/putio v0.0.0-20200123120452-16d982cac2b8
	github.com/rclone/gofakes3 v0.0.3-0.20240716093803-d6abc178be56
	github.com/rfjakob/eme v1.1.2
	github.com/rivo/uniseg v0.4.7
	github.com/rogpeppe/go-internal v1.12.0
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/sirupsen/logrus v1.9.3
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	github.com/t3rm1n4l/go-mega v0.0.0-20240219080617-d494b6a8ace7
	github.com/willscott/go-nfs v0.0.2
	github.com/winfsp/cgofuse v1.5.1-0.20221118130120-84c0898ad2e0
	github.com/xanzy/ssh-agent v0.3.3
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a
	github.com/yunify/qingstor-sdk-go/v3 v3.2.0
	go.etcd.io/bbolt v1.3.10
	goftp.io/server/v2 v2.0.1
	golang.org/x/crypto v0.25.0
	golang.org/x/exp v0.0.0-20240716175740-e3f259677ff7
	golang.org/x/net v0.27.0
	golang.org/x/oauth2 v0.21.0
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.22.0
	golang.org/x/text v0.16.0
	golang.org/x/time v0.5.0
	google.golang.org/api v0.188.0
	gopkg.in/validator.v2 v2.0.1
	gopkg.in/yaml.v2 v2.4.0
	storj.io/uplink v1.13.0
)

require (
	cloud.google.com/go/auth v0.7.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	cloud.google.com/go/compute/metadata v0.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/ProtonMail/bcrypt v0.0.0-20211005172633-e235017c1baf // indirect
	github.com/ProtonMail/gluon v0.17.1-0.20230724134000-308be39be96e // indirect
	github.com/ProtonMail/go-mime v0.0.0-20230322103455-7d82a3887f2f // indirect
	github.com/ProtonMail/go-srp v0.0.7 // indirect
	github.com/ProtonMail/gopenpgp/v2 v2.7.4 // indirect
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/akavel/rsrc v0.10.2 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/antchfx/xpath v1.2.3 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bradenaw/juniper v0.15.2 // indirect
	github.com/calebcase/tmpfile v1.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/cronokirby/saferith v0.33.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emersion/go-message v0.18.0 // indirect
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594 // indirect
	github.com/emersion/go-vcard v0.0.0-20230815062825-8fda7d206ec9 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/flynn/noise v1.0.1 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/geoffgarside/ber v1.1.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/go-resty/resty/v2 v2.7.0 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.5 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/yaml v0.1.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jtolio/noiseconn v0.0.0-20230111204749-d7ec1a08b0b8 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/pengsrc/go-shared v0.2.1-0.20190131101655-1999055a4a14 // indirect
	github.com/perimeterx/marshmallow v1.1.4 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20221212215047-62379fc7944b // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rasky/go-xdr v0.0.0-20170124162913-1a41d1a06c93 // indirect
	github.com/relvacode/iso8601 v1.3.0 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryszard/goskiplist v0.0.0-20150312221310-2dfbae5fcf46 // indirect
	github.com/shabbyrobe/gocovmerge v0.0.0-20230507112040-c3350d9342df // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	github.com/spacemonkeygo/monkit/v3 v3.0.22 // indirect
	github.com/tklauser/go-sysconf v0.3.13 // indirect
	github.com/tklauser/numcpus v0.7.0 // indirect
	github.com/willscott/go-nfs-client v0.0.0-20240104095149-b44639837b00 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/blake3 v0.2.3 // indirect
	github.com/zeebo/errs v1.3.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	golang.org/x/mod v0.19.0 // indirect
	golang.org/x/tools v0.23.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240708141625-4ad9e859172b // indirect
	google.golang.org/grpc v1.64.1 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	storj.io/common v0.0.0-20240424123607-5f226fc92c16 // indirect
	storj.io/drpc v0.0.33 // indirect
	storj.io/eventkit v0.0.0-20240306141230-6cb545e5f892 // indirect
	storj.io/infectious v0.0.2 // indirect
	storj.io/picobuf v0.0.3 // indirect
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/ProtonMail/go-crypto v1.0.0
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/pkg/xattr v0.4.9
	golang.org/x/mobile v0.0.0-20240716161057-1ad2df20a8b6
	golang.org/x/term v0.22.0
)
