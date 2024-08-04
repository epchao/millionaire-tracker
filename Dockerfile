FROM gocv/opencv:latest

RUN apt-get clean && \
    apt-get -y update && \
    apt-get -y upgrade && \
    apt-get install -y sudo liblept5 libtesseract-dev libleptonica-dev tesseract-ocr

RUN sudo apt-get install -y ffmpeg

RUN apt-get install -y tesseract-ocr-eng

WORKDIR /usr/src/millionaire-tracker
RUN mkdir out/

RUN go install github.com/air-verse/air@latest

COPY . .
RUN go mod tidy

ENV HOST 0.0.0.0

EXPOSE 3000