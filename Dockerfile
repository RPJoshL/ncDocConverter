# Build go binary
FROM docker.io/golang:1.20-alpine3.17 AS builder

ARG VERSION=0.0.0
WORKDIR /build

# To optimize the cache only copy and install the dependencies inside of the file "go.sum"
COPY go.sum go.mod ./
RUN go mod download

# Copy now all files
COPY . .

# Build the binary
RUN GOOS=linux GOARCH=amd64 go build -o ncDocConverth -ldflags "-X main.version=${VERSION}" ./cmd/ncDocConverth


# Image to run the binary
FROM docker.io/alpine:3.18

COPY --from=builder --chmod=0777 /build/ncDocConverth /app/ncDocConverth

CMD [ "/app/ncDocConverth", "--config", "/config/config.yaml" ] 