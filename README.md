# Serbia Government Interoperability Platform

Platforma za interoperabilnost Vlade Republike Srbije.

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Node.js 18+ (za frontend)
- Make (optional)

> **Detaljne instrukcije**: Pogledaj [docs/LOCAL-SETUP.md](docs/LOCAL-SETUP.md) za kompletnu dokumentaciju.

### Najbrži start (AI Demo)

```bash
# Pokreni ceo stack jednom komandom
docker compose -f deploy/docker/docker-compose.ai-demo.yml up -d

# Otvori u browseru
# Demo UI: http://localhost:3001
# API: http://localhost:8080
# KurrentDB: http://localhost:2113
```

### Development Setup

1. **Start infrastructure services:**

```bash
docker-compose up -d
```

2. **Run the platform:**

```bash
go run ./cmd/platform
```

Or with Make:

```bash
make run
```

3. **Run frontend (optional):**

```bash
cd web && npm install && npm run dev
```

4. **Access the API:**

- Demo UI: http://localhost:3001 (ako koristiš AI Demo)
- API: http://localhost:8080/api/v1
- Health: http://localhost:8080/health
- KurrentDB UI: http://localhost:2113
- Keycloak: http://localhost:8180 (admin/admin)
- Grafana: http://localhost:3000 (admin/admin)

### API Endpoints

#### Agencies

```
GET    /api/v1/agencies           # List agencies
POST   /api/v1/agencies           # Create agency
GET    /api/v1/agencies/{id}      # Get agency
PUT    /api/v1/agencies/{id}      # Update agency
DELETE /api/v1/agencies/{id}      # Delete agency
```

#### Workers

```
GET    /api/v1/agencies/{id}/workers  # List workers in agency
POST   /api/v1/agencies/{id}/workers  # Create worker
GET    /api/v1/workers/{id}           # Get worker
PUT    /api/v1/workers/{id}           # Update worker
DELETE /api/v1/workers/{id}           # Delete worker
```

### Project Structure

```
/cmd/platform          # Application entry point
/internal
  /agency              # Agency module (CRUD)
  /case                # Case module (DDD)
  /document            # Document module
  /audit               # Audit module (hash chain, KurrentDB)
  /ai                  # AI integration
  /federation          # Multi-agency federation
  /privacy             # Privacy guard (PII protection)
  /simulation          # Demo simulation
  /tsa                 # Time Stamping Authority (RFC 3161)
  /shared              # Shared kernel
    /auth              # Authentication middleware
    /config            # Configuration
    /database          # Database connection & migrations
    /errors            # Error types
    /events            # Event bus (KurrentDB)
    /types             # Common types (ID, JMBG, etc.)
/web                   # React frontend
/services/ai-mock      # AI mock service (Python)
/docs                  # Architecture documentation
/deploy/docker         # Docker configurations
```

### Documentation

- [Local Setup Guide](docs/LOCAL-SETUP.md) - Detaljna instalacija
- [Tech Stack](docs/tech-stack.md)
- [Domain Model](docs/domain-model.md)
- [Event Catalog](docs/event-catalog.md)
- [AI Usage in System](docs/AI-USAGE-IN-SYSTEM.md)
- [MVP Plan](docs/MVP-IMPLEMENTATION-PLAN.md)

## License

Proprietary - Government of Serbia
