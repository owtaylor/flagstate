#!/bin/bash
PGPASSWORD=mypassword psql -h 127.0.0.1 -U metastore -p 7432 "$@"

