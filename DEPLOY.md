# Docker Deployment

This project already ships a multi-stage `Dockerfile` that builds the Vue frontend, embeds it into the Go backend, and runs the compiled `sub2api` binary on Alpine. The default `docker-compose.yml` in the repository root runs only the application container; PostgreSQL and Redis must be reachable external services.

## 1. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` and point these variables at your external PostgreSQL and Redis:

```env
DATABASE_HOST=your-postgres-host
DATABASE_PORT=5432
DATABASE_USER=sub2api
DATABASE_PASSWORD=your-postgres-password
DATABASE_DBNAME=sub2api

REDIS_HOST=your-redis-host
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
```

Use the existing variable names from the application: `DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_USER`, `DATABASE_PASSWORD`, `DATABASE_DBNAME`, and `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`. Do not rename them to `DB_HOST` style variables unless the application code is changed.

For production, set stable secrets:

```bash
openssl rand -hex 32
```

Use generated values for `JWT_SECRET` and `TOTP_ENCRYPTION_KEY`.

## 2. Build Image

```bash
docker build -t sub2api:latest .
```

## 3. Start

```bash
docker compose up -d
```

View logs:

```bash
docker compose logs -f app
```

The app is published at `http://<server-ip>:${SERVER_PORT}`. The container always listens on port `8080`; change the host port with `SERVER_PORT` in `.env`.

## External PostgreSQL / Redis Connectivity

If PostgreSQL or Redis runs directly on the same Docker host, either use the host LAN/private IP or keep `host.docker.internal` in `.env`. The compose file includes:

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

On Linux this maps `host.docker.internal` to the Docker host gateway. PostgreSQL and Redis must listen on a reachable interface, not only `127.0.0.1`, and the host firewall or cloud security group must allow traffic from the container network.

If PostgreSQL and Redis are managed cloud services or separate servers, use their private/VPC endpoint addresses and allow access from the Docker host.

## Stop / Upgrade

```bash
docker compose up -d --build
```

To stop the app:

```bash
docker compose down
```
