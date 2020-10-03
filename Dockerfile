FROM golang:latest AS compile
ADD . /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -ldflags "-s -w"

FROM alpine:latest AS mtr
RUN apk add wget make gcc libc-dev
RUN wget https://www.bitwizard.nl/mtr/files/mtr-0.94.tar.gz
RUN tar xf mtr-0.94.tar.gz
WORKDIR /mtr-0.94
RUN LDFLAGS="-static" ./configure --disable-dependency-tracking
RUN make

FROM mtr AS upx
RUN apk add upx
COPY --from=compile /src/looking-glass /busybox/looking-glass
COPY --from=busybox:uclibc /bin/ping /busybox/ping
COPY --from=mtr /mtr-0.94/mtr /busybox/mtr
COPY --from=mtr /mtr-0.94/mtr-packet /busybox/mtr-packet
WORKDIR /busybox
RUN upx -9 *
RUN ln -s ping traceroute
RUN chmod 4555 /busybox/ping
RUN echo -e 'root:x:0:0:root:/root:/sbin/nologin\nnobody:x:65534:65534:nobody:/:/sbin/nologin' > /etc/passwd

FROM scratch
COPY --from=upx /etc/passwd /etc/passwd
COPY --from=upx /busybox /bin
COPY --from=upx /lib/ld-musl-x86_64.so.1 /lib/ld-musl-x86_64.so.1
USER nobody
ENV PATH=/bin
ENTRYPOINT ["/bin/looking-glass"]