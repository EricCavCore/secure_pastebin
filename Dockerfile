FROM golang:1.26.0 AS builder

LABEL org.opencontainers.image.authors="ecaverly@corenetwork.ca"

WORKDIR /app

COPY ./app ./

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o spb

FROM alpine:3.21

RUN apk add --no-cache ca-certificates curl \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/spb .
COPY --from=builder /app/www ./www

USER appuser

EXPOSE 8080

ENTRYPOINT ["./spb"]
