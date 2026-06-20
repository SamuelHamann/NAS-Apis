# syntax=docker/dockerfile:1

# --- Build stage -------------------------------------------------------------
FROM golang:1.25-alpine AS build

WORKDIR /src

# Download dependencies first so this layer is cached until go.mod/go.sum change.
COPY go.mod ./
COPY go.su[m] ./
RUN go mod download

# Build a small, static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/whatsfordinner ./cmd/server

# --- Runtime stage -----------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/whatsfordinner /app/whatsfordinner

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/whatsfordinner"]
