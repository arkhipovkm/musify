FROM golang:alpine
WORKDIR /app
COPY ./ ./
RUN go build ./streamer/main.go 
CMD ./main