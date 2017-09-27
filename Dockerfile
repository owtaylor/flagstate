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
	\
	$(: skopeo build dependencies ) \
    	btrfs-progs-devel \
        device-mapper-devel \
        glib2-devel \
        gpgme-devel \
        libassuan-devel \
        ostree-devel \
	&& \
    dnf clean all

RUN GOPATH=/go go get github.com/estesp/manifest-tool
RUN set -x && git clone https://github.com/projectatomic/skopeo /go/src/github.com/projectatomic/skopeo && \
    cd /go/src/github.com/projectatomic/skopeo && \
    GOPATH=/go make binary-local && install -m 0755 ./skopeo /go/bin

WORKDIR $DISTRIBUTION_DIR
COPY . $DISTRIBUTION_DIR

RUN GOPATH=/go make binary

EXPOSE 8088
CMD ["./services/index/entrypoint.sh"]
