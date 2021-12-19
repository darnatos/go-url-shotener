FROM golang:alpine as builder

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o shortener .

WORKDIR /dist

RUN cp /build/shortener .

FROM scratch

COPY --from=builder /dist/shortener /app/

COPY configuration.json /app/

WORKDIR /app

CMD ["./shortener"]