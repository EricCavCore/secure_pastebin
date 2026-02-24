FROM golang:1.26.0

LABEL org.opencontainers.image.authors="ecaverly@corenetwork.ca"

WORKDIR /app

COPY ./app ./

RUN go mod download

RUN go build -o spb

ENTRYPOINT [ "./spb" ]