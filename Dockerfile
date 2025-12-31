FROM golang:latest as builder

WORKDIR /go

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o fx .

FROM alpine

COPY --from=builder /go/fx /bin/fx

WORKDIR /data

ENV COLORTERM=truecolor

ENTRYPOINT ["/bin/fx"]
