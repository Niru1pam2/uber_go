FROM alpine
WORKDIR /app

ADD shared shared
ADD build build

ENTRYPOINT ["/app/driver-service"]