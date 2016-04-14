FROM golang:alpine

COPY . /go/src/github.com/lightcode/kube2consul

RUN apk add --no-cache --virtual .build-deps \
        gcc \
        g++ \
    && rm -rf /var/cache/apk/* \
    && cd /go/src/github.com/lightcode/kube2consul \
    && go install -v \
    && mv /go/bin/kube2consul /usr/bin \
    && rm -rf /go \
    && apk del .build-deps

CMD ["/usr/bin/kube2consul"]
