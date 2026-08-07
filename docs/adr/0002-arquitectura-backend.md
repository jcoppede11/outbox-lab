# ADR 0002: Arquitectura del backend (escritor transaccional, relay y broker)

## Estado

Aceptado

## Contexto

Sobre el esquema definido en [ADR 0001](0001-esquema-inicial.md), el backend
debe demostrar el patrón Outbox de punta a punta: escribir el pago y su evento
de forma atómica, publicar los eventos pendientes en segundo plano, y permitir
simular la caída del broker para evidenciar la garantía *at-least-once*.

## Decisión

### Stack

- **Go 1.26** con biblioteca estándar `net/http` para el router (patrones
  `METHOD /ruta` de `ServeMux`), evitando dependencias de routing.
- **`pgx/v5`** (`pgxpool`) como driver/pool de PostgreSQL.
- Única dependencia externa: `pgx`. Logging con `log/slog`.

### Layout por paquetes

`internal/{domain,store,broker,relay,api}` + `cmd/server`. El acceso a datos
vive solo en `store`; `api` no habla con la base directamente.

### Escritor transaccional (`store.CreatePayment`)

`tx.Begin` → `INSERT payments` → `INSERT outbox` (payload `PaymentAuthorized`) →
`tx.Commit`. O se persisten ambas filas o ninguna: es la garantía central del
patrón. `defer tx.Rollback` cubre cualquier salida temprana (no-op tras commit).

### Broker falso in-process (`broker.Broker`)

Implementación en memoria, thread-safe (`sync.Mutex`), que registra los eventos
recibidos y expone un flag de caída (`down`). Mientras está caído, `Publish`
devuelve `ErrDown`. Es un doble de prueba: no hay broker real (Kafka/RabbitMQ),
lo que mantiene la demo autocontenida sin perder la semántica del patrón.

### Relay como goroutine (`relay.Relay`)

Un `time.Ticker` (período `RELAY_INTERVAL`, default `1s`) dispara el drenado.
En cada tick, `store.DrainPending`:

1. Selecciona hasta `RELAY_BATCH` filas `pending` con
   `ORDER BY id FOR UPDATE SKIP LOCKED` (FIFO, apto para múltiples relays).
2. Publica cada evento; si el broker está caído, **corta** el lote.
3. Marca `sent` + `published_at` solo las publicadas con éxito, dentro de la
   misma transacción. Las no publicadas quedan `pending` y se reintentan.

Esto da entrega *at-least-once*: por eso los consumidores deben ser idempotentes.

### API HTTP (`api`)

- `POST /api/payments`: escritor transaccional.
- `GET /api/state`: expone los tres actores (payments, outbox, broker) en una
  sola respuesta, para visualizar el invariante.
- `POST /api/chaos`: alterna el flag de caída del broker.
- CORS abierto (`*`) para el frontend Angular en desarrollo.

### Ciclo de vida (`cmd/server`)

Configuración por variables de entorno; `pgxpool` con `Ping` al arranque; relay
lanzado como goroutine; apagado ordenado del HTTP server ante `SIGINT`/`SIGTERM`
vía `signal.NotifyContext` y `srv.Shutdown`.

## Consecuencias

- La atomicidad pago+evento queda garantizada por transacción SQL (ADR 0001) y
  ejercida por `CreatePayment`.
- El broker falso permite el modo caos sin infraestructura externa, pero **no**
  es persistente: los eventos recibidos se pierden al reiniciar el backend (la
  outbox en PostgreSQL sí persiste).
- `FOR UPDATE SKIP LOCKED` deja la puerta abierta a escalar a varios relays.
- El polling por ticker introduce una latencia acotada por `RELAY_INTERVAL`
  (aceptable para la demo; una versión productiva podría usar `LISTEN/NOTIFY`).
