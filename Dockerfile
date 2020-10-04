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
RUN mkdir lib && ldd mtr | awk '{ print $1 }' | xargs -i cp {} lib/

FROM mtr AS upx
COPY --from=compile /src/looking-glass /busybox/looking-glass
COPY --from=busybox:uclibc /bin/ping /busybox/ping
COPY --from=mtr /mtr-0.94/mtr /busybox/mtr
COPY --from=mtr /mtr-0.94/mtr-packet /busybox/mtr-packet
WORKDIR /busybox
RUN apk add upx && upx -9 * || true
RUN ln -s ping traceroute
RUN chmod 4555 /busybox/ping /busybox/mtr-packet
RUN echo -e 'root:x:0:0:root:/root:/sbin/nologin\nnobody:x:65534:65534:nobody:/:/sbin/nologin' > /etc/passwd

FROM scratch
COPY --from=upx /etc/passwd /etc/passwd
COPY --from=upx /busybox /bin
COPY --from=upx /mtr-0.94/lib /lib
USER nobody
ENV PATH=/bin
ENTRYPOINT ["/bin/looking-glass"]