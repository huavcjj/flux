FROM golang:1.24-alpine
ARG AIR_VERSION=v1.61.7

RUN apk add --no-cache curl && \
    go install github.com/air-verse/air@${AIR_VERSION}

WORKDIR /app


COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
