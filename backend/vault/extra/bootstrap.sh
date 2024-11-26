#!/bin/bash

# When the containerized vault application starts up, we want the database
# already populated. For that we need the database to be up.
#
# In addition, we need a search index to be present.
#
# Using `build` in docker-compose won't solve this: https://stackoverflow.com/q/75386770/89391
#
# Solution: use docker-compose-wait && ./bootstrap.sh as CMD

set -eu -o pipefail
set -x # debug

echo "but first, for something completely different"

# create elasticsearch index for vault search
INDEX_NAME=vault-treenode-metadata-00001
curl -sL -XPUT http://elasticsearch:9200/$INDEX_NAME
curl -s http://elasticsearch:9200/
curl -s http://elasticsearch:9200/$INDEX_NAME

# https://www.elastic.co/guide/en/elasticsearch/reference/current/fix-watermark-errors.html
curl -X PUT "http://elasticsearch:9200/_cluster/settings?pretty" -H 'Content-Type: application/json' -d'
{
  "persistent": {
    "cluster.routing.allocation.disk.watermark.low": "90%",
    "cluster.routing.allocation.disk.watermark.low.max_headroom": "100GB",
    "cluster.routing.allocation.disk.watermark.high": "95%",
    "cluster.routing.allocation.disk.watermark.high.max_headroom": "20GB",
    "cluster.routing.allocation.disk.watermark.flood_stage": "97%",
    "cluster.routing.allocation.disk.watermark.flood_stage.max_headroom": "5GB",
    "cluster.routing.allocation.disk.watermark.flood_stage.frozen": "97%",
    "cluster.routing.allocation.disk.watermark.flood_stage.frozen.max_headroom": "5GB"
  }
}
'
curl -X PUT "http://elasticsearch:9200/*/_settings?expand_wildcards=all&pretty" -H 'Content-Type: application/json' -d'
{
  "index.blocks.read_only_allow_delete": null
}
'

cat <<EOM

Y8b Y88888P                   888   d8     888 88b,                       d8           d8
 Y8b Y888P   ,"Y88b 8888 8888 888  d88     888 88P'  e88 88e   e88 88e   d88    dP"Y  d88   888,8,  ,"Y88b 888 88e
  Y8b Y8P   "8" 888 8888 8888 888 d88888   888 8K   d888 888b d888 888b d88888 C88b  d88888 888 "  "8" 888 888 888b
   Y8b Y    ,ee 888 Y888 888P 888  888     888 88b, Y888 888P Y888 888P  888    Y88D  888   888    ,ee 888 888 888P
    Y8P     "88 888  "88 88"  888  888     888 88P'  "88 88"   "88 88"   888   d,dP   888   888    "88 888 888 88"
                                                                                                           888
EOM


make migrate

# create superuser admin:admin
DJANGO_SUPERUSER_PASSWORD=admin DJANGO_SUPERUSER_USERNAME=admin DJANGO_SUPERUSER_EMAIL=admin@example.com ./venv/bin/python manage.py createsuperuser --noinput

# fixtures are exported manually from a manually set up vault instance
#
#   $ ./venv/bin/python manage.py dumpdata \
#           --exclude auth.permission \
#           --exclude contenttypes \
#           --natural-primary \
#           --natural-foreign > fixture.json
#
# note: need to split out the treenode creation, so it can be loaded
# separately, first (otherwise, we got a "treenode missing" error)

F0001TREENODE="$(mktemp bootstrap-fixture-0001-XXXXXXXX-treenode.json)"
F0002TESTUSER="$(mktemp bootstrap-fixture-0002-XXXXXXXX-testuser.json)"

# TODO(martin): loaddata should accept data from stdin (https://code.djangoproject.com/ticket/27978)
cat << 'EOF1' > "$F0001TREENODE"
[{"model":"vault.treenode","pk":1,"fields":{"node_type":"ORGANIZATION","parent":null,"path":"1","name":"testlib","md5_sum":null,"sha1_sum":null,"sha256_sum":null,"size":0,"file_count":0,"file_type":null,"created_at":"2024-05-22T11:32:04.319Z","uploaded_at":null,"pre_deposit_modified_at":null,"modified_at":"2024-05-22T11:32:04.319Z","deleted_at":null,"uploaded_by":null,"comment":null,"deleted":false,"flow_identifier":null,"original_relative_path":null,"upload_state":"REGISTERED","deposit":null,"next_fixity_report":null,"metadata":null}}]
EOF1

cat << 'EOF2' > "$F0002TESTUSER"
[{"model":"vault.organization","pk":1,"fields":{"name":"testlib","quota_bytes":1099511627776,"quota_consumed_bytes":0,"tree_node":1,"org_group":"testgroup"}},{"model":"vault.user","fields":{"password":"pbkdf2_sha256$320000$HskYIEdvzxeRSnou3gJ8CF$95b5Y3DfGh1BSjNif7Q8FtCHRKLUhoZVrAJgiyYsvuM=","last_login":"2024-05-22T11:30:56Z","is_superuser":true,"username":"admin","first_name":"","last_name":"","is_staff":true,"is_active":true,"date_joined":"2024-05-22T11:30:26Z","email":"admin@example.com","organization":1,"role":"USER","groups":[],"user_permissions":[],"authorized_collections":[]}},{"model":"vault.indexerentry","pk":1,"fields":{"created_at":"2024-05-22T11:32:04.306Z","table_name":"vault_treenode","operation":"INSERT","old_json":null,"new_json":"{\"id\":1,\"name\":\"testlib\",\"parent_id\":null,\"path\":\"1\",\"comment\":null,\"deleted_at\":null,\"file_type\":null,\"md5_sum\":null,\"node_type\":\"ORGANIZATION\",\"sha1_sum\":null,\"sha256_sum\":null,\"size\":0,\"uploaded_by_id\":null,\"modified_at\":\"2024-05-22T11:32:04.319713+00:00\",\"uploaded_at\":null,\"pre_deposit_modified_at\":null,\"deleted\":false,\"file_count\":0,\"created_at\":\"2024-05-22T11:32:04.319649+00:00\",\"deposit_id\":null,\"flow_identifier\":null,\"original_relative_path\":null,\"upload_state\":\"REGISTERED\",\"next_fixity_report\":null,\"metadata\":null}"}},{"model":"admin.logentry","pk":1,"fields":{"action_time":"2024-05-22T11:32:04.353Z","user":["admin"],"content_type":["vault","organization"],"object_id":"1","object_repr":"testlib","action_flag":1,"change_message":"[{\"added\": {}}]"}},{"model":"admin.logentry","pk":2,"fields":{"action_time":"2024-05-22T11:32:09.859Z","user":["admin"],"content_type":["vault","user"],"object_id":"1","object_repr":"admin","action_flag":2,"change_message":"[{\"changed\": {\"fields\": [\"Organization\"]}}]"}}]
EOF2

./venv/bin/python manage.py loaddata "$F0001TREENODE"
./venv/bin/python manage.py loaddata "$F0002TESTUSER"

rm -f "$F0001TREENODE"
rm -f "$F0002TESTUSER"

# download "mc" minio tool (don't need that with DEVNULL)
mkdir -p /usr/local/bin
# TODO: move this to nexus
curl -sL --fail https://dl.min.io/client/mc/release/linux-amd64/mc --create-dirs -o /usr/local/bin/mc
# this is to notice upstream changes and to track them, manually for now
# echo '070f831f1df265ca7de913e6be0174a7555cb3e9 /usr/local/bin/mc' | sha1sum -c
# chmod +x /usr/local/bin/mc
chmod +x ./mc
# The default configuration stores deposited content in a local S3 daemon. The
# bucket in which replica content is stored must be created manually.
#
# create organization bucket; this is manual; for chunks it is done on the fly
# in s3_chunk_manager.py:_put
# important: this name must correspond to the ENV VAR set in settings

# > mc commands that operate on S3-compatible services require specifying an
# alias for that service. -- https://min.io/docs/minio/linux/reference/minio-mc/mc-alias.html
./mc alias set vault http://minio:9000 accesskey secretkey
./mc mb vault/s3storagemanagerdevstorage

# run workers in the background; taken from Makefile: make run-workers; we do
# not care how these are teared down
make run-workers &

# start web app
RUNSERVER_ADDRPORT=0.0.0.0:8000 make run
