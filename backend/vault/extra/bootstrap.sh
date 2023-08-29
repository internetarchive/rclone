#!/bin/bash

# When the containerized vault application starts up, we want the database
# already populated. For that we need the database to be up.
#
# Using `build` in docker-compose won't solve this: https://stackoverflow.com/q/75386770/89391
#
# Solution: use docker-compose-wait && ./bootstrap.sh as CMD

set -eu -o pipefail
set -x # debug

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
[{"model":"vault.treenode","pk":1,"fields":{"node_type":"ORGANIZATION","parent":null,"path":"1","name":"testlib","md5_sum":null,"sha1_sum":null,"sha256_sum":null,"size":0,"file_count":0,"file_type":null,"pbox_path":null,"created_at":"2023-08-22T16:51:55.389Z","uploaded_at":null,"pre_deposit_modified_at":null,"modified_at":"2023-08-22T16:51:55.389Z","deleted_at":null,"uploaded_by":null,"comment":null,"deleted":false,"flow_identifier":null,"original_relative_path":null,"upload_state":"REGISTERED","deposit":null,"next_fixity_report":null}}]
EOF1

cat << 'EOF2' > "$F0002TESTUSER"
[{"model":"vault.plan","pk":1,"fields":{"name":"default","price_per_terabyte":"1000.00","default_replication":2,"default_fixity_frequency":"TWICE_YEARLY"}},{"model":"vault.organization","pk":1,"fields":{"name":"testlib","plan":1,"quota_bytes":1099511627776,"quota_consumed_bytes":0,"tree_node":1,"org_group":null}},{"model":"vault.user","fields":{"password":"pbkdf2_sha256$320000$uI1SS4WyCgGvlveX50A7Uq$+MoJht7S8+h8dhihL41b0Bxbepbf/KSJFMXIfxpt7Sc=","last_login":"2023-08-22T16:47:31Z","is_superuser":true,"username":"admin","first_name":"","last_name":"","is_staff":true,"is_active":true,"date_joined":"2023-08-22T16:46:57Z","email":"martin.czygan@gmail.com","organization":1,"role":"USER","groups":[],"user_permissions":[],"authorized_collections":[]}},{"model":"admin.logentry","pk":1,"fields":{"action_time":"2023-08-22T16:51:43.721Z","user":["admin"],"content_type":["vault","plan"],"object_id":"1","object_repr":"default","action_flag":1,"change_message":"[{\"added\": {}}]"}},{"model":"admin.logentry","pk":2,"fields":{"action_time":"2023-08-22T16:51:55.398Z","user":["admin"],"content_type":["vault","organization"],"object_id":"1","object_repr":"testlib","action_flag":1,"change_message":"[{\"added\": {}}]"}},{"model":"admin.logentry","pk":3,"fields":{"action_time":"2023-08-22T16:52:07.112Z","user":["admin"],"content_type":["vault","user"],"object_id":"1","object_repr":"admin","action_flag":2,"change_message":"[{\"changed\": {\"fields\": [\"Organization\"]}}]"}}]
EOF2

./venv/bin/python manage.py loaddata "$F0001TREENODE"
./venv/bin/python manage.py loaddata "$F0002TESTUSER"

rm -f "$F0001TREENODE"
rm -f "$F0002TESTUSER"

# run workers in the background; taken from Makefile: make run-workers; we do
# not care how these are teared down
make run-workers &

# start web app
RUNSERVER_ADDRPORT=0.0.0.0:8000 make run
