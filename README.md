# Outbox Lab

[![CI](https://github.com/jcoppede11/outbox-lab/actions/workflows/ci.yml/badge.svg)](https://github.com/jcoppede11/outbox-lab/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Laboratorio para demostrar el patrón **Transactional Outbox** en pagos *dual write*.

**Dual write:** escribir en la base de datos y en el broker como si fueran dos pasos
separados, sin transacción compartida. Si uno falla y el otro no, queda
inconsistencia (pago sin evento o evento sin pago). Outbox evita eso
guardando el evento en la misma transacción SQL que el pago; el relay publica
después, de manera asíncrona y reintentable.

Actores: 
* `payments` (estado de negocio).
* `outbox` (eventos `pending`/`sent`/`failed`)
* **broker falso in-process** (lo que acturación/notificaciones realmente reciben)
* **relay** en segundo plano drena la outbox hacia el broker. El modo caos permite tirar el broker y ver que ningún evento se pierde: quedan pendientes y se drenan solos al reactivarlo.

## Quick start

```bash
git clone <URL-del-repo>
cd outbox-lab
docker compose up -d --build
```

Abrí **http://localhost:4200** o el puerto que seteaste y probá el patrón:

1. **Activá el modo caos** (tirá el broker).
2. **Creá un pago** → su evento queda en la outbox como `pending` (no se pierde).
3. **Desactivá el caos** → el relay drena el evento y pasa a `sent`.

Para correr en otro puerto: `FRONTEND_PORT=4300 docker compose up -d --build`.
Para parar todo: `docker compose down` (agregá `-v` para resetear la base).

## Requisitos

- Docker
- Go 1.26+ (solo para el flujo de desarrollo con `go run`)
- Node + **pnpm** (solo para el frontend; su uso está forzado vía `only-allow`)

## Tests

El backend incluye tests de integración que reproducen la demo del modo caos
contra un PostgreSQL real, levantado con
[testcontainers-go](https://golang.testcontainers.org/).

## Documentación

- Decisiones de arquitectura: [`docs/adr/`](docs/adr/).

## Licencia

[MIT](LICENSE).
