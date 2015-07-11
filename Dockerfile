FROM golang:1.4.2

RUN mkdir -p /go/src/github.com/samertm/syncfbevents
WORKDIR /go/src/github.com/samertm/syncfbevents

COPY . /go/src/github.com/samertm/syncfbevents

RUN ln -sf conf.prod.toml conf.toml

RUN go get -v github.com/samertm/syncfbevents

CMD ["syncfbevents"]

EXPOSE 8000
