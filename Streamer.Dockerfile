FROM golang:alpine
RUN apk update && apk add ffmpeg
WORKDIR /app
COPY ./ ./
RUN go mod download
RUN go build ./streamer/main.go 
CMD ./main