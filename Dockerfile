FROM golang:1.21.6

WORKDIR /usr/src/test

RUN apt-get update && apt-get install ffmpeg -y && apt-get install libopencv-dev -y

RUN go install github.com/cosmtrek/air@latest
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .
RUN go mod tidy