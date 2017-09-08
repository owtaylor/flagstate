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

License
-------
metastore is distributed is distributed under the [Apache License, Version 2.0](LICENSE).
