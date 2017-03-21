FROM scratch

ADD events /events

ENV DOCKER_ENDPOINT unix:///var/run/docker.sock
ENV ETCD_ENDPOINT http://etcd1.isd.ictu:4001
ENV ETCD_BASEKEY /skydns/

CMD ["/events"]

