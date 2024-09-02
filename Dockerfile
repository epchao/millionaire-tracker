FROM gocv/opencv:latest

RUN apt-get update && \
    apt-get install -y --no-install-recommends sudo liblept5 libtesseract-dev libleptonica-dev tesseract-ocr ffmpeg tesseract-ocr-eng && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/src/millionaire-tracker

RUN mkdir out/

RUN go install github.com/air-verse/air@latest

COPY . .
RUN go mod tidy

ENV HOST=0.0.0.0
ENV PORT=8080
EXPOSE ${PORT}

CMD ["air", "./api/main.go", "-b", "0.0.0.0", "--port", "8080"]