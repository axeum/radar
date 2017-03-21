FROM alpine:3.5

ADD radar /radar

ENV DOCKER_ENDPOINT unix:///var/run/docker.sock
ENV ETCD_ENDPOINT http://etcd1.isd.ictu:4001
ENV ETCD_BASEKEY /skydns

ENTRYPOINT ["/radar"]
