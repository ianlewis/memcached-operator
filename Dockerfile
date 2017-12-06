FROM scratch

# Build-time metadata as defined at http://label-schema.org
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION
LABEL maintainer="ianmlewis@gmail.com" \
      org.label-schema.build-date=$BUILD_DATE \
	  org.label-schema.name="Memcached Operator" \
	  org.label-schema.description="A Kubernetes operator for memcached" \
	  org.label-schema.url="https://github.com/ianlewis/memcached-operator" \
	  org.label-schema.vcs-ref=$VCS_REF \
	  org.label-schema.vcs-url="e.g. https://github.com/microscaling/microscaling" \
	  org.label-schema.version=$VERSION \
	  org.label-schema.schema-version="1.0"

ADD memcached-operator /

ENTRYPOINT ["/memcached-operator"]
CMD ["-logtostderr"]
