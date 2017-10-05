Metastore
---------
metastore is a service designed to be deployed alongside a
[docker/distribution](https://github.com/docker/distribution/) registry instance
to collect and serve up metadata for images in the registry. Being separate from
the registry has a number of advantages:

 * The filesystem-like blob storage that the registry uses is not suitable for
   searching on metadata.
 * The metadata server may have different needs for scaling than the main registry
 * Not all sites that want to stand up a registry need metadata storage

See https://github.com/docker/distribution/issues/206

Some possible ways that metastore could be used:

 * Providing comprehensive dumps of metadata
 * Providing a simple web interface to allow seeing what is in the registry
 * Providing an implementation of the `/v1/search/` API behind `docker search`

Design
------
The basic idea is that metastore scans the registry, either entirely, or as
updated by [webhook notifications](https://docs.docker.com/registry/notifications/),
and then the information that is harvested is stored into a database.
The current code stores the metadata in a Postgresql database, using a
number of Postgresql-specific features, such as `ON CONFLICT` and the `jsonb`
data type; the storage is kept abstract so different backends would be possible.

Receiving notifications from the registry and updating the database must be
done by only a single server, but other servers can be deployed in a
read-only fashion to handle queries. The database would be the bottleneck for
heavy usage unless queries could be cached via a front-end cache.

Deployment
----------

Configuration is done by a yaml file:

``` yaml
registry:
    url: https://registry.example.com
	# This is an URL to the registry that will be returned in request bodies. It can be
	# an absolute URL, or a relative URL with a full path. Defaults to the value of url.
	public_url: /
components:
    # If true, a basic web user interface will be provided at /
    web_ui: true
    # If true, an interface for programmatic testing will be enabled at /assert
    assert_endpoint: false
events:
	# If set to an non-empty value, a 'Authorization: Bearer <token>' must be
	# present for webhook notification posts to the /events endpoint
	token: "<token>"
database:
	# Information about the database backend; postgres is the only backend at the moment
    postgres:
		# Standard postgresql URI
		# https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
		# Login information can also be passed inthe normal fashion via
		# PGUSER/PGPASSWORD/PGPASSFILE environment variables.
        url: postgres://<username>:<password>@<host>:<port>/<database>?sslmode=disable
cache:
	# Cache-Control: max-age=<> for programmatic queries
	max_age_index: 5s
	# Cache-Control: max-age=<> for the HTML user interface
	max_age_html: 5s
interval:
	# How frequently a global scan of the registry is done. 0 for never.
    fetch_all: 1h
	# How often images that are no longer referenced will be deleted from the registry
    garbage_collect: 30m
```

The database needs to be populated by sourcing the `schema.sql` file.

```
psql -d <url> < schema.sql
```

**Note** that `schema.sql` currently drops any existing content from the database,
assuming that it can simply be refetched from the registry.

Development
-----------
There is a test environment that can be started with:

```
docker-compose up
```

This starts a cluster of database, registry, metastore and a web proxy that joins
the registry and metastore into a single web presence, available on
http://127.0.0.1:7080, or https://127.0.0.1:7443. On Fedora or RHEL you can

```
make trust-local
```

To add the CA cert for the https URL to your system cert store and a hostname to
/etc/hosts. (Note: modifies your system config with sudo!), and then you can access
the web frontend on https://registry.local.fishsoup.net:7443. (The point of this
complexity is mostly to allow is to allow testing tools that access the registry/index
via HTTPS and expect a verifiable certificate.)

License
-------
metastore is distributed is distributed under the [Apache License, Version 2.0](LICENSE).
