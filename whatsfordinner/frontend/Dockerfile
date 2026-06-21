# syntax=docker/dockerfile:1

# --- Build stage -------------------------------------------------------------
# Run the JS build on the host's native platform (no emulation needed — the
# output is plain static files that work on any architecture). esbuild's
# postinstall fetches the right platform binary for *this* (BUILDPLATFORM)
# image, which is what we need for the build, not the target.
FROM --platform=$BUILDPLATFORM node:24-alpine AS build

WORKDIR /src

# Install dependencies first so this layer is cached until the lockfile changes.
# We disable install scripts globally (npm 11+ is strict by default) and then
# explicitly rebuild esbuild so it downloads its native binary.
COPY package.json package-lock.json ./
RUN npm ci --include=dev --ignore-scripts \
    && npm rebuild esbuild

# Build the minified SPA bundle into public/dist/app.js.
COPY tsconfig.json ./
COPY src ./src
COPY public ./public
RUN npm run build

# --- Runtime stage -----------------------------------------------------------
# nginx-unprivileged listens on 8080 by default so we can run as a non-root
# user — same posture as the API image (which uses distroless nonroot).
FROM nginxinc/nginx-unprivileged:1.27-alpine

# Static site root.
COPY --from=build /src/public /usr/share/nginx/html

# Replace the default site with the SPA + /api proxy config.
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080
