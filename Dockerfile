FROM golang:1.17 AS builder
WORKDIR /app
COPY . .

RUN go mod download
