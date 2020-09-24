FROM golang:alpine as builder

LABEL maintainer="Ralf Geschke <ralf@kuerbis.org>"
LABEL last_changed="2020-09-24"

RUN apk update && apk add --no-cache git

# Build dynpower-cli
# Maybe this is a bit unconventional, but it works until I'll find a better way (try go get...)
RUN mkdir /build-cli && cd /build-cli && git clone https://github.com/geschke/dynpower-cli.git .
WORKDIR /build-cli
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o dynpower-cli .

# Build dynpower
RUN mkdir /build
ADD . /build/
WORKDIR /build 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o dynpower .

# Build minimal image with dynpower and dynpower-cli only
FROM scratch
COPY --from=builder /build/dynpower /app/
COPY --from=builder /build-cli/dynpower-cli /app/
ENV PATH "$PATH:/app"
WORKDIR /app
CMD ["./dynpower"]
