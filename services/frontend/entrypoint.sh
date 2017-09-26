#!/bin/bash

set -e

if ! [ -e /etc/pki/tls/certs/metastore.crt ] ; then
    generate-cert.sh
fi

echo -n "Waiting for registry ... "
while ! curl -s http://registry:5000/v2 > /dev/null ; do
    sleep 1
done
echo "started"

echo -n "Waiting for index  ... "
while ! curl -s http://index:8088/ > /dev/null ; do
    sleep 1
done
echo "started"

if ! check-for-data.py http://registry:5000 ; then
    create-test-data.sh
fi

exec httpd -DNO_DETACH -DFOREGROUND
