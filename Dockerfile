FROM golang:1.13.4-alpine as build
WORKDIR /build
ENV CGO_ENABLED=0
COPY . .
RUN go mod download && \
 go build -o bapro .

FROM busybox:1.31.1
LABEL maintainer="Yusaku Matsuki <ymatzki@gmail.com>"
USER nobody
COPY --chown=nobody:nogroup --from=build /build/bapro /
ENTRYPOINT ["/bapro"]
CMD ["save", "-d", "/prometheus"]