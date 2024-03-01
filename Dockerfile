FROM golang:1.21 AS builder
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -o /logsock

FROM alpine:3.18
RUN apk add --no-cache tini
COPY --from=builder /logsock /logsock
ENTRYPOINT ["tini", "--"]
CMD ["/logsock"]
