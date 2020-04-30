FROM golang:alpine
WORKDIR /app
COPY ./ ./
RUN go mod download
RUN go build ./bot/main.go 
CMD ./main