FROM golang AS builder
RUN mkdir /build
ADD go.mod go.sum service.go /build/
WORKDIR /build
RUN export CGO_ENABLED=0 && go build

FROM alpine
COPY --from=builder /build/src /app/
WORKDIR /app
CMD ["./src"]
