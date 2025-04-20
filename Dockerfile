FROM golang:1.24 AS base

WORKDIR /shm

COPY go.mod go.sum ./
RUN go mod download

COPY /cmd /shm/cmd
COPY /internal /shm/internal

FROM base AS alert
RUN go build -v -o alert cmd/alert/main.go
CMD ["./alert"]

FROM base AS checker
RUN go build -v -o checker cmd/checker/main.go
CMD ["./checker"]

FROM base AS scheduler
RUN go build -v -o scheduler cmd/scheduler/main.go
CMD ["./scheduler"]

FROM base AS server
RUN go build -v -o server cmd/server/main.go
CMD ["./server"]

FROM base AS tgbot
RUN go build -v -o tgbot cmd/tgbot/main.go
CMD ["./tgbot"]
