FROM gocv/opencv:latest

RUN apt-get clean && \
    apt-get -y update && \
    apt-get -y upgrade && \
    apt-get install -y sudo liblept5 libtesseract-dev libleptonica-dev tesseract-ocr && \
    apt-get install -y ffmpeg && \
    apt-get install -y tesseract-ocr-eng

WORKDIR /usr/src/millionaire-tracker

RUN mkdir out/

RUN go install github.com/air-verse/air@latest

COPY . .
RUN go mod tidy

ENV HOST=0.0.0.0
ENV PORT=8080
EXPOSE ${PORT}
EXPOSE ${PORT}/udp
EXPOSE ${PORT}/tcp

CMD ["air", "./api/main.go", "-b", "0.0.0.0", "--port", "8080"]