# AI Demo - Detekcija Anomalija u Medicinskoj Dokumentaciji

## Brzi Start

```bash
# Iz deploy/docker foldera
cd deploy/docker
docker-compose -f docker-compose.ai-demo.yml up -d --build
```

## Pristup

| Servis | URL | Opis |
|--------|-----|------|
| **Demo UI** | http://localhost:3001 | Web interfejs za testiranje |
| **Platform API** | http://localhost:8080 | REST API |
| **AI Mock** | http://localhost:5000 | Simulacija LLM servisa |
| **PostgreSQL** | localhost:5432 | Baza podataka |
| **KurrentDB** | localhost:2113 | Event streaming |

## Kako Koristiti

1. Otvori http://localhost:3001 u browseru
2. Izaberi jedan od test primera ili unesi sopstveni tekst
3. Klikni "Analiziraj"
4. Pogledaj detektovane anomalije

## Test Primeri

### 1. Nemoguće Uputstvo - Slepi Pacijent
Pacijent sa dijabetičkom retinopatijom (H36.0 - slepoća) dobija uputstvo da čita i zapisuje vrednosti glikemije.

**Očekivana anomalija:** `IMPOSSIBLE_INSTRUCTION` (CRITICAL)

### 2. Kritična Hipoglikemija - Logička Nekonzistentnost
Izveštaj savetnika zaključuje da je postupanje bilo "u skladu sa dobrom praksom" uprkos:
- Glikemiji od 0.7 mmol/L (kritična hipoglikemija)
- Izjavi lekara da "nega nije obezbeđena"

**Očekivana anomalija:** `LOGICAL_INCONSISTENCY` (CRITICAL)

### 3. Konflikt Podataka
Otpusna lista navodi da će se "sestra brinuti", ali socijalni karton pokazuje da pacijent živi sam.

**Očekivana anomalija:** `DATA_CONFLICT` (WARNING)

### 4. Normalan Dokument
Pravilno formirana medicinska dokumentacija bez anomalija.

**Očekivano:** Nema detektovanih anomalija

## API Endpoints

### Analiza dokumenta
```bash
curl -X POST http://localhost:8080/api/v1/ai/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "document_text": "Dijagnoza: H36.0 (Retinopathia diabetica)...",
    "document_type": "medical"
  }'
```

### Primeri za testiranje
```bash
curl http://localhost:8080/api/v1/ai/examples
```

### Health check
```bash
curl http://localhost:8080/api/v1/ai/health
```

## Arhitektura

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Demo UI      │────►│  Platform API   │────►│   AI Mock       │
│   (nginx:80)    │     │   (Go:8080)     │     │  (Python:5000)  │
└─────────────────┘     └────────┬────────┘     └─────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │                         │
              ┌─────▼─────┐            ┌──────▼─────┐
              │ PostgreSQL│            │ KurrentDB  │
              │  (5432)   │            │  (2113)    │
              └───────────┘            └────────────┘
```

## AI Mock vs Produkcija

| Aspekt | AI Mock (Demo) | Produkcija |
|--------|----------------|------------|
| Model | Pattern matching | OpenBioLLM-70B + DeepSeek-R1 |
| Hosting | Docker kontejner | Data Centar Kragujevac |
| Latencija | ~100ms | ~500-2000ms |
| Tačnost | Predefinisani primeri | 86%+ na medicinskim benchmarkovima |

## Zaustavljanje

```bash
docker-compose -f docker-compose.ai-demo.yml down
```

Sa brisanjem podataka:
```bash
docker-compose -f docker-compose.ai-demo.yml down -v
```

## Troubleshooting

### AI Mock ne radi
```bash
docker logs ai-demo-ai-mock
```

### Platform API ne može da se konektuje na AI
Proveri da li je AI Mock zdrav:
```bash
curl http://localhost:5000/health
```

### Demo UI prikazuje "Servis nedostupan"
Proveri sve servise:
```bash
docker-compose -f docker-compose.ai-demo.yml ps
```

---

**ARGUS | Udruženje građana "Lišeni Svega"**
https://lisenisvega.rs
