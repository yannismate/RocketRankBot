FROM golang:1.21 as build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go vet -v cmd/commander/commander.go
RUN go test -v cmd/commander/commander.go
RUN CGO_ENABLED=0 go build -o /go/bin/app cmd/commander/commander.go

FROM gcr.io/distroless/static-debian12
COPY --from=build /go/bin/app /

# Expecting config file at /config.json
CMD ["/app"]