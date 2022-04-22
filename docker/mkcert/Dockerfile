# cribbed from https://github.com/brunopadz/mkcert-docker/blob/master/Dockerfile

FROM golang:1.12-alpine

RUN apk --no-cache add git

RUN go get -u github.com/FiloSottile/mkcert

WORKDIR /root

ADD entrypoint.sh .

VOLUME [ "/root/certs" ]
VOLUME [ "/root/caroot" ]

ENTRYPOINT [ "sh", "entrypoint.sh" ]