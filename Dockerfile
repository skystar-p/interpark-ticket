# golang
from golang:1.22.5-alpine3.20 as builder
WORKDIR /app
COPY ./go.mod .
COPY ./go.sum .
COPY ./main.go .
RUN go mod download
RUN go build -o main .

# final stage
FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/main .
CMD ["./main"]

