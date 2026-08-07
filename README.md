# Outbox Lab

Laboratorio para demostrar el patrón **Transactional Outbox** en pagos: cómo
registrar un pago y publicar su evento sin caer en el problema del *dual write*.

Tres actores visibles: la tabla `orders` (estado de negocio), la tabla `outbox`
(eventos `pending`/`sent`/`failed`) y un **broker falso in-process** (lo que
facturación/notificaciones realmente reciben). Un **relay** en segundo plano
drena la outbox hacia el broker. El **modo caos** permite tirar el broker y ver
que ningún evento se pierde: quedan pendientes y se drenan solos al reactivarlo.

## Requisitos

- Docker + Docker Compose
- Go 1.26+ (solo para el flujo de desarrollo con `go run`)

## Estructura

```
backend/
  cmd/server/          main: wiring de pool, broker, relay y HTTP server
  internal/
    domain/            tipos compartidos (Order, Event, OutboxRow)
    store/             acceso a datos: escritor transaccional + drenado
    broker/            broker falso in-process con flag de caída
    relay/             goroutine con time.Ticker que publica pendientes
    api/               handlers HTTP + CORS
  migrations/          migraciones golang-migrate (orders + outbox)
  Dockerfile           build multi-stage (distroless)
docs/adr/              decisiones de arquitectura
docker-compose.yml     postgres + migrate (one-shot) + backend
```

## Cómo correrlo

### Opción A — todo con Docker

Levanta PostgreSQL, aplica las migraciones y arranca el backend, en orden:

```bash
docker compose up -d
curl localhost:8080/api/state
```

El arranque respeta las dependencias: `postgres` (healthy) → `migrate`
(corre y termina) → `backend`.

### Opción B — backend en desarrollo con `go run`

Levanta solo la base y aplica migraciones; corré el servidor a mano:

```bash
docker compose up -d postgres      # solo la DB
docker compose run --rm migrate    # aplica migraciones
cd backend && go run ./cmd/server  # conecta a localhost:5432 por defecto
```

### Reset completo de la base

```bash
docker compose down -v
```

## API

Base URL: `http://localhost:8080`

| Método | Ruta          | Descripción                                                        |
|--------|---------------|--------------------------------------------------------------------|
| POST   | `/api/orders` | Registra un pago y su evento `PaymentAuthorized` en una transacción |
| GET    | `/api/state`  | Devuelve los tres actores: `orders`, `outbox` y `broker`           |
| POST   | `/api/chaos`  | Tira (`{"down":true}`) o levanta (`{"down":false}`) el broker      |

### Ejemplos

```bash
# Crear un pago
curl -X POST localhost:8080/api/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer":"alice","amount":99.90,"currency":"USD"}'

# Ver el estado (orders, outbox, broker)
curl localhost:8080/api/state

# Modo caos: tirar el broker
curl -X POST localhost:8080/api/chaos -H 'Content-Type: application/json' -d '{"down":true}'
```

`POST /api/orders` acepta `{customer, amount, currency?}` (`currency` por
defecto `USD`; `amount` debe ser `> 0`).

## Demo del modo caos

1. Tirá el broker: `POST /api/chaos {"down":true}`.
2. Creá varios pagos: sus eventos se acumulan como `pending` en la outbox
   (nada se pierde) y `broker.received_count` no cambia.
3. Levantá el broker: `POST /api/chaos {"down":false}`.
4. El relay drena solo los pendientes: la outbox pasa toda a `sent` y
   `broker.received_count` iguala la cantidad de pagos. **Invariante:** todo
   pago registrado termina notificado.

## Variables de entorno (backend)

| Variable         | Default                                                              | Descripción                     |
|------------------|----------------------------------------------------------------------|---------------------------------|
| `DATABASE_URL`   | `postgres://outbox:outbox@localhost:5432/outbox_lab?sslmode=disable` | Conexión a PostgreSQL           |
| `HTTP_ADDR`      | `:8080`                                                               | Dirección de escucha del server |
| `RELAY_INTERVAL` | `1s`                                                                  | Período del ticker del relay    |
| `RELAY_BATCH`    | `50`                                                                  | Máximo de eventos por tick      |

## Documentación

Decisiones de arquitectura: [`docs/adr/`](docs/adr/).
