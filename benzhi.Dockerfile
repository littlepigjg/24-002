FROM golang:1.22

WORKDIR /app

COPY go.mod ./
COPY doc.go ./
COPY cmd ./cmd/
COPY internal ./internal/
COPY pkg ./pkg/
COPY static ./static/

EXPOSE 8080

CMD ["go", "run", "./cmd/server"]
