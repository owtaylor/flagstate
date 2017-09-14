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

README.html: README.md codehilite.css Makefile
	( echo '<!DOCTYPE html><html><head><link rel="stylesheet" type="text/css" href="codehilite.css"></head><body>' && \
	  markdown_py-3 -x codehilite  -x partial_gfm -o html5 $< && \
	  echo '</body>' \
	) > $@.tmp && \
	    mv $@.tmp $@ || rm -f $@.tmp

codehilite.css:
	pygmentize -S default -f html > codehilite.css

.PHONY: binary test

