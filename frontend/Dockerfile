# syntax=docker/dockerfile:1

# --- 1. Compilación ---
FROM node:22-alpine AS build
WORKDIR /app

# Usar la versión de pnpm fijada en package.json ("packageManager").
RUN corepack enable

# Instalar dependencias primero para cachear. 
#pnpm-workspace.yaml aplica la configuración requerida del store.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

COPY . .
RUN pnpm build

# --- 2. Ejecución ---
FROM nginx:1.27-alpine
COPY --from=build /app/dist/frontend/browser /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
