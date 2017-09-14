#!/bin/bash
PATH=$HOME/go/bin/:$PATH
echo $PATH
set -e -x

schema_v2=application/vnd.docker.distribution.manifest.v2+json
schema_list_v2=application/vnd.docker.distribution.manifest.list.v2+json

skopeo copy --dest-tls-verify=false docker://docker.io/busybox:latest docker://localhost:7000/test/busybox:latest

skopeo copy --dest-tls-verify=false docker://docker.io/busybox:latest docker://localhost:7000/test.list/busybox:latest
digest=$(curl -s -H "Accept: $schema_v2" --head  http://localhost:7000/v2/test.list/busybox/manifests/latest | grep Docker-Content-Digest | tr -d \\015 | awk '{ print $2 }')
size=$(curl -s -H "Accept: $schema_v2" --head  http://localhost:7000/v2/testlist/busybox/manifests/latest | grep Content-Length | tr -d \\015  | awk '{ print $2 }')

cat > busybox-manifest-list.json<<EOF
{
  "schemaVersion": 2,
  "mediaType": "$schema_list_v2",
  "manifests": [
    {
      "mediaType": "$schema_v2",
      "size": $size,
      "digest": "$digest",
      "platform": {
        "architecture": "amd64",
        "os": "linux"
      }
    }
  ]
}
EOF

curl -H "Content-Type: application/vnd.docker.distribution.manifest.list.v2+json" --upload-file busybox-manifest-list.json http://localhost:7000/v2/test.list/busybox/manifests/latest

skopeo copy --dest-tls-verify=false oci:banner-flatpak-oci docker://localhost:7000/flatpak/banner:latest

skopeo copy --dest-tls-verify=false oci:banner-flatpak-oci docker://localhost:7000/flatpak.list/banner:latest
curl -H "Content-Type: application/vnd.oci.image.index.v1+json" --upload-file banner-flatpak-oci/index.json http://localhost:7000/v2/flatpak.list/banner/manifests/latest


