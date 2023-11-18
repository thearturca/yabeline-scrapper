FROM golang:1.21-alpine as build
WORKDIR /app
COPY go.mod go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dist/app .

FROM alpine:latest AS bot
RUN apk upgrade -U \ 
    && apk add ca-certificates ffmpeg \
    && rm -rf /var/cache/*
COPY --from=build /app/dist/app ./
ENTRYPOINT ./app
