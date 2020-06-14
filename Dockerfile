FROM golang:latest
RUN mkdir /dock
ADD . /dock
WORKDIR /dock/app
RUN go build main.go
CMD ["/dock/app/main"]