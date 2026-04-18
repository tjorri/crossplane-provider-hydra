FROM golang:1.25-alpine3.22 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /provider ./cmd/provider/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /provider /usr/local/bin/provider
ENTRYPOINT ["provider"]
