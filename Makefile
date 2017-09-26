all: hooks binary

hooks:
	@: ; \
	gitdir=$$(git rev-parse --git-dir) ; \
	if [ -d $$gitdir/hooks ] ; then \
	    cmp $$gitdir/../util/pre-commit  $$gitdir/hooks/pre-commit 2>/dev/null || \
	         echo "Updating $$gitdir/hooks/pre-commit" && cp $$gitdir/../util/pre-commit $$gitdir/hooks/pre-commit ; \
	fi

binary:
	go build

test:
	go test

coverage:
	go test -coverprofile=coverage.out && go tool cover -html=coverage.out

reset-data:
	docker-compose down || true
	docker volume rm metastore_db metastore_registry || true

trust-local:
	docker-compose exec frontend cat /etc/pki/tls/certs/metastore_ca.crt > metastore.crt
	sudo sh -c 'cp metastore.crt /etc/pki/ca-trust/source/anchors/ && update-ca-trust'
	sudo sh -c 'grep -l registry.local.fishsoup.net /etc/hosts > /dev/null || echo "127.0.0.1	registry.local.fishsoup.net" >> /etc/hosts'
	rm -f metastore.crt

untrust-local:
	sudo sh -c 'rm /etc/pki/ca-trust/source/anchors/metastore.crt && update-ca-trust'
	sudo sh -c 'sed -i /registry.local.fishsoup.net/d /etc/hosts'

README.html: README.md codehilite.css Makefile
	( echo '<!DOCTYPE html><html><head><link rel="stylesheet" type="text/css" href="codehilite.css"></head><body>' && \
	  markdown_py-3 -x codehilite  -x partial_gfm -o html5 $< && \
	  echo '</body>' \
	) > $@.tmp && \
	    mv $@.tmp $@ || rm -f $@.tmp

codehilite.css:
	pygmentize -S default -f html > codehilite.css

.PHONY: binary test coverage reset-local trust-local untrust-local

