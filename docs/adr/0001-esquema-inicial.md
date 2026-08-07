# ADR 0001: Esquema inicial de base de datos

## Estado

Aceptado

## Contexto

PayOutbox demuestra el patrón Transactional Outbox para pagos. La base de datos debe almacenar el estado de negocio (órdenes/pagos) y los eventos pendientes de publicar hacia el broker, garantizando atomicidad entre ambos.

## Decisión

### Extensión `pgcrypto`

Se declara `CREATE EXTENSION IF NOT EXISTS pgcrypto` para habilitar `gen_random_uuid()`. Aunque está disponible en el core de PostgreSQL desde la versión 13, la extensión se declara explícitamente por compatibilidad y claridad de dependencias.

### Tabla `orders`

Almacena el estado de negocio de cada orden/pago: cliente, monto, moneda y estado (`created`, `authorized`, `failed`).

### Tabla `outbox`

Almacena eventos pendientes de publicar hacia el broker. Se escribe en la misma transacción que `orders` para garantizar atomicidad: o se persisten ambos o ninguno.

Columnas relevantes:

- `payload` (JSONB): contenido del evento.
- `status`: `pending`, `sent` o `failed`.
- `published_at`: marca temporal de publicación exitosa.

### Índice `idx_outbox_pending`

Índice parcial sobre `outbox(id)` filtrado por `status = 'pending'`. Solo indexa filas pendientes, ordenadas por `id` (FIFO), optimizando el polling del relay:

```sql
SELECT ... WHERE status = 'pending' ORDER BY id FOR UPDATE SKIP LOCKED LIMIT N
```

### Migración down

La migración de rollback elimina las tablas `outbox` y `orders` pero no dropea la extensión `pgcrypto`, ya que puede ser utilizada por otros objetos del esquema.

## Consecuencias

- El relay puede hacer polling eficiente de eventos pendientes sin escanear filas ya enviadas.
- La atomicidad entre orden y evento queda garantizada a nivel de transacción SQL.
- Los consumidores del broker deben ser idempotentes (entrega at-least-once).
