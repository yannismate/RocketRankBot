FROM golang:1.21 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go vet -v cmd/twitchconnector/twitchconnector.go
RUN go test -v cmd/twitchconnector/twitchconnector.go
RUN CGO_ENABLED=0 go build -o /go/bin/app cmd/twitchconnector/twitchconnector.go

FROM gcr.io/distroless/static-debian12
COPY --from=build /go/bin/app /

# Expecting config file at /config.json
CMD ["/app"]