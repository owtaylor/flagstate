#!/bin/bash
set -e -x

cd /mnt/test-data
while read src arch dest ; do
    arch_arg=
    if [ "$arch" != "*" ] ; then
	arch_arg="--arch=$arch"
    fi
    registry_copy.py --dest-tls-verify=false $src $arch_arg docker:registry:5000/$dest
done < contents
