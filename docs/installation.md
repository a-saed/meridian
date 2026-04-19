# Installation

## Option A — Go binary (local / server)

**Requirements:** Go 1.21+

```bash
git clone https://github.com/your-org/meridian.git
cd meridian
go run ./cmd/server          # development
go build -o meridian ./cmd/server && ./meridian   # production binary
```

The binary is self-contained (~23 MB). No runtime dependencies.

---

## Option B — Docker Compose (with PostGIS)

Starts Meridian + PostGIS 15 together:

```bash
docker compose up --build -d
```

- Meridian: `http://localhost:8080`
- PostGIS: `localhost:5432` (user: `gis`, password: `gis`, db: `gis`)

```bash
docker compose down       # stop
docker compose down -v    # stop and delete all data
```

**Mounting your GeoJSON files:**

Add a volume to the meridian service in `docker-compose.yml`:

```yaml
services:
  meridian:
    volumes:
      - ./my_data.geojson:/data/my_data.geojson
      - meridian_data:/data/db
    environment:
      DB_PATH: /data/db/meridian.db
```

Then register the source using the container path `/data/my_data.geojson`.

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | Listen address (e.g. `:9000` or `0.0.0.0:8080`) |
| `DB_PATH` | `meridian.db` | Path to the SQLite config database |
| `S3_ENDPOINT` | _(empty)_ | Custom S3-compatible API endpoint (e.g. `http://minio:9000`) |
| `S3_REGION` | `us-east-1` | Region string used in SigV4 signing |
| `S3_BUCKET` | _(empty)_ | When set **with** `S3_ACCESS_KEY` and `S3_SECRET_KEY`, GeoJSON uploads go to object storage instead of local disk |
| `S3_ACCESS_KEY` | _(empty)_ | Access key id |
| `S3_SECRET_KEY` | _(empty)_ | Secret access key |
| `S3_KEY_PREFIX` | _(empty)_ | Optional prefix prepended to every object key |
| `S3_USE_PATH_STYLE` | `0` | Set to `1` or `true` to force path-style URLs (also implied when `S3_ENDPOINT` is set) |

Logs are stored in-process in a ring buffer and exposed at `GET /api/v1/logs`. They are also visible in the UI under the **Logs** tab. No external infrastructure is required.

---

## Production deployment

### Reverse proxy with nginx

Run Meridian on a non-public port and proxy through nginx for TLS:

```nginx
server {
    listen 443 ssl;
    server_name maps.example.com;

    ssl_certificate     /etc/letsencrypt/live/maps.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/maps.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Health check

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

Use this as your load balancer or container orchestrator health check endpoint.

### Persistent data

Meridian stores all layer/source/style config in a SQLite file (`meridian.db` by default). Back this file up regularly. In Docker, always use a named volume so data survives container restarts:

```yaml
volumes:
  meridian_data:
```

### PostGIS note

PostGIS sources are loaded at startup. After registering a new PostGIS source via the API, restart Meridian to load it:

```bash
docker compose restart meridian
# or, for the binary:
systemctl restart meridian
```
