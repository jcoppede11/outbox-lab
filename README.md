# Outbox Lab

Laboratorio para demostrar el patrón Transactional Outbox en pagos.

## Requisitos

- Docker
- Docker Compose
- Go 1.26+

## Base de datos

Levantar PostgreSQL:

```bash
docker compose up -d postgres
```

Aplicar migraciones:

```bash
docker compose run --rm migrate
```

Detener servicios y eliminar volúmenes (reset completo de la DB):

```bash
docker compose down -v
```

## API

Levantar el servidor HTTP (requiere PostgreSQL en marcha y migraciones aplicadas):

```bash
cd backend && go run ./cmd/server    # go.exe
```

## Documentación

Decisiones de arquitectura: [`docs/adr/`](docs/adr/).
