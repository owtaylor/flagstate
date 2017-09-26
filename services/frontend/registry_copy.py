#!/usr/bin/python3

"""
Copyright (c) 2017 Red Hat, Inc
All rights reserved.

This software may be modified and distributed under the terms
of the BSD license. See the LICENSE file for details.
"""


from __future__ import unicode_literals
import argparse
import json
import logging
import os
import requests
from requests.exceptions import SSLError, ConnectionError
from requests.auth import AuthBase
import re
import shutil
import tempfile
from urllib.parse import urlparse, urlunparse, urlencode
import www_authenticate

import sys

logger = logging.getLogger('registry_copy')
logging.basicConfig(level=logging.INFO)

MEDIA_TYPE_MANIFEST_V2 = 'application/vnd.docker.distribution.manifest.v2+json'
MEDIA_TYPE_LIST_V2 = 'application/vnd.docker.distribution.manifest.list.v2+json'
MEDIA_TYPE_OCI = 'application/vnd.oci.image.manifest.v1+json'
MEDIA_TYPE_OCI_INDEX = 'application/vnd.oci.image.index.v1+json'


def registry_hostname(registry):
    """
    Strip a reference to a registry to just the hostname:port
    """
    if registry.startswith('http:') or registry.startswith('https:'):
        return urlparse(registry).netloc
    else:
        return registry


class Config(object):
    def __init__(self, path):
        self.json_secret_path = path
        try:
            with open(self.json_secret_path) as fp:
                self.json_secret = json.load(fp)
        except Exception:
            msg = "failed to read registry secret"
            logger.error(msg, exc_info=True)
            raise RuntimeError(msg)

    def get_credentials(self, docker_registry):
        # For maximal robustness we check the host:port of the passed in
        # registry against the host:port of the items in the secret. This is
        # somewhat similar to what the Docker CLI does.
        #
        docker_registry = registry_hostname(docker_registry)
        try:
            return self.json_secret[docker_registry]
        except KeyError:
            for reg, creds in self.json_secret.items():
                if registry_hostname(reg) == docker_registry:
                    return creds

            logger.warn('%s not found in %s', docker_registry, self.json_secret_path)
            return {}


class DirectorySpec(object):
    def __init__(self, directory):
        self.directory = directory

    def get_endpoint(self):
        return DirectoryEndpoint(self.directory)


class RegistrySpec(object):
    def __init__(self, registry, repo, tag, creds, tls_verify):
        if registry == 'docker.io':
            self.registry = 'registry-1.docker.io'
        else:
            self.registry = registry
        self.repo = repo
        self.tag = tag
        self.creds = creds
        self.tls_verify = tls_verify

    def get_session(self):
        return RegistrySession(self.registry, insecure = not self.tls_verify,
                               creds=self.creds)

    def get_endpoint(self):
        return RegistryEndpoint(self)


def parse_spec(parser, spec, creds, tls_verify):
    if spec.startswith('dir:'):
        _, directory = spec.split(':', 1)
        if creds:
            parser.print_usage()
            parser.exit("Credentials can't be specified for a directory")
        return DirectorySpec(directory)
    elif spec.startswith('docker:'):
        _, rest = spec.split(':', 1)

        parts = rest.split('/', 1)
        if len(parts) == 1:
            parser.exit("Registry specification should be docker:REGISTRY<PATH[:TAG]")

        registry, path = parts
        parts = path.split(':', 1)
        if len(parts) == 1:
            repo, tag = parts[0], 'latest'
        else:
            repo, tag = parts

        return RegistrySpec(registry, repo, tag, creds, tls_verify)
    else:
        parser.print_usage()
        parser.exit("Unknown source/destination: {}".format(spec))


class BearerAuth(AuthBase):
    def __init__(self, token):
        self.token = token

    def __call__(self, r):
        r.headers['Authorization'] = 'Bearer ' + self.token
        return r


class RegistrySession(object):
    def __init__(self, registry, insecure=False, creds=None):
        self.registry = registry
        self._resolved = None
        self.insecure = insecure

        self.auth = None
        if creds is not None:
            username, password = creds.split(':', 1)
            self.auth = requests.auth.HTTPBasicAuth(username, password)

        self._fallback = None
        if re.match('http(s)?://', self.registry):
            self._base = self.registry
        else:
            self._base = 'https://{}'.format(self.registry)
            if insecure:
                # In the insecure case, if the registry is just a hostname:port, we
                # don't know whether to talk HTTPS or HTTP to it, so we try first
                # with https then fallback
                self._fallback = 'http://{}'.format(self.registry)

        self.session = requests.Session()

    def _get_token_auth(self, res):
        parsed = www_authenticate.parse(res.headers['www-authenticate'])
        if not 'bearer' in parsed:
            return

        challenge = parsed['bearer']
        realm=challenge.get('realm')
        service=challenge.get('service')
        scope=challenge.get('scope')

        if not service and scope and realm:
            return False

        url = realm + '?' + urlencode([('service', service), ('scope', scope)])
        res = requests.get(url, auth=self.auth)
        if res.status_code != 200:
            return False

        token = res.json()['token']
        self.auth = BearerAuth(token)

        return True

    def _do(self, f, relative_url, *args, **kwargs):
        kwargs['auth'] = self.auth
        kwargs['verify'] = not self.insecure
        res = None
        if self._fallback:
            try:
                res = f(self._base + relative_url, *args, **kwargs)
                self._fallback = None  # don't fallback after one success
            except (SSLError, ConnectionError):
                self._base = self._fallback
                self._fallback = None
        if res is None:
            res = f(self._base + relative_url, *args, **kwargs)
        if res.status_code == requests.codes.UNAUTHORIZED:
            if self._get_token_auth(res):
                kwargs['auth'] = self.auth
                res = f(self._base + relative_url, *args, **kwargs)
        return res

    def get(self, relative_url, data=None, **kwargs):
        return self._do(self.session.get, relative_url, **kwargs)

    def head(self, relative_url, data=None, **kwargs):
        return self._do(self.session.head, relative_url, **kwargs)

    def post(self, relative_url, data=None, **kwargs):
        return self._do(self.session.post, relative_url, data=data, **kwargs)

    def put(self, relative_url, data=None, **kwargs):
        return self._do(self.session.put, relative_url, data=data, **kwargs)

    def delete(self, relative_url, **kwargs):
        return self._do(self.session.delete, relative_url, **kwargs)


class ManifestInfo(object):
    def __init__(self, contents, digest, media_type, size):
        self.contents = contents
        self.digest = digest
        self.media_type = media_type
        self.size = size


def get_manifest(session, repository, ref):
    """
    Downloads a manifest from a registry. ref can be a digest, or a tag.
    """
    logger.debug("%s: Retrieving manifest for %s:%s", session.registry, repository, ref)

    headers = {
        'Accept': ', '.join((
            MEDIA_TYPE_MANIFEST_V2,
            MEDIA_TYPE_LIST_V2,
            MEDIA_TYPE_OCI,
            MEDIA_TYPE_OCI_INDEX
        ))
    }

    url = '/v2/{}/manifests/{}'.format(repository, ref)
    response = session.get(url, headers=headers)
    response.raise_for_status()
    return ManifestInfo(response.content,
                        response.headers['Docker-Content-Digest'],
                        response.headers['Content-Type'],
                        int(response.headers['Content-Length']))


class DirectoryEndpoint(object):
    def __init__(self, directory, already_exists=False):
        self.directory = directory
        self.already_exists = already_exists

    def start(self):
        if not self.already_exists:
            if os.path.exists(self.directory):
                raise RuntimeError("{} already exists", self.directory)
            os.makedirs(self.directory)

        with open(os.path.join(self.directory, 'oci-layout'), 'w') as f:
            f.write('{"imageLayoutVersion": "1.0.0"}\n')

    def cleanup(self):
        if not self.already_exists:
            shutil.rmtree(self, self.directory)

    def get_blob_path(self, digest):
        algorithm, digest = digest.split(':', 2)
        return os.path.join(self.directory, 'blobs', algorithm, digest)

    def get_blob(self, digest):
        with open(self.get_blob_path(digest), 'rb') as f:
            return f.read()

    def has_blob(self, digest):
        return os.path.exists(self.get_blob_path(digest))

    def write_blob(self, digest, contents):
        path = self.get_blob_path(digest)
        parent = os.path.dirname(path)
        if not os.path.exists(parent):
            os.makedirs(parent)
        with open(path, 'wb') as f:
            f.write(contents)

    def get_manifest(self, digest=None, media_type=None):
        if digest is None:
            with open(os.path.join(self.directory, 'index.json'), 'rb') as f:
                contents = f.read()
            parsed = json.loads(contents)
            media_type = parsed.get('mediaType', MEDIA_TYPE_OCI)
        else:
            contents = self.get_blob(digest)

        return ManifestInfo(contents, digest, media_type, len(contents))

    def _get_arch_from_config(self, info):
        parsed = json.loads(info.contents)
        config_blob = self.get_blob(parsed['config']['digest'])
        parsed_config = json.loads(config_blob)

        return parsed_config['architecture']

    def write_manifest(self, info, toplevel=False):
        if toplevel and info.media_type in (MEDIA_TYPE_LIST_V2, MEDIA_TYPE_OCI_INDEX):
            with open(os.path.join(self.directory, 'index.json'), 'wb') as f:
                f.write(info.contents)
        else:
            self.write_blob(info.digest, info.contents)
            if toplevel:
                if info.media_type == MEDIA_TYPE_MANIFEST_V2:
                    list_type = MEDIA_TYPE_LIST_V2
                elif info.media_type == MEDIA_TYPE_OCI:
                    list_type = MEDIA_TYPE_OCI_INDEX
                else:
                    raise RuntimeError('manifest has an unsupported type: {}'
                                       .format(media_type))

                arch = self._get_arch_from_config(info)
                index_contents = json.dumps({
                    "schemaVersion": 2,
                    "mediaType": list_type,
                    "manifests": [
                        {
                            "mediaType": info.media_type,
                            "size": info.size,
                            "digest": info.digest,
                            "platform": {
                                "architecture": arch,
                                "os": "linux"
                            }
                        }
                    ],
                }, indent=4)
                with open(os.path.join(self.directory, 'index.json'), 'w') as f:
                    f.write(index_contents)


class RegistryEndpoint(object):
    def __init__(self, spec):
        self.session = spec.get_session()
        self.registry = spec.registry
        self.repo = spec.repo
        self.tag = spec.tag

    def start(self):
        pass

    def cleanup(self):
        pass

    def download_blob(self, digest, size, blob_path):
        logger.info("%s: Downloading %s (size=%s)", self.registry, blob_path, size)

        url = "/v2/{}/blobs/{}".format(self.repo, digest)
        result = self.session.get(url, stream=True)
        result.raise_for_status()

        parent = os.path.dirname(blob_path)
        if not os.path.exists(parent):
            os.makedirs(parent)

        try:
            with open(blob_path, 'wb') as f:
                for block in result.iter_content(10 * 1024):
                    f.write(block)
        finally:
            result.close()

    def upload_blob(self, digest, size, blob_path):
        logger.info("%s: Uploading %s (size=%s)", self.registry, blob_path, size)

        url = "/v2/{}/blobs/uploads/".format(self.repo)
        result = self.session.post(url, data='')
        result.raise_for_status()

        if result.status_code != requests.codes.ACCEPTED:
            raise RuntimeError("Unexpected successful response %s", result.status_code)

        upload_url = result.headers.get('Location')
        parsed = urlparse(upload_url)
        if parsed.query == '':
            query = 'digest=' + digest
        else:
            query = parsed.query + '&digest=' + digest
        relative = urlunparse(('', '', parsed.path, parsed.params, query, ''))

        headers = {
            'Content-Length': str(size),
            'Content-Type': 'application/octet-stream'
        }
        with open(blob_path, 'rb') as f:
            result = self.session.put(relative, data=f, headers=headers)

        result.raise_for_status()
        if result.status_code != requests.codes.CREATED:
            raise RuntimeError("Unexpected successful response %s", result.status_code)

    def link_blob(self, digest, src_repo):
        logger.info("%s: Linking blob %s from %s to %s",
                     self.registry, digest, src_repo, self.repo)

        # Check that it exists in the source repository
        url = "/v2/{}/blobs/{}".format(src_repo, digest)
        result = self.session.head(url)
        if result.status_code == requests.codes.NOT_FOUND:
            logger.debug("%s: blob %s, not present in %s, skipping",
                         self.registry, digest, src_repo)
            # Assume we don't need to copy it - maybe it's a foreign layer
            return
        result.raise_for_status()

        url = "/v2/{}/blobs/uploads/?mount={}&from={}".format(self.repo, digest, src_repo)
        result = self.session.post(url, data='')
        result.raise_for_status()

        if result.status_code != requests.codes.CREATED:
            # A 202-Accepted would mean that the source blob didn't exist and
            # we're starting an upload - but we've checked that above
            raise RuntimeError("Blob mount had unexpected status {}".format(result.status_code))

    def has_blob(self, digest):
        url = "/v2/{}/blobs/{}".format(self.repo, digest)
        result = self.session.head(url, stream=True)
        if result.status_code == 404:
            return False
        result.raise_for_status()
        return True

    def get_manifest(self, digest=None, media_type=None):
        if digest is None:
            return get_manifest(self.session, self.repo, self.tag)
        else:
            return get_manifest(self.session, self.repo, digest)

    def write_manifest(self, info, toplevel=False, arch=None):
        if toplevel:
            ref = self.tag
        else:
            ref = info.digest

        logger.info("%s: Storing manifest as %s", self.registry, ref)

        url = '/v2/{}/manifests/{}'.format(self.repo, ref)
        headers = {'Content-Type': info.media_type}
        response = self.session.put(url, data=info.contents, headers=headers)
        response.raise_for_status()


class Copier(object):
    def __init__(self, src, dest, arch=None):
        self.src = src
        self.dest = dest
        self.arch = arch

    def _copy_blob(self, digest, size):
        if self.dest.has_blob(digest):
            return

        if isinstance(self.src, RegistryEndpoint) and isinstance(self.dest, DirectoryEndpoint):
            self.src.download_blob(digest, size, self.dest.get_blob_path(digest))
        elif isinstance(self.src, DirectoryEndpoint) and isinstance(self.dest, RegistryEndpoint):
            self.dest.upload_blob(digest, size, self.src.get_blob_path(digest))
        elif isinstance(self.src, RegistryEndpoint) and isinstance(self.dest, RegistryEndpoint):
            if self.src.registry == self.dest.registry:
                self.dest.link_blob(digest, self.src.repo)
            else:
                raise RuntimeError("Direct copying between repositories not implemented")
        else:
            raise RuntimeError("Source and destination can't both be directories")

    def _copy_manifest(self, info, toplevel=False):
        references = []
        if info.media_type in (MEDIA_TYPE_MANIFEST_V2, MEDIA_TYPE_OCI):
            manifest = json.loads(info.contents)
            references.append((manifest['config']['digest'], manifest['config']['size']))
            for layer in manifest['layers']:
                references.append((layer['digest'], layer['size']))
        else:
            raise RuntimeError("Unhandled media type %s", info.media_type)

        for digest, size in references:
            self._copy_blob(digest, size)

        self.dest.write_manifest(info, toplevel=toplevel)

    def _do_copy(self):
        info = self.src.get_manifest()
        if info.media_type in (MEDIA_TYPE_MANIFEST_V2, MEDIA_TYPE_OCI):
            self._copy_manifest(info, toplevel=True)
        elif info.media_type in (MEDIA_TYPE_LIST_V2, MEDIA_TYPE_OCI_INDEX):
            manifest = json.loads(info.contents)
            if self.arch is not None:
                for m in manifest['manifests']:
                    if m['platform']['architecture'] == self.arch:
                        referenced = self.src.get_manifest(digest=m['digest'], media_type=m['mediaType'])
                        self._copy_manifest(referenced, toplevel=True)
                        return
                raise RuntimeError("Couldn't find architecture {} in manifest".format(self.arch))

            for m in manifest['manifests']:
                referenced = self.src.get_manifest(digest=m['digest'], media_type=m['mediaType'])
                self._copy_manifest(referenced)
            self.dest.write_manifest(info, toplevel=True)

        else:
            raise RuntimeError("Unhandled media type %s", media_type)

    def copy(self):
        try:
            self.dest.start()
            self._do_copy()
        except:
            self.dest.cleanup()
            raise


parser = argparse.ArgumentParser(prog='registry-copy')
parser.add_argument('src', metavar='{dir:DIR,docker:REGISTRY<PATH[:TAG]}',
                    help="Directory or docker registry reference to copy from")
parser.add_argument('dest', metavar='{dir:DIR,docker:REGISTRY<PATH[:TAG]}',
                    help="Directory or docker registry reference to copy to")
parser.add_argument('--config', metavar='CONFIG_JSON',
                    help="Config file")
parser.add_argument('--src-creds', metavar='USERNAME:PASSWORD',
                    help="Credentials to log into the source docker registry")
parser.add_argument('--dest-creds', metavar='USERNAME:PASSWORD',
                    help="Credentials to log into the destination docker registry")
parser.add_argument('--src-tls-verify', choices=['true', 'false'], default='true',
                    help="Whether to verify TLS cerificates for the source")
parser.add_argument('--dest-tls-verify', choices=['true', 'false'], default='true',
                    help="Whether to verify TLS certificates for the destination")
parser.add_argument('--arch',
                    help="Extract only a single architecture from a manifest list or image index")

args = parser.parse_args()
args.src_tls_verify = args.src_tls_verify == 'true'
args.dest_tls_verify = args.src_tls_verify == 'true'

src = parse_spec(parser, args.src, args.src_creds, args.src_tls_verify)
dest = parse_spec(parser, args.dest, args.dest_creds, args.dest_tls_verify)

if isinstance(src, DirectorySpec) and isinstance(dest, DirectorySpec):
    parser.exit("Source and destination can't both be directories")
elif isinstance(src, RegistrySpec) and isinstance(dest, RegistrySpec) and src.registry != dest.registry:
    with tempfile.TemporaryDirectory(suffix=None, prefix=None, dir=None) as tempdir:
        tmp = DirectoryEndpoint(tempdir, already_exists=True)
        Copier(src.get_endpoint(), tmp, arch=args.arch).copy()
        Copier(tmp, dest.get_endpoint(), arch=args.arch).copy()
else:
        Copier(src.get_endpoint(), dest.get_endpoint(), arch=args.arch).copy()
