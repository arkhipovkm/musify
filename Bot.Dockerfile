FROM golang:alpine
WORKDIR /app
COPY ./ ./
RUN go build ./bot/main.go 
CMD ./main