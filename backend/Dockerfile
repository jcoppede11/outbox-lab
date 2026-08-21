# syntax=docker/dockerfile:1

# --- 1. Compilación ---
FROM golang:1.26 AS build
WORKDIR /src

# Cachear módulos primero: solo se vuelven a descargar si cambian go.mod/go.sum.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# --- 2. Ejecución ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
