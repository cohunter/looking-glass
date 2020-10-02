FROM golang:alpine AS compile
ADD . /src
WORKDIR /src
RUN go build

FROM alpine:latest
COPY --from=compile /src/looking-glass /looking-glass
RUN apk add iputils mtr
ENTRYPOINT ["/looking-glass"]