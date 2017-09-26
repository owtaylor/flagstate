FROM fedora:26

# https://bugzilla.redhat.com/show_bug.cgi?id=1483553
RUN ( dnf --refresh -y update glibc || true ) && \
    dnf -y update && \
    dnf -y install \
	httpd	   \
	mod_ssl	   \
	openssl	   \
	python3    \
	python3-requests \
	python3-www-authenticate

ADD \
	check-for-data.py \
	create-test-data.sh \
	entrypoint.sh \
	generate-cert.sh \
	registry_copy.py \
    /usr/local/bin/
ADD frontend.conf /etc/httpd/conf.d

VOLUME /etc/pki/tls

EXPOSE 80
EXPOSE 443
CMD ["/usr/local/bin/entrypoint.sh"]
