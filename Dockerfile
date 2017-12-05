FROM scratch
ADD memcached-operator /
ENTRYPOINT ["/memcached-operator"]
CMD ["-logtostderr"]
