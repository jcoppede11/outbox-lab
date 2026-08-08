# Guía: entorno Docker (puertos, red interna y contenedor de migraciones)

Guía práctica de cómo está armado el `docker-compose.yml` de este proyecto y por
qué. Pensada para entender el entorno y para enseñar tres conceptos que suelen
confundir: **mapeo de puertos**, **red interna de Docker** y el patrón de
**contenedor one-shot** para migraciones.

## Contenedores del proyecto

El compose define **3 contenedores**:

| Contenedor | Rol | Ciclo de vida |
|---|---|---|
| `outbox-lab-postgres` | Base de datos PostgreSQL | queda corriendo |
| `outbox-lab-backend` | API (Go) | queda corriendo |
| `outbox-lab-migrate-1` | Aplica migraciones y termina | one-shot (corre y sale) |

Por eso, tras `docker compose up -d`, ves **2 servicios `Up`** y `migrate` en
estado `Exited (0)`: no es un error, es su comportamiento esperado.

---

## 1. Mapeo de puertos: `HOST:CONTENEDOR`

En este proyecto Postgres se publica así:

```yaml
ports:
  - "5433:5432"
```

La sintaxis es `HOST:CONTENEDOR`:

```
"5433:5432"
   │     │
   │     └─ puerto DENTRO del contenedor (donde Postgres realmente escucha)
   └─────── puerto en TU MÁQUINA (el host)
```

Puntos clave:

- Postgres **siempre escucha en 5432 dentro de su contenedor**. Eso no se cambia
  (requeriría reconfigurar Postgres). Lo único que elegimos es por qué puerto del
  host se accede a él.
- El mapeo es un "reenvío" que Docker crea **solo para el host**.

### Por qué 5433 y no 5432

El host (tu Mac) ya tiene un **Postgres nativo ocupando el 5432**, usado por otras
DBs locales. Publicar el Postgres de Docker también en 5432 da:

```
Error response from daemon: ports are not available:
listen tcp 0.0.0.0:5432: bind: address already in use
```

Mapeándolo a `5433:5432`, ambos Postgres conviven sin tocar el nativo:

| Postgres | Puerto host | Uso |
|---|---|---|
| Máquina (nativo) | `5432` | otras DBs locales |
| Docker (outbox-lab) | `5433` | este proyecto |

### Conexión desde tu máquina

Desde fuera de Docker (psql, un cliente GUI, etc.) se usa el puerto publicado, **5433**:

```bash
psql postgres://outbox:outbox@localhost:5433/outbox_lab
```

---

## 2. Red interna de Docker: por qué el backend usa `postgres:5432`

Hay **dos caminos** para llegar a la base, y usan puertos distintos.

**A. Desde el host (fuera de Docker)** → por el puerto publicado, **5433**:

```
Tu Mac  ──localhost:5433──►  [ mapeo ]  ──►  contenedor postgres:5432
```

**B. Entre contenedores (backend → postgres)** → por la **red interna de Docker**,
usando el nombre del servicio y el puerto **real del contenedor**, **5432**:

```
contenedor backend  ──postgres:5432──►  contenedor postgres
```

El backend **no pasa por el mapeo del host**: va directo por la red privada que
Compose crea, donde el mapeo `5433:5432` ni existe. Por eso su `DATABASE_URL`
apunta a `postgres:5432` y **no cambia** aunque el puerto del host sea 5433:

```yaml
backend:
  environment:
    DATABASE_URL: postgres://outbox:outbox@postgres:5432/outbox_lab?sslmode=disable
```

> `postgres` (el hostname) es el nombre del servicio en el compose. Docker resuelve
> ese nombre por DNS interno a la IP del contenedor.

### Analogía

Un edificio de oficinas:

- Dentro del edificio, la oficina de Postgres es la **puerta 5432**. Los demás
  contenedores (backend) están en el mismo edificio y golpean esa puerta directo.
- El **5433** es el número de calle de la entrada, para que la gente de afuera
  (tu Mac) pueda entrar. Cambiar el número de la calle no cambia el número de la
  oficina adentro.

**Regla mnemotécnica:** el mapeo de puertos es solo para el host. Contenedor a
contenedor siempre se hablan por el puerto interno.

---

## 3. `migrate` como contenedor one-shot

`migrate` usa la imagen oficial `migrate/migrate` y corre una sola vez: aplica las
migraciones y termina.

```yaml
migrate:
  image: migrate/migrate:v4.18.1
  depends_on:
    postgres:
      condition: service_healthy
  restart: "no"
  volumes:
    - ./backend/migrations:/migrations:ro
  command:
    - "-path=/migrations"
    - "-database=postgres://outbox:outbox@postgres:5432/outbox_lab?sslmode=disable"
    - "up"
```

### Ventajas del patrón

1. **Separación de responsabilidades.** El backend solo sirve la API; no mezcla
   "migrar el esquema" con "atender requests". Son trabajos con ciclos de vida
   distintos (uno termina, el otro vive).

2. **Mismo entorno, sin instalar nada local.** La versión de `migrate` queda fija
   en la imagen (`v4.18.1`). Nadie tiene que instalar la herramienta en su máquina
   ni pelearse con versiones: todos corren exactamente lo mismo.

3. **Orden garantizado con `depends_on`.** La cadena es:

   ```
   postgres (healthy) ──► migrate (termina OK) ──► backend arranca
   ```

   ```yaml
   backend:
     depends_on:
       migrate:
         condition: service_completed_successfully
   ```

   El backend **no arranca hasta que las migraciones terminaron bien**. Si una
   migración falla, `migrate` sale con error y el backend directamente no levanta.
   Se evita el bug clásico de un backend corriendo contra un esquema a medias.

4. **Se puede correr solo**, sin levantar el resto:

   ```bash
   docker compose run --rm migrate
   ```

   Útil para probar una migración nueva, o como paso independiente en CI/CD.

5. **No queda colgado ocupando recursos.** Al ser one-shot, cuando termina queda
   en `Exited (0)`. Su único propósito es ejecutarse una vez por arranque.

### ¿Por qué no migrar dentro del backend al arrancar?

Es una alternativa válida y común, pero con desventajas:

- Con **varias réplicas** del backend, todas intentarían migrar a la vez →
  condiciones de carrera.
- Mezcla dos responsabilidades en el mismo proceso.
- Cuesta más correr "solo la migración" sin levantar la app.

**Trade-off honesto:** para un lab chico, migrar dentro del backend también
funcionaría y sería una pieza menos. El contenedor separado es la práctica que
mejor escala hacia producción — que es justo lo que este lab busca enseñar.

---

## Referencia rápida

```bash
# Levantar todo (postgres → migrate → backend)
docker compose up -d

# Ver estado y puertos
docker compose ps

# Conectarse a la DB desde el host
psql postgres://outbox:outbox@localhost:5433/outbox_lab

# Aplicar migraciones a mano (sin levantar el backend)
docker compose run --rm migrate
```

| Concepto | Valor |
|---|---|
| Puerto Postgres en el host | `5433` |
| Puerto Postgres dentro de la red Docker | `5432` |
| Host que usa el backend | `postgres` (nombre del servicio) |
