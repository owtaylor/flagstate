FROM fedora:26

ENV DISTRIBUTION_DIR /go/src/github.com/owtaylor/metastore

# https://bugzilla.redhat.com/show_bug.cgi?id=1483553
RUN ( dnf --refresh -y update glibc || true ) && \
    dnf -y update && \
    dnf -y install \
    	git \
    	golang \
	make \
   	postgresql \
      && \
    dnf clean all

WORKDIR $DISTRIBUTION_DIR
COPY . $DISTRIBUTION_DIR

RUN GOPATH=/go make binary

EXPOSE 8088
CMD ["./services/index/entrypoint.sh"]
