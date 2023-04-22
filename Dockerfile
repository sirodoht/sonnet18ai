FROM golang:1.20

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o sonnet18ai cmd/server/main.go

CMD ["./sonnet18ai"]
