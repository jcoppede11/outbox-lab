# Outbox Lab

[![CI](https://github.com/jcoppede11/outbox-lab/actions/workflows/ci.yml/badge.svg)](https://github.com/jcoppede11/outbox-lab/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Laboratorio para demostrar el patrón **Transactional Outbox** en pagos: cómo
registrar un pago y publicar su evento sin caer en el problema del *dual write*.

**Dual write:** escribir en la base de datos y en el broker como dos pasos
separados, sin transacción compartida. Si uno falla y el otro no, quedás
inconsistente (pago sin evento, o evento sin pago). El outbox evita eso
guardando el evento en la **misma transacción SQL** que el pago; un relay lo
publica después, de forma asíncrona y reintentable.

```mermaid
flowchart LR
    Cliente -->|POST /api/payments| API
    subgraph TX["1 transacción SQL (atómica)"]
        Payments[("payments")]
        Outbox[("outbox<br/>pending")]
    end
    API --> Payments
    API --> Outbox
    Relay["Relay<br/>(goroutine + ticker)"] -->|lee pending| Outbox
    Relay -->|publica| Broker["Broker falso<br/>(facturación / notificaciones)"]
    Relay -->|marca sent| Outbox
    Broker -. "modo caos: caído → publish falla,<br/>las filas siguen pending" .-> Relay
```

Tres actores visibles:

- **`payments`** — el estado de negocio (el pago registrado).
- **`outbox`** — los eventos a publicar (`pending` / `sent` / `failed`).
- **broker falso in-process** — lo que facturación y notificaciones realmente
  reciben. Un **relay** en segundo plano drena la outbox hacia el broker.

El **modo caos** permite tirar el broker y ver que ningún evento se pierde:
quedan `pending` y se drenan solos al reactivarlo.

## Quick start

```bash
git clone https://github.com/jcoppede11/outbox-lab.git
cd outbox-lab
docker compose up -d --build
```

Abrí **http://localhost:4200** y probá el patrón:

1. **Activá el modo caos** (tirá el broker).
2. **Creá un pago** → su evento queda en la outbox como `pending` (no se pierde).
3. **Desactivá el caos** → el relay drena el evento y pasa a `sent`.

Para correr en otro puerto: `FRONTEND_PORT=4300 docker compose up -d --build`.
Para parar todo: `docker compose down` (agregá `-v` para resetear la base).

## Requisitos

- Docker (única dependencia para levantar todo).
- Go 1.26+ y Node + **pnpm** solo para el flujo de desarrollo (ver más abajo).

## API

Base URL: `http://localhost:8080`

| Método | Ruta            | Descripción                                                         |
|--------|-----------------|---------------------------------------------------------------------|
| POST   | `/api/payments` | Registra un pago y su evento `PaymentAuthorized` en una transacción |
| GET    | `/api/state`    | Devuelve los tres actores: `payments`, `outbox` y `broker`          |
| POST   | `/api/chaos`    | Tira (`{"down":true}`) o levanta (`{"down":false}`) el broker       |

```bash
# Crear un pago (currency por defecto USD; amount debe ser > 0)
curl -X POST localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -d '{"customer":"alice","amount":99.90,"currency":"USD"}'

# Ver el estado (payments, outbox, broker)
curl localhost:8080/api/state

# Modo caos: tirar el broker
curl -X POST localhost:8080/api/chaos -H 'Content-Type: application/json' -d '{"down":true}'
```

## Desarrollo (backend con `go run`)

```bash
docker compose up -d postgres      # solo la DB
docker compose run --rm migrate    # aplica migraciones
cd backend && cp .env.example .env # ajustá si hace falta
go run ./cmd/server
```

| Variable         | Default                                                              | Descripción                     |
|------------------|---------------------------------------------------------------------|---------------------------------|
| `DATABASE_URL`   | `postgres://outbox:outbox@localhost:5432/outbox_lab?sslmode=disable` | Conexión a PostgreSQL           |
| `HTTP_ADDR`      | `:8080`                                                              | Dirección de escucha del server |
| `RELAY_INTERVAL` | `1s`                                                                 | Período del ticker del relay    |
| `RELAY_BATCH`    | `50`                                                                 | Máximo de eventos por tick      |

> El Postgres de Docker se publica en el host en el puerto **5433** (el 5432 lo
> suele ocupar un Postgres nativo). Para `go run` contra la DB de Docker, apuntá
> `DATABASE_URL` a `localhost:5433`. Ver [`docs/entorno-docker.md`](docs/entorno-docker.md).

> ⚠️ **Credenciales de demo.** El par `outbox`/`outbox` es **solo para
> desarrollo local**. En producción usá una password fuerte inyectada por un
> gestor de secretos.

## Tests

El backend incluye tests de integración que reproducen la demo del modo caos
contra un **PostgreSQL real**, levantado con
[testcontainers-go](https://golang.testcontainers.org/). Verifican el invariante
del patrón: con el broker caído nada se pierde, y al reactivarlo **todo pago
registrado termina notificado** exactamente una vez.

Requieren un daemon de Docker en ejecución y se corren con el build tag
`integration`:

```bash
cd backend
go test -tags integration -race ./...
```

Sin el tag, `go test ./...` no ejecuta nada que dependa de Docker. En CI se
corren en cada push y pull request, junto con `gofmt`, `go vet`,
`golangci-lint` y el build de las imágenes Docker.

## Documentación

- Decisiones de arquitectura: [`docs/adr/`](docs/adr/).
- Entorno Docker: [`docs/entorno-docker.md`](docs/entorno-docker.md).

## Licencia

[MIT](LICENSE).
