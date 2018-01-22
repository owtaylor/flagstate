#!/bin/bash
PGPASSWORD=mypassword psql -h 127.0.0.1 -U flagstate -p 7432 "$@"

