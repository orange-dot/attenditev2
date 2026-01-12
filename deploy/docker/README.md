# Docker Deployment

Tri opcije za Docker deployment:

| Setup | Prostor | Komponente | Namena |
|-------|---------|------------|--------|
| **Minimal** | ~5 GB | PostgreSQL, KurrentDB, API | Brzi razvoj |
| **Dev** | ~15-20 GB | + Keycloak, MinIO, OPA, Monitoring | Kompletan dev |
| **Prod** | ~50+ GB | HA cluster, SSL, Traefik | Staging/Demo |

---

## Quick Start

### 1. Minimalni Setup (5 GB)

```bash
# Pokreni
docker-compose -f docker-compose.minimal.yml up -d

# Proveri status
docker-compose -f docker-compose.minimal.yml ps

# Logovi
docker-compose -f docker-compose.minimal.yml logs -f platform

# Zaustavi
docker-compose -f docker-compose.minimal.yml down
```

**Pristup:**
- API: http://localhost:8080
- KurrentDB UI: http://localhost:2113

---

### 2. Full Dev Setup (15-20 GB)

```bash
# Kreiraj konfiguracije
cp .env.example .env
# Edituj .env sa svojim vrednostima

# Pokreni sve
docker-compose -f docker-compose.dev.yml up -d

# Prati logove
docker-compose -f docker-compose.dev.yml logs -f
```

**Pristup:**
| Servis | URL | Kredencijali |
|--------|-----|--------------|
| API | http://localhost:8080 | JWT token |
| Keycloak | http://localhost:8180 | admin / admin_dev_2024 |
| Grafana | http://localhost:3000 | admin / admin_dev_2024 |
| Prometheus | http://localhost:9090 | - |
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin_dev_2024 |
| KurrentDB UI | http://localhost:2113 | - |
| Loki | http://localhost:3100 | - |

---

### 3. Production Setup (50+ GB)

```bash
# Priprema
cp .env.example .env
# OBAVEZNO: Promeni SVE šifre u .env!

# Generiši SSL sertifikate (ili koristi Let's Encrypt)
mkdir -p certs/postgres
openssl req -new -x509 -days 365 -nodes \
  -out certs/postgres/server.crt \
  -keyout certs/postgres/server.key

# Pokreni
docker-compose -f docker-compose.prod.yml up -d

# Proveri zdravlje
docker-compose -f docker-compose.prod.yml ps
curl https://api.${DOMAIN}/health
```

**Pristup (sa SSL):**
| Servis | URL |
|--------|-----|
| API | https://api.yourdomain.com |
| Auth | https://auth.yourdomain.com |
| Dashboard | https://dashboard.yourdomain.com |
| Metrics | https://metrics.yourdomain.com |
| Storage | https://storage.yourdomain.com |

---

## Struktura Fajlova

```
deploy/docker/
├── docker-compose.minimal.yml   # Minimalni setup
├── docker-compose.dev.yml       # Full dev setup
├── docker-compose.prod.yml      # Production setup
├── Dockerfile                   # Multi-stage Go build
├── .env.example                 # Template za environment
├── README.md                    # Ovaj fajl
│
├── init-scripts/                # PostgreSQL inicijalizacija
│   └── 01-init.sql
│
├── prometheus/                  # Prometheus config
│   ├── prometheus.yml
│   └── alerts.yml
│
├── grafana/                     # Grafana config
│   └── provisioning/
│       └── datasources/
│           └── datasources.yml
│
├── loki/                        # Loki config
│   └── loki.yml
│
├── kurrentdb/                   # KurrentDB config (if needed)
│   └── (empty - uses defaults)
│
├── keycloak/                    # Keycloak realm (TODO)
│   └── realm-export.json
│
└── nginx/                       # Nginx configs (prod)
    └── minio.conf
```

---

## Korisne Komande

### Logovi
```bash
# Svi servisi
docker-compose -f docker-compose.dev.yml logs -f

# Specifičan servis
docker-compose -f docker-compose.dev.yml logs -f platform

# Poslednje greške
docker-compose -f docker-compose.dev.yml logs --tail=100 platform | grep -i error
```

### Shell pristup
```bash
# PostgreSQL
docker exec -it platform-postgres psql -U platform -d platform

# API container
docker exec -it platform-api sh

# KurrentDB
docker exec -it platform-kurrentdb curl http://localhost:2113/streams
```

### Backup
```bash
# PostgreSQL dump
docker exec platform-postgres pg_dump -U platform platform > backup.sql

# MinIO
docker exec platform-minio mc mirror local/documents ./backup/documents
```

### Reset
```bash
# Zaustavi i obriši sve (OPREZ: briše podatke!)
docker-compose -f docker-compose.dev.yml down -v

# Ponovo pokreni
docker-compose -f docker-compose.dev.yml up -d
```

---

## Troubleshooting

### Container ne startuje
```bash
# Proveri logove
docker-compose -f docker-compose.dev.yml logs <service-name>

# Proveri resurse
docker stats
```

### Keycloak spor start
Keycloak može da traje 60-90 sekundi za prvi start. Prati:
```bash
docker-compose -f docker-compose.dev.yml logs -f keycloak
```

### PostgreSQL connection refused
```bash
# Proveri da li je zdrav
docker-compose -f docker-compose.dev.yml exec postgres pg_isready

# Restartuj
docker-compose -f docker-compose.dev.yml restart postgres
```

### Disk space
```bash
# Proveri Docker disk usage
docker system df

# Očisti nekorišćene resurse
docker system prune -a
```

---

## Security Notes

1. **NIKAD** ne koristi default šifre u produkciji
2. **UVEK** koristi SSL/TLS u produkciji
3. **OGRANIČI** pristup Docker socket-u
4. **BACKUP** redovno (minimum dnevno)
5. **MONITOR** logove i metrike
6. **UPDATE** slike redovno (security patches)

---

## Minimum Hardware Requirements

| Setup | CPU | RAM | Disk |
|-------|-----|-----|------|
| Minimal | 2 cores | 4 GB | 10 GB |
| Dev | 4 cores | 8 GB | 30 GB |
| Prod | 8+ cores | 16+ GB | 100+ GB |

---

## Sledeći Koraci

1. Za Kubernetes deployment: `deploy/k8s/`
2. Za CI/CD: `.gitlab-ci.yml`
3. Za monitoring dashboards: `deploy/docker/grafana/dashboards/`
