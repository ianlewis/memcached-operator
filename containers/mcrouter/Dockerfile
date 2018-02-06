FROM            ubuntu:14.04

# Build-time metadata as defined at http://label-schema.org
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION
LABEL maintainer="ianmlewis@gmail.com" \
      org.label-schema.build-date=$BUILD_DATE \
	  org.label-schema.name="mcrouter" \
	  org.label-schema.description="The mcrouter memcached proxy" \
	  org.label-schema.url="https://github.com/ianlewis/memcached-operator" \
	  org.label-schema.vcs-ref=$VCS_REF \
	  org.label-schema.vcs-url="e.g. https://github.com/microscaling/microscaling" \
	  org.label-schema.version=$VERSION \
	  org.label-schema.schema-version="1.0"

ENV             MCROUTER_BRANCH         release-36-0
ENV             MCROUTER_DIR            /usr/local/mcrouter
ENV             MCROUTER_REPO           https://github.com/facebook/mcrouter.git
ENV             DEBIAN_FRONTEND         noninteractive

RUN             set -x && \
                apt-get update && \
                apt-get install -y git && \
                mkdir -p $MCROUTER_DIR/repo && \
                cd $MCROUTER_DIR/repo && git clone $MCROUTER_REPO && \
                cd $MCROUTER_DIR/repo/mcrouter/mcrouter/scripts && \
                git checkout $MCROUTER_BRANCH && \
                ./install_ubuntu_14.04.sh $MCROUTER_DIR && \
                ./clean_ubuntu_14.04.sh $MCROUTER_DIR && rm -rf $MCROUTER_DIR/repo && \
                ln -s $MCROUTER_DIR/install/bin/mcrouter /usr/local/bin/mcrouter

ENV             DEBIAN_FRONTEND newt
