ARG GO_VERSION=1.27.1
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY app ./app
RUN CGO_ENABLED=1 go build -trimpath -o /forum ./app

FROM debian:bookworm-slim
WORKDIR /srv/forum
COPY --from=build /forum /usr/local/bin/forum
COPY templates ./templates
COPY css ./css
RUN mkdir /data && chown 65532:65532 /data
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/forum"]
CMD ["-root","/srv/forum","-db","/data/forum.db","-listen","0.0.0.0:8080"]
