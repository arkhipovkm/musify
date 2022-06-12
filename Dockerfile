FROM golang:1.16.7-bullseye
RUN apt -y update && apt -y install ffmpeg
WORKDIR /app
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download
COPY ./ ./
RUN go build -ldflags '-s'
CMD ./musify