FROM golang:1.22.7-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /out/tiny-llm-runner ./app/cmd/server

FROM gcr.io/distroless/static:nonroot

COPY --from=build /out/tiny-llm-runner /server

USER nonroot:nonroot
ENTRYPOINT ["/server"]
