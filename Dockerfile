FROM node:24-bookworm-slim AS frontend
WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY angular.json tsconfig.json ./
COPY api/ api/
COPY frontend/ frontend/
RUN pnpm run generate-openapi-bundle && pnpm build

FROM golang:1.26-bookworm AS backend
RUN apt-get update && apt-get install -y --no-install-recommends libmupdf-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/internal/frontend/dist/ ./internal/frontend/dist/
ARG VERSION=snapshot
ARG COMMIT=unknown
RUN go build \
    -ldflags="-s -w -X github.com/godatei/datei/internal/buildconfig.version=${VERSION} -X github.com/godatei/datei/internal/buildconfig.commit=${COMMIT}" \
    -o dist/datei .

FROM gcr.io/distroless/cc-debian12:nonroot
WORKDIR /
COPY --from=backend /app/dist/datei /datei
USER 65532:65532
ENTRYPOINT ["/datei"]
CMD ["serve"]
