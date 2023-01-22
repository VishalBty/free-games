############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git make build-base
WORKDIR /root/cloud-game
ENV PATH=$HOME/go/bin:$PATH 
ENV CGO_ENABLED=1
RUN git clone https://github.com/boypt/simple-game.git . && \
    go get -v -t -d .

RUN go build -trimpath -ldflags "-s -w -X main.VERSION=$(git describe --tags)" -o /usr/local/bin/cloud-game
############################
# STEP 2 build a small image
############################
FROM alpine
COPY --from=builder /usr/local/bin/cloud-game /usr/local/bin/cloud-game
RUN apk update && apk add ca-certificates libstdc++
ENTRYPOINT ["cloud-game"]
