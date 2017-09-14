#!/bin/sh

psql() {
    PGPASSWORD=mypassword /usr/bin/psql -h db -U metastore "$@"
}

topdir=$(dirname $0)/../..

while true ; do
    if psql -l | grep -q metastore ; then
	break
    fi
    sleep 1
done

exists=false
if echo '\dt' | psql 2>/dev/null | grep -q image_tag ; then
    exists=true
fi

if ! $exists ; then
    psql < schema.sql
fi

exec $topdir/metastore -config $(dirname $0)/config.yaml
