# syntax=docker/dockerfile:1

# --- Build stage -------------------------------------------------------------
# Run the Go compiler on the host's native platform (fast, no emulation).
# TARGETOS / TARGETARCH are injected by `docker buildx build --platform …`
# and passed to the Go compiler so the output binary matches the target.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

# Download dependencies first so this layer is cached until go.mod/go.sum change.
COPY go.mod ./
COPY go.su[m] ./
RUN go mod download

# Cross-compile a static binary for the target platform.
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/whatsfordinner ./cmd/server

# --- Runtime stage -----------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/whatsfordinner /app/whatsfordinner

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/whatsfordinner"]
