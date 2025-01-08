FROM golang:1.23.4-alpine3.20 AS builder
WORKDIR /gotinystatus
RUN apk update && apk upgrade --available && sync && apk add --no-cache --virtual .build-deps
COPY . .
RUN go build -ldflags="-w -s" .
FROM alpine:3.21.1
RUN apk update && apk upgrade --available && sync
COPY --from=builder /gotinystatus/gotinystatus /gotinystatus
COPY --from=builder /gotinystatus/incidents.html /incidents.html
COPY --from=builder /gotinystatus/checks.yaml /checks.yaml
ENTRYPOINT ["/gotinystatus"]
