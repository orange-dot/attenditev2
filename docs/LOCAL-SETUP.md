# Lokalna instalacija i pokretanje

Ovaj dokument opisuje kako postaviti i pokrenuti Serbia Government Interoperability Platform lokalno za razvoj.

## Sadržaj

- [Preduslovi](#preduslovi)
- [Kloniranje repozitorijuma](#kloniranje-repozitorijuma)
- [Brza instalacija (AI Demo)](#brza-instalacija-ai-demo)
- [Ručna instalacija](#ručna-instalacija)
- [Pokretanje servisa](#pokretanje-servisa)
- [Pristup aplikaciji](#pristup-aplikaciji)
- [Razvoj](#razvoj)
- [Testiranje](#testiranje)
- [Troubleshooting](#troubleshooting)

---

## Preduslovi

### Obavezno

| Alat | Verzija | Provera instalacije |
|------|---------|---------------------|
| **Go** | 1.22+ | `go version` |
| **Docker** | 20.10+ | `docker --version` |
| **Docker Compose** | 2.0+ | `docker compose version` |
| **Node.js** | 18+ | `node --version` |
| **npm** | 9+ | `npm --version` |
| **Git** | 2.0+ | `git --version` |

### Opciono (za razvoj)

| Alat | Svrha | Instalacija |
|------|-------|-------------|
| **Make** | Build automatizacija | `choco install make` (Windows) / dolazi sa OS (Linux/Mac) |
| **Air** | Hot reload za Go | `go install github.com/air-verse/air@latest` |
| **golangci-lint** | Linting | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |

### Sistemski zahtevi

- **RAM**: Minimum 8GB (preporučeno 16GB)
- **Disk**: ~5GB slobodnog prostora za Docker images
- **CPU**: 4+ jezgara

---

## Kloniranje repozitorijuma

```bash
# Kloniraj repozitorijum
git clone https://github.com/serbia-gov/platform.git

# Uđi u direktorijum
cd platform
```

---

## Brza instalacija (AI Demo)

Najbrži način da pokrenete ceo sistem sa svim komponentama:

```bash
# Pokreni sve servise jednom komandom
docker compose -f deploy/docker/docker-compose.ai-demo.yml up -d
```

Ovo pokreće:
- **PostgreSQL** - Relaciona baza podataka (port 5500)
- **KurrentDB** - Event sourcing baza (port 2113)
- **AI Mock Service** - Simulacija LLM-a (port 5000)
- **Platform API** - Backend Go servis (port 8080)
- **Demo UI** - React frontend (port 3001)

### Provera statusa

```bash
# Proveri da li su svi kontejneri healthy
docker compose -f deploy/docker/docker-compose.ai-demo.yml ps

# Pogledaj logove
docker compose -f deploy/docker/docker-compose.ai-demo.yml logs -f
```

### Zaustavljanje

```bash
# Zaustavi sve servise
docker compose -f deploy/docker/docker-compose.ai-demo.yml down

# Zaustavi i obriši volumes (resetuj bazu)
docker compose -f deploy/docker/docker-compose.ai-demo.yml down -v
```

---

## Ručna instalacija

Ako želite više kontrole ili razvijate lokalno bez Docker-a za backend.

### 1. Pokreni infrastrukturne servise

```bash
# Samo PostgreSQL i KurrentDB
docker compose up -d
```

### 2. Instaliraj Go dependencies

```bash
# Preuzmi sve Go module
go mod download

# Verifikuj
go mod verify
```

### 3. Pokreni migracije baze

```bash
# Migracije se automatski pokreću pri startu aplikacije
# Ali možeš i ručno:
go run ./cmd/platform migrate
```

### 4. Pokreni backend

```bash
# Opcija 1: Direktno pokretanje
go run ./cmd/platform

# Opcija 2: Build pa pokreni
go build -o bin/platform ./cmd/platform
./bin/platform

# Opcija 3: Sa hot reload-om (zahteva Air)
air
```

### 5. Pokreni frontend (opciono)

```bash
# Uđi u web folder
cd web

# Instaliraj dependencies
npm install

# Pokreni dev server
npm run dev
```

---

## Pokretanje servisa

### Korišćenje Makefile-a

```bash
# Pokreni Docker servise
make docker-up

# Build i pokreni backend
make run

# Pokreni sa hot reload-om
make dev

# Zaustavi Docker servise
make docker-down

# Pogledaj sve dostupne komande
make help
```

### Environment varijable

Backend koristi sledeće environment varijable (sa default vrednostima):

```bash
# Server
ENV=development
SERVER_PORT=8080

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=platform
DB_PASSWORD=platform
DB_NAME=platform

# KurrentDB
KURRENTDB_HOST=localhost
KURRENTDB_PORT=2113
KURRENTDB_INSECURE=true

# AI Service
AI_ENABLED=true
AI_SERVICE_URL=http://localhost:5000

# Auth
JWT_SECRET=dev-secret-key
```

Možeš ih setovati u `.env` fajlu ili direktno u terminalu:

```bash
# Primer: Promeni port
SERVER_PORT=9000 go run ./cmd/platform
```

---

## Pristup aplikaciji

### Web interfejsi

| Servis | URL | Kredencijali |
|--------|-----|--------------|
| **Demo UI** | http://localhost:3001 | - |
| **Platform API** | http://localhost:8080 | - |
| **KurrentDB UI** | http://localhost:2113 | - |
| **AI Mock Service** | http://localhost:5000 | - |
| **Keycloak** | http://localhost:8180 | admin / admin |
| **Grafana** | http://localhost:3000 | admin / admin |
| **MinIO Console** | http://localhost:9001 | minioadmin / minioadmin |

### API Endpoints

```bash
# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Lista agencija
curl http://localhost:8080/api/v1/agencies

# Audit verifikacija lanca
curl http://localhost:8080/api/v1/audit/verify

# Pokreni simulaciju
curl -X POST http://localhost:8080/api/v1/simulation/start
```

### KurrentDB Stream Browser

Otvori http://localhost:2113 u browseru:
1. Klikni na "Stream Browser" u meniju
2. Pretraži streamove:
   - `$audit` - Audit log entries
   - `$audit-checkpoints` - Checkpoint events
   - `gov-*` - Domain events

---

## Razvoj

### Struktura projekta

```
platform/
├── cmd/
│   └── platform/          # Main entry point
│       └── main.go
├── internal/
│   ├── agency/            # Agency modul (CRUD)
│   ├── audit/             # Audit modul (hash chain)
│   ├── case/              # Case modul (DDD)
│   ├── document/          # Document modul
│   ├── ai/                # AI integracija
│   ├── federation/        # Multi-agency federation
│   ├── privacy/           # Privacy guard
│   ├── simulation/        # Demo simulacija
│   ├── tsa/               # Time Stamping Authority
│   └── shared/
│       ├── auth/          # Auth middleware
│       ├── config/        # Konfiguracija
│       ├── database/      # DB konekcija i migracije
│       ├── errors/        # Error types
│       ├── events/        # Event bus (KurrentDB)
│       └── types/         # Common types (ID, JMBG...)
├── web/                   # React frontend
├── services/
│   └── ai-mock/           # AI mock service (Python)
├── deploy/
│   └── docker/            # Docker konfiguracije
├── docs/                  # Dokumentacija
└── Makefile
```

### Dodavanje novog modula

1. Kreiraj folder u `internal/`
2. Implementiraj:
   - `repository.go` - Data access
   - `handler.go` - HTTP handlers
   - `types.go` - Domain types
3. Registruj rute u `cmd/platform/main.go`
4. Dodaj events u `internal/shared/events/`

### Rad sa KurrentDB

```go
// Publish event
event := events.NewEvent("case.created", "case-service", caseData)
event = event.WithActor(actorID, "worker", agencyID)
bus.Publish(ctx, event)

// Subscribe to events
bus.Subscribe(ctx, "case.*", "my-handler", func(ctx context.Context, e events.Event) error {
    // Handle event
    return nil
})
```

### Rad sa Audit logom

```go
// Audit se automatski popunjava iz events
// Ali možeš i ručno dodati entry:
entry := audit.NewAuditEntry(
    audit.ActorTypeWorker,
    workerID,
    &agencyID,
    "document.uploaded",
    "document",
    &documentID,
    changes,
    "", // prevHash se automatski računa
)
auditRepo.Append(ctx, entry)
```

---

## Testiranje

### Unit testovi

```bash
# Pokreni sve testove
go test ./...

# Sa verbose output-om
go test -v ./...

# Samo specifičan paket
go test -v ./internal/audit/...

# Sa coverage-om
make test-coverage
# Otvori coverage.html u browseru
```

### Integration testovi

```bash
# Zahteva pokrenute Docker servise
docker compose up -d

# Pokreni integration testove
go test -tags=integration ./...
```

### E2E testiranje sa Demo UI

1. Pokreni ceo stack: `docker compose -f deploy/docker/docker-compose.ai-demo.yml up -d`
2. Otvori http://localhost:3001
3. Navigiraj na "Simulacija" stranicu
4. Klikni "Pokreni simulaciju"
5. Prati events u real-time

---

## Troubleshooting

### Docker problemi

**Problem: Port already in use**
```bash
# Pronađi šta koristi port
netstat -ano | findstr :8080  # Windows
lsof -i :8080                  # Linux/Mac

# Promeni port u docker-compose ili env varijablama
```

**Problem: Container ne startuje**
```bash
# Pogledaj logove
docker logs ai-demo-platform

# Restartuj sa čistom bazom
docker compose -f deploy/docker/docker-compose.ai-demo.yml down -v
docker compose -f deploy/docker/docker-compose.ai-demo.yml up -d
```

**Problem: KurrentDB health check fails**
```bash
# KurrentDB treba malo vremena da se pokrene
# Sačekaj 30 sekundi i proveri ponovo
docker logs ai-demo-kurrentdb
```

### Go problemi

**Problem: Module not found**
```bash
# Očisti module cache
go clean -modcache

# Ponovo preuzmi
go mod download
```

**Problem: Build fails**
```bash
# Proveri Go verziju
go version  # Treba 1.22+

# Ažuriraj dependencies
go mod tidy
```

### Frontend problemi

**Problem: npm install fails**
```bash
# Očisti npm cache
npm cache clean --force

# Obriši node_modules i ponovo instaliraj
rm -rf node_modules package-lock.json
npm install
```

### Baza podataka

**Problem: Migration fails**
```bash
# Proveri konekciju
psql -h localhost -p 5432 -U platform -d platform

# Reset baze
docker compose down -v
docker compose up -d
```

### Česti problemi na Windows

1. **Line endings**: Git može konvertovati CRLF u LF
   ```bash
   git config --global core.autocrlf false
   ```

2. **Docker Desktop**: Uključi WSL2 backend za bolje performanse

3. **Firewall**: Dozvoli Docker-u pristup mreži

---

## Dodatni resursi

- [Tech Stack dokumentacija](tech-stack.md)
- [Domain Model](domain-model.md)
- [Event Catalog](event-catalog.md)
- [AI Usage in System](AI-USAGE-IN-SYSTEM.md)
- [MVP Implementation Plan](MVP-IMPLEMENTATION-PLAN.md)

---

## Kontakt

Za pitanja ili probleme:
- Otvori GitHub Issue
- Kontaktiraj tim za razvoj
