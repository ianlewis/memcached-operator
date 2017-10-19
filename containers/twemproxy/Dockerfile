FROM alpine:3.6

RUN apk add --update --no-cache --virtual .build-deps \
        curl \
        build-base \
        autoconf \
        automake \
        libtool \
    && curl -qL https://github.com/twitter/twemproxy/archive/v0.4.1.tar.gz | tar xzf - \
    && cd twemproxy-0.4.1 \
    && autoreconf -fvi \
    && ./configure --enable-debug=log; cat config.log \
    && make \
    && mv src/nutcracker /twemproxy \
    && cd .. \
    && rm -rf nutcracker-0.4.1 \
    && apk del .build-deps \
    && echo "Build complete."

ENTRYPOINT ["/twemproxy"] 
