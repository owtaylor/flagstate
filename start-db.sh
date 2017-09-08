#!/bin/bash

refresh=false
if [ "$1" = "--refresh" ] ; then
    refresh=true
fi

if docker ps -f name=metastore-db | grep metastore-db > /dev/null ; then
    if $refresh ; then
	docker rm -f metastore-db
    else
	echo "Already running"
	exit 1
    fi
fi

exists=false
if docker volume ls | grep metastore-db > /dev/null ; then
    exists=true
fi

if $refresh ; then
    docker volume rm metastore-db || true
    exists=false
fi

docker run					\
       --detach                                 \
       --rm=true				\
       --name=metastore-db			\
       -v metastore-db:/var/lib/pgsql/data	\
       -p 7432:5432				\
       -e POSTGRES_DB=metastore			\
       -e POSTGRES_USER=metastore		\
       -e POSTGRES_PASSWORD=mypassword		\
       postgres

while true ; do
    if $(dirname $0)/psql.sh -l 2>/dev/null | grep metastore ; then
	break
    fi
    sleep 1
done

if ! $exists ; then
    $(dirname $0)/psql.sh < schema.sql
fi
