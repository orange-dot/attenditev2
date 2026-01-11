# Adapter Architecture for Serbian Public Administration Interoperability

> Verzija: 1.0 | Datum: 2026-01-10

---

## Executive Summary

Ovaj dokument definiše arhitekturu adaptera za integraciju fragmentisanih IT sistema javne uprave Srbije sa centralnom platformom za koordinaciju u Državnom data centru u Kragujevcu. Poseban fokus je na zdravstvenim podacima i real-time case management.

**Ključni paradoks**: Srbija je rangirana **2. u Evropi** na World Bank GovTech Maturity Index, ali State Audit Institution nalazi da ne postoji sistem za upravljanje bezbednošću informacija u sistemima socijalnog osiguranja. Sistemi koji rade dobro služe administrativnoj pogodnosti; koordinacija koju ranjive populacije trebaju tokom hitnih situacija **ne postoji**.

---

## 0. Paradoks postojeće infrastrukture

### 0.1 Sistemi postoje - koordinacija ne

Srbija je investirala značajna sredstva u IT infrastrukturu javne uprave:

| Investicija | Iznos | Status | Problem |
|-------------|-------|--------|---------|
| **SOZIS** (CSR softver) | €12.09M (2020-2024) | Aktivan (171 CSR) | Samo za CSR, ne povezuje zdravstvo/policiju |
| **Socijalna karta** | €5.6M (656M RSD) | Aktivan | 44,000 izgubilo pomoć zbog netačnih podataka |
| **112 jedinstveni broj** | €27M+ | **NIJE OPERATIVAN** | Sredstva nepotrošena od 2019 |
| **eZUP** | - | Aktivan (400+ institucija) | Dizajniran za dokumente, ne real-time |

### 0.2 Šta se dešava kada policija primi socijalni poziv u 2 ujutru

```
TRENUTNI TOK (bez integracije):

1. Policijski dispečer prima poziv i procenjuje bezbednost

2. Ako su deca ili ranjive odrasle osobe u opasnosti:
   ├── Policija pokušava telefonom da dođe do dežurnog CSR radnika
   ├── Dostupnost CSR zavisi od lokalnih aranžmana:
   │   ├── Neki imaju dežurstvo do 17:00
   │   └── Drugi se oslanjaju na pripravnost (mobilni)

3. Ako dežurni radnik ne odgovara:
   ├── Policija mora da zove više brojeva
   └── Ili čeka do jutra

4. NEMA automatskog obaveštavanja CSR case managera
   o policijskim intervencijama u njihovim aktivnim predmetima

5. Zdravstvene ustanove koje tretiraju žrtve NEMAJU
   automatsku vezu sa policijskim ili CSR evidencijama

REZULTAT: Mart 2024 - žena (22) umrla čekajući dok su hitne službe
iz Grocke i Kaluđerice raspravljale o nadležnosti
```

### 0.3 Zašto eZUP i Servisna magistrala nisu rešenje

**eZUP sistem** (400+ institucija, 10,000+ zaposlenih) uspešno eliminiše putovanja građana po uverenja - ali je dizajniran za **razmenu dokumenata**, ne za **real-time koordinaciju hitnih situacija**.

```
eZUP MOŽE:                           eZUP NE MOŽE:
├── Izvod iz matične knjige          ├── Real-time notifikaciju CSR-u
├── Potvrda o prebivalištu           ├── Automatsku koordinaciju agencija
├── Uverenje o državljanstvu         ├── Eskalaciju hitnih slučajeva
└── Query-based pristup registrima   └── Event-driven komunikaciju
```

### 0.4 Rezidencijalne ustanove potpuno izolovane

**KRITIČAN GAP**: Domovi za stare i penzionere (npr. Dom Kikinda) koriste potpuno odvojeni GIZ sistem za izveštavanje Republičkom i Pokrajinskom zavodu - **1-2 puta godišnje**. Nemaju real-time vezu sa CSR sistemima koji prate njihove korisnike.

| Nivo | Sistem | Status veze |
|------|--------|-------------|
| Ministarstvo (MINRZS) | SOZIS central, Soc. karta | Centralni hub |
| Pokrajina (Vojvodina) | Pokrajinski zavod | Samo izveštaji |
| Opština (CSR) | SOZIS klijent | Povezan |
| **Rezidencijalna nega** | GIZ softver | **IZOLOVAN** |

---

## 1. Analiza zabrinutosti iz pisama

### 1.1 Identifikovani problemi

| Problem | Izvor | Kritičnost |
|---------|-------|------------|
| **Fragmentacija zdravstvenih podataka** | DZ nema podatke iz bolnice | KRITIČNO |
| **Pacijenti kao kuriri** | Fizičko nošenje dokumentacije | VISOKO |
| **CSR bez bolničkih podataka** | Nedostaje istorija hospitalizacija | VISOKO |
| **Gerontološki centri izolovani** | Nema uvid u CSR podatke | SREDNJE |
| **Lekari bez kompletne istorije** | Manuelni unos podataka | VISOKO |
| **Policija nejasna uloga** | Nedefinisani protokoli | SREDNJE |
| **Pripravnost CSR simulirana** | Improvizacija u hitnim slučajevima | KRITIČNO |

### 1.2 Root Cause Analysis

```
┌─────────────────────────────────────────────────────────────────┐
│                    UZROCI FRAGMENTACIJE                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. TEHNIČKI                                                     │
│     ├── Različiti vendori bez integracije                        │
│     ├── Proprietary formati podataka                             │
│     ├── Legacy sistemi bez API-ja                                │
│     └── Odsustvo nacionalnog standarda razmene                   │
│                                                                  │
│  2. ORGANIZACIONI                                                │
│     ├── Silos mentalitet institucija                             │
│     ├── Nejasne nadležnosti                                      │
│     ├── Nedostatak protokola saradnje                            │
│     └── Otpor prema deljenju podataka                            │
│                                                                  │
│  3. REGULATORNI                                                  │
│     ├── GDPR/ZZPL interpretacije                                 │
│     ├── Nedefinisani pravni osnovi razmene                       │
│     └── Strah od odgovornosti                                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Analiza postojećih sistema u Srbiji

### 2.1 Zdravstveni sektor

| Sistem | Opis | Standardi | Status |
|--------|------|-----------|--------|
| **IZIS** | Integrisani zdravstveni IS (od 2008) | Proprietary | Aktivan |
| **eZdravlje** | Portal za građane, uvid u zdravstvene podatke | Proprietary | Aktivan |
| **eRecept** | Elektronski recepti (od nov. 2017) | Proprietary | Aktivan |
| **eKarton** | Elektronski zdravstveni karton (od jan. 2025) | Proprietary | **Puna impl.** |
| **eBolovanje** | Digitalno bolovanje (od mar. 2025) | REST/JSON | Aktivan |
| **HIS (Heliant, InfoMedis)** | Bolnički informacioni sistemi | HL7 v2 parcijalno | Fragmentisan |
| **RFZO sistemi** | 237 servera, 70,000 radnika | Proprietary | Aktivan |

**Ključni nalaz**: IZIS napreduje, ali **ne postoji automatska razmena podataka sa CSR-om**. Kada žrtva nasilja dođe u bolnicu, ne postoji automatska notifikacija socijalnih službi.

**Operativni eHealth servisi (januar 2026)**:
- **eRecept**: 2-6 meseci obnovljivih recepata za hronične pacijente
- **eZakazivanje/MojDoktor**: Nacionalno zakazivanje + mobilna aplikacija
- **eBolovanje**: Potpuno digitalna dokumentacija između lekara, poslodavaca i RFZO
- **eKarton**: Jedinstveni zdravstveni karton kroz javne i privatne pružaoce

### 2.2 Socijalna zaštita

| Sistem | Opis | Investicija | Integracije | Status |
|--------|------|-------------|-------------|--------|
| **SOZIS** | Softver za 171 CSR (Asseco SEE) | €12.09M | Samo CSR međusobno | Aktivan |
| **Socijalna karta** | Centralni registar, ~135 tačaka podataka | €5.6M | MUP, PIO, Katastar, Poreska | Aktivan |
| **GIZ softver** | Izveštavanje rezidencijalnih ustanova | - | **Izolovan** | Aktivan |

**SOZIS problemi (dokumentovano)**:
- Socijalni radnici prijavljuju da sistem **usporava rad** umesto da ga ubrzava
- Sindikat CSR Beograd je u februaru 2025 štrajkovao, navodeći disfunkciju SOZIS-a
- **250 predmeta po radniku** (u EU ~30)
- 27 verzija ažuriranja, ali bez integracije sa zdravstvom/policijom

**Socijalna karta problemi**:
- Automatsko uparivanje podataka dovelo do kontroverznih ishoda
- **44,000+ osoba** izgubilo novčanu socijalnu pomoć nakon implementacije
- Korisnici isključeni "bez njihovog učešća u postupcima"
- Algoritam koji određuje funkcionisanje sistema **nije javno objavljen** (poslovne tajne)

**Ključni nalaz**: Socijalna karta koristi Servisnu magistralu za preuzimanje podataka - ovo je temelj za proširenje na real-time koordinaciju.

### 2.3 Policija i hitne službe

| Sistem | Opis | Status |
|--------|------|--------|
| **eUprava MUP** | 2.6M registrovanih korisnika | Aktivan |
| **AFIS** | Automatska identifikacija otisaka | Aktivan (udvostručena stopa identifikacije) |
| **112 hitni broj** | Jedinstveni evropski broj | **NIJE OPERATIVAN** |

**112 problem**:
- €1.5M EU finansiranje (2019-2020) - nepotrošeno
- €25.6M kineska donacija (2022) - nepotrošeno
- Građani još uvek moraju da zovu odvojene brojeve: 192 (policija), 193 (vatrogasci), 194 (hitna)
- **Nema jedinstvenog dispečerskog centra**

**Zakon o sprečavanju nasilja u porodici (2017, izmenjen 2023)**:
- Koordinacione grupe se sastaju svakih 15 dana (tužilac, policija, CSR)
- Policija može izdati 48-satne hitne mere zaštite
- Zakon zahteva trenutno obaveštavanje CSR-a
- **ALI**: protokol pretpostavlja dostavu dokumenata istog dana i dostupnost u radnom vremenu

### 2.4 Infrastruktura eUprave

| Komponenta | Opis | Kapacitet |
|------------|------|-----------|
| **Državni data centar Kragujevac** | Tier 4, EN 50600 Class 4 | 1180 rack kabineta |
| **eUprava portal** | 2.5M korisnika, 1000+ servisa | Aktivan |
| **eZUP** | Razmena dokumenata | 400+ institucija, 10,000+ zaposlenih |
| **Servisna magistrala** | Government Service Bus | Aktivan (Socijalna karta) |
| **AI platforma** | 5 PetaFlops, nVidia DGX A100 | Aktivan |
| **Planirana ekspanzija** | +40MW kapaciteta | MoU potpisan 2025 |

**Ključni nalaz**: Infrastruktura postoji - nedostaju adapteri za legacy sisteme i **event-driven koordinacija**.

### 2.5 DRI Audit nalazi (Septembar 2025)

Državna revizorska institucija (DRI) je objavila performansnu reviziju CROSO (Centralni registar obaveznog socijalnog osiguranja) koja otkriva kritične ranjivosti:

| Nalaz | Opis | Kritičnost |
|-------|------|------------|
| **Nema ISMS-a** | Sistem za upravljanje bezbednošću informacija ne postoji | KRITIČNO |
| **Deljeni nalog** | Generički admin nalog dele 2 CROSO + 3 vendor zaposlena | KRITIČNO |
| **Vendor pristup** | Dobavljač ima direktan pristup produkcijskoj bazi | VISOKO |
| **Nema monitoringa** | Procedure za praćenje event logova ne postoje | VISOKO |
| **Backup netestiran** | Backup sistemi nikada nisu testirani | VISOKO |
| **Retroaktivni unos** | Sistem omogućava retroaktivno datiranje registracija | SREDNJE |

**DRI nalaz o CSR-u**:
- Rad "nije organizovan u skladu sa principima odgovornog upravljanja"
- Samo **1,671 stručni radnik** opslužuje **750,000 korisnika**
- Srbija nema Strategiju socijalne zaštite od 2010. (nacrt iz 2018. nikad nije usvojen)

**EU ocena (2024-2025)**:
- "Umereno pripremljena sa **nikakvim napretkom**" u reformi javne uprave
- "**Ograničen napredak**" u digitalnoj transformaciji
- Prosečna ocena EU spremnosti: **3.11 od 5**
- 9 pregovaračkih poglavlja bez napretka

### 2.6 Međunarodna poređenja

| Zemlja | Centralna koordinacija | Real-time integracija | Socijal-zdravlje-policija | 24/7 socijalna hitna |
|--------|----------------------|----------------------|--------------------------|---------------------|
| **Estonija** | RIA autoritet | X-Road operativan | Via X-Road upiti | Opštinski servisi |
| **Finska** | Kela + THL | Kanta obavezan | Integrisano od 2023 | Regionalni okruzi |
| **Holandija** | Agencija za digitalizaciju | Wijkteams lokalni | Veilig Thuis centri | SAMUR piloti |
| **Srbija** | Fragmentisano | **Nije implementirano** | **Samo protokoli** | **Nije dostupno** |

**Estonija X-Road** (model za Srbiju):
- Operativan od 2001, **2.2 milijarde transakcija godišnje**
- 52,000 organizacija povezano
- Ključni principi:
  - **Once-only**: Građani daju informaciju jednom, sistemi moraju da je ponovo koriste
  - **Nema centralnog repozitorijuma**: Svaka agencija čuva svoje podatke
  - **Sve transakcije logirane**: Kompletna revizijska traga
  - **Open source**: MIT licenca od 2016

**Finska Kanta sistem**:
- Obavezan za socijalne usluge od 2023
- Omogućava protok podataka pacijenata/klijenata preko organizacionih granica
- Ugrađeno upravljanje pristankom

**Holandija wijkteams** (najlakše za implementaciju):
- Interdisciplinarni timovi koji kolokuju socijalne radnike, zdravstvene profesionalce, i policijske veze
- "One-stop-shop" za građane kojima je potrebna pomoć
- 450+ partnership-a u nezi
- **Minimalna IT investicija** - samo kolokacija i protokoli koordinacije

### 2.7 Standardi u upotrebi

```
TRENUTNO STANJE:

Zdravstvo:
├── HL7 v2.x - delimično (pojedine bolnice)
├── DICOM - radiologija
├── ICD-10 - dijagnoze
└── ATC - lekovi

Socijalna zaštita:
├── Proprietary XML/JSON
└── REST API (Socijalna karta)

eUprava:
├── REST/JSON
├── SOAP (legacy)
└── X.509 sertifikati
```

---

## 3. Ciljana arhitektura adaptera

### 3.1 Konceptualni model

```
┌────────────────────────────────────────────────────────────────────────────┐
│                        ADAPTER ARCHITECTURE                                 │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   LEGACY SISTEMI                    ADAPTERI                  CENTRALNA     │
│   (Fragmentacija)                   (Edge)                    PLATFORMA     │
│                                                               (Kragujevac)  │
│   ┌─────────────┐                                                          │
│   │ Bolnica A   │──┐                                                       │
│   │ (Heliant)   │  │    ┌──────────────┐                                   │
│   └─────────────┘  ├───►│   Health     │                                   │
│   ┌─────────────┐  │    │   Adapter    │──┐                                │
│   │ Bolnica B   │──┤    │  (HL7/FHIR)  │  │     ┌─────────────────────┐   │
│   │ (InfoMedis) │  │    └──────────────┘  │     │                     │   │
│   └─────────────┘  │                      │     │   INTEROPERABILITY  │   │
│   ┌─────────────┐  │    ┌──────────────┐  │     │      PLATFORM       │   │
│   │ Dom zdravlja│──┘    │   Primary    │──┤     │                     │   │
│   │ (Various)   │──────►│   Care       │  ├────►│  ┌─────────────┐    │   │
│   └─────────────┘       │   Adapter    │  │     │  │ Event Bus   │    │   │
│                         └──────────────┘  │     │  │ (NATS)      │    │   │
│   ┌─────────────┐                         │     │  └─────────────┘    │   │
│   │ CSR Lokalni │       ┌──────────────┐  │     │                     │   │
│   │ (Various)   │──────►│   Social     │──┤     │  ┌─────────────┐    │   │
│   └─────────────┘       │   Protection │  │     │  │ Case Mgmt   │    │   │
│   ┌─────────────┐       │   Adapter    │  │     │  │ (Real-time) │    │   │
│   │ Gerontološki│──────►│              │──┤     │  └─────────────┘    │   │
│   │ centar      │       └──────────────┘  │     │                     │   │
│   └─────────────┘                         │     │  ┌─────────────┐    │   │
│                         ┌──────────────┐  │     │  │ Federation  │    │   │
│   ┌─────────────┐       │   Police     │  │     │  │ Gateway     │    │   │
│   │ MUP/Policija│──────►│   Adapter    │──┤     │  └─────────────┘    │   │
│   │ (Internal)  │       │              │  │     │                     │   │
│   └─────────────┘       └──────────────┘  │     └─────────────────────┘   │
│                                           │                               │
│                         ┌──────────────┐  │                               │
│   ┌─────────────┐       │   Document   │  │                               │
│   │ Socijalna   │──────►│   Exchange  │──┘                               │
│   │ karta       │       │   Adapter    │                                  │
│   └─────────────┘       └──────────────┘                                  │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Adapter tipovi

#### 3.2.1 Health Adapter (Zdravstveni adapter)

```go
// internal/adapters/health/adapter.go

type HealthAdapter interface {
    // Čitanje podataka iz legacy sistema
    FetchPatientRecord(jmbg string) (*PatientRecord, error)
    FetchHospitalizations(jmbg string, from, to time.Time) ([]Hospitalization, error)
    FetchLabResults(jmbg string, from, to time.Time) ([]LabResult, error)
    FetchPrescriptions(jmbg string, active bool) ([]Prescription, error)

    // Real-time događaji
    SubscribeAdmissions(handler func(Admission)) error
    SubscribeDischarges(handler func(Discharge)) error
    SubscribeEmergencies(handler func(Emergency)) error

    // Transformacija u FHIR
    ToFHIRPatient(record *PatientRecord) (*fhir.Patient, error)
    ToFHIREncounter(hosp *Hospitalization) (*fhir.Encounter, error)
}
```

#### 3.2.2 Social Protection Adapter

```go
// internal/adapters/social/adapter.go

type SocialProtectionAdapter interface {
    // Integracija sa Socijalnom kartom
    FetchBeneficiaryStatus(jmbg string) (*BeneficiaryStatus, error)
    FetchFamilyComposition(jmbg string) (*FamilyUnit, error)

    // CSR sistemi
    FetchOpenCases(jmbg string) ([]SocialCase, error)
    FetchCaseHistory(jmbg string) ([]SocialCase, error)
    FetchRiskAssessment(jmbg string) (*RiskAssessment, error)

    // Real-time
    SubscribeCaseUpdates(handler func(CaseUpdate)) error
    SubscribeEmergencyInterventions(handler func(Intervention)) error

    // Koordinacija
    NotifyCSR(agencyCode string, notification Notification) error
    RequestIntervention(request InterventionRequest) error
}
```

#### 3.2.3 Document Exchange Adapter

```go
// internal/adapters/document/adapter.go

type DocumentExchangeAdapter interface {
    // Preuzimanje dokumenata
    FetchDocument(docID string, source string) (*Document, error)
    FetchDocumentsByPerson(jmbg string, docTypes []string) ([]Document, error)

    // Slanje dokumenata
    SendDocument(doc *Document, target string) error
    RequestDocument(request DocumentRequest) (*Document, error)

    // Verifikacija
    VerifySignature(doc *Document) (*SignatureVerification, error)
    VerifyChain(docID string) (*ChainVerification, error)
}
```

### 3.3 Deployment model - Edge adapteri

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      EDGE DEPLOYMENT MODEL                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   LOKACIJA: Bolnica / CSR / Dom zdravlja                                 │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                    EDGE ADAPTER POD                              │   │
│   │                                                                  │   │
│   │   ┌──────────────────┐    ┌──────────────────┐                  │   │
│   │   │  Legacy System   │    │   Adapter Core   │                  │   │
│   │   │   Connector      │───►│                  │                  │   │
│   │   │                  │    │  - Transform     │                  │   │
│   │   │  - DB polling    │    │  - Validate      │                  │   │
│   │   │  - File watch    │    │  - Enrich        │                  │   │
│   │   │  - API proxy     │    │  - Queue         │                  │   │
│   │   └──────────────────┘    └────────┬─────────┘                  │   │
│   │                                    │                             │   │
│   │   ┌──────────────────┐    ┌────────▼─────────┐                  │   │
│   │   │   Local Cache    │    │   Sync Engine    │                  │   │
│   │   │   (SQLite/       │◄───│                  │                  │   │
│   │   │    BoltDB)       │    │  - Batch sync    │                  │   │
│   │   │                  │    │  - Real-time     │                  │   │
│   │   │  - Offline mode  │    │  - Retry logic   │                  │   │
│   │   │  - Buffer        │    │  - Conflict res  │                  │   │
│   │   └──────────────────┘    └────────┬─────────┘                  │   │
│   │                                    │                             │   │
│   │                           ┌────────▼─────────┐                  │   │
│   │                           │   Secure Tunnel  │                  │   │
│   │                           │                  │                  │   │
│   │                           │  - mTLS          │                  │   │
│   │                           │  - Certificate   │                  │   │
│   │                           │  - Compression   │                  │   │
│   │                           └────────┬─────────┘                  │   │
│   │                                    │                             │   │
│   └────────────────────────────────────┼─────────────────────────────┘   │
│                                        │                                  │
│                                        ▼                                  │
│                          ═══════════════════════════                     │
│                          ║   SECURE NETWORK (VPN)  ║                     │
│                          ═══════════════════════════                     │
│                                        │                                  │
│                                        ▼                                  │
│                           ┌─────────────────────┐                        │
│                           │  DRŽAVNI DATA CENTAR │                        │
│                           │     KRAGUJEVAC       │                        │
│                           │                      │                        │
│                           │  ┌───────────────┐   │                        │
│                           │  │  API Gateway  │   │                        │
│                           │  └───────────────┘   │                        │
│                           │         │            │                        │
│                           │  ┌──────▼──────┐     │                        │
│                           │  │  Platform   │     │                        │
│                           │  │   Core      │     │                        │
│                           │  └─────────────┘     │                        │
│                           └─────────────────────┘                        │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Real-time arhitektura za Case Management

### 4.1 Event-driven model

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    REAL-TIME CASE MANAGEMENT                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   EDGE (Institucija)              CENTRAL (Kragujevac)                   │
│                                                                          │
│   ┌──────────────┐               ┌──────────────────────────────────┐   │
│   │ Case Event   │               │         NATS JetStream           │   │
│   │ (lokalno)    │               │                                  │   │
│   │              │──WebSocket───►│  Streams:                        │   │
│   │ - Created    │               │  ├── cases.created               │   │
│   │ - Updated    │               │  ├── cases.updated               │   │
│   │ - Escalated  │               │  ├── cases.escalated             │   │
│   │ - Closed     │               │  ├── cases.shared                │   │
│   └──────────────┘               │  └── cases.emergency             │   │
│                                  │                                  │   │
│                                  │  Consumer Groups:                │   │
│                                  │  ├── audit-service               │   │
│                                  │  ├── notification-service        │   │
│                                  │  ├── analytics-service           │   │
│                                  │  └── coordination-service        │   │
│                                  └──────────────────────────────────┘   │
│                                                                          │
│   LATENCY TARGETS:                                                       │
│   ├── Edge to Central: < 100ms (95th percentile)                        │
│   ├── Event processing: < 50ms                                          │
│   └── Total end-to-end: < 500ms                                         │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Case Coordination Protocol

```go
// internal/coordination/protocol.go

// CaseCoordinationEvent predstavlja real-time događaj
type CaseCoordinationEvent struct {
    ID            string                 `json:"id"`
    Type          CaseEventType          `json:"type"`
    CaseID        string                 `json:"case_id"`
    AgencyCode    string                 `json:"agency_code"`
    Timestamp     time.Time              `json:"timestamp"`
    Priority      Priority               `json:"priority"`
    Data          map[string]interface{} `json:"data"`
    RequiresAck   bool                   `json:"requires_ack"`
    AckDeadline   *time.Time             `json:"ack_deadline,omitempty"`
}

type CaseEventType string

const (
    CaseCreated         CaseEventType = "case.created"
    CaseUpdated         CaseEventType = "case.updated"
    CaseEscalated       CaseEventType = "case.escalated"
    CaseEmergency       CaseEventType = "case.emergency"
    CaseShared          CaseEventType = "case.shared"
    CaseTransferred     CaseEventType = "case.transferred"
    ParticipantAdded    CaseEventType = "case.participant.added"
    InterventionNeeded  CaseEventType = "case.intervention.needed"
    CoordinationRequest CaseEventType = "case.coordination.request"
)

type Priority int

const (
    PriorityLow      Priority = 1
    PriorityNormal   Priority = 2
    PriorityHigh     Priority = 3
    PriorityUrgent   Priority = 4
    PriorityCritical Priority = 5 // Životna ugroženost
)
```

### 4.3 Emergency Protocol - Hitni slučajevi

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    EMERGENCY COORDINATION FLOW                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   1. DETEKCIJA (Edge)                                                    │
│      │                                                                   │
│      │  Adapter detektuje:                                               │
│      │  - Hospitalizacija (bolnica)                                      │
│      │  - Poziv hitnoj (MUP)                                             │
│      │  - Prijava nasilja (CSR)                                          │
│      │                                                                   │
│      ▼                                                                   │
│   2. ESKALACIJA (< 100ms)                                                │
│      │                                                                   │
│      │  Event: case.emergency                                            │
│      │  Priority: CRITICAL                                               │
│      │  RequiresAck: true                                                │
│      │  AckDeadline: 5 minuta                                            │
│      │                                                                   │
│      ▼                                                                   │
│   3. KOORDINACIJA (Central)                                              │
│      │                                                                   │
│      │  Coordination Service:                                            │
│      │  ├── Identifikuje sve relevantne agencije                        │
│      │  ├── Preuzima kontekst (Socijalna karta, eZdravlje)              │
│      │  ├── Kreira koordinacioni predmet                                │
│      │  └── Šalje notifikacije                                          │
│      │                                                                   │
│      ▼                                                                   │
│   4. NOTIFIKACIJA (< 500ms total)                                        │
│      │                                                                   │
│      │  Parallelno obaveštava:                                           │
│      │  ├── CSR (pripravnost ili dežurstvo)                             │
│      │  ├── Policija (ako je potrebno)                                  │
│      │  ├── Hitna pomoć (ako je potrebno)                               │
│      │  └── Nadležni radnik (email, SMS, push)                          │
│      │                                                                   │
│      ▼                                                                   │
│   5. PRAĆENJE                                                            │
│      │                                                                   │
│      │  - Svaki ACK se loguje                                           │
│      │  - Timeout = automatska eskalacija                               │
│      │  - Audit trail kompletne koordinacije                            │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Zdravstveni podaci - FHIR transformacija

### 5.1 Mapiranje na FHIR resurse

| Legacy podatak | FHIR Resource | Napomena |
|----------------|---------------|----------|
| Pacijent | Patient | JMBG → identifier |
| Hospitalizacija | Encounter | Tip: inpatient |
| Pregled | Encounter | Tip: ambulatory |
| Dijagnoza | Condition | ICD-10 kodovi |
| Laboratorija | Observation | LOINC kodovi |
| Recept | MedicationRequest | ATC kodovi |
| Otpusna lista | DocumentReference | CDA ili PDF |
| Upućivanje | ServiceRequest | Referral |

### 5.2 FHIR profili za Srbiju

```json
// deploy/fhir/profiles/SerbianPatient.json
{
  "resourceType": "StructureDefinition",
  "id": "serbian-patient",
  "url": "https://fhir.srbija.gov.rs/StructureDefinition/serbian-patient",
  "name": "SerbianPatient",
  "status": "draft",
  "fhirVersion": "4.0.1",
  "kind": "resource",
  "abstract": false,
  "type": "Patient",
  "baseDefinition": "http://hl7.org/fhir/StructureDefinition/Patient",
  "differential": {
    "element": [
      {
        "id": "Patient.identifier",
        "path": "Patient.identifier",
        "slicing": {
          "discriminator": [{"type": "value", "path": "system"}],
          "rules": "open"
        },
        "min": 1
      },
      {
        "id": "Patient.identifier:jmbg",
        "path": "Patient.identifier",
        "sliceName": "jmbg",
        "min": 1,
        "max": "1"
      },
      {
        "id": "Patient.identifier:jmbg.system",
        "path": "Patient.identifier.system",
        "fixedUri": "https://fhir.srbija.gov.rs/sid/jmbg"
      },
      {
        "id": "Patient.identifier:lbo",
        "path": "Patient.identifier",
        "sliceName": "lbo",
        "min": 0,
        "max": "1",
        "comment": "Lični broj osiguranika (RFZO)"
      },
      {
        "id": "Patient.identifier:lbo.system",
        "path": "Patient.identifier.system",
        "fixedUri": "https://fhir.srbija.gov.rs/sid/lbo"
      }
    ]
  }
}
```

### 5.3 Adapter za Heliant HIS

```go
// internal/adapters/health/heliant/adapter.go

package heliant

import (
    "context"
    "database/sql"
    "time"

    "github.com/serbia-gov/platform/internal/adapters/health"
    "github.com/serbia-gov/platform/internal/shared/types"
)

type HeliantAdapter struct {
    db           *sql.DB
    hospitalCode string
    syncInterval time.Duration
    eventChan    chan health.HealthEvent
}

func NewHeliantAdapter(cfg HeliantConfig) (*HeliantAdapter, error) {
    // Povezivanje na Heliant bazu (read-only)
    db, err := sql.Open("sqlserver", cfg.ConnectionString)
    if err != nil {
        return nil, err
    }

    return &HeliantAdapter{
        db:           db,
        hospitalCode: cfg.HospitalCode,
        syncInterval: cfg.SyncInterval,
        eventChan:    make(chan health.HealthEvent, 1000),
    }, nil
}

func (a *HeliantAdapter) FetchPatientRecord(ctx context.Context, jmbg string) (*health.PatientRecord, error) {
    // Query Heliant tables
    query := `
        SELECT
            p.PatientID,
            p.JMBG,
            p.FirstName,
            p.LastName,
            p.DateOfBirth,
            p.Gender,
            p.Address,
            p.Phone,
            p.LBO
        FROM Patients p
        WHERE p.JMBG = @jmbg
    `

    var record health.PatientRecord
    err := a.db.QueryRowContext(ctx, query, sql.Named("jmbg", jmbg)).Scan(
        &record.LocalID,
        &record.JMBG,
        &record.FirstName,
        &record.LastName,
        &record.DateOfBirth,
        &record.Gender,
        &record.Address,
        &record.Phone,
        &record.LBO,
    )
    if err != nil {
        return nil, err
    }

    record.SourceSystem = "heliant"
    record.SourceHospital = a.hospitalCode

    return &record, nil
}

func (a *HeliantAdapter) FetchHospitalizations(ctx context.Context, jmbg string, from, to time.Time) ([]health.Hospitalization, error) {
    query := `
        SELECT
            h.HospitalizationID,
            h.AdmissionDate,
            h.DischargeDate,
            h.Department,
            h.DiagnosisICD10,
            h.DiagnosisText,
            h.AttendingPhysician,
            h.DischargeType
        FROM Hospitalizations h
        INNER JOIN Patients p ON h.PatientID = p.PatientID
        WHERE p.JMBG = @jmbg
          AND h.AdmissionDate >= @from
          AND h.AdmissionDate <= @to
        ORDER BY h.AdmissionDate DESC
    `

    rows, err := a.db.QueryContext(ctx, query,
        sql.Named("jmbg", jmbg),
        sql.Named("from", from),
        sql.Named("to", to),
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var hospitalizations []health.Hospitalization
    for rows.Next() {
        var h health.Hospitalization
        if err := rows.Scan(
            &h.ID,
            &h.AdmissionDate,
            &h.DischargeDate,
            &h.Department,
            &h.DiagnosisICD10,
            &h.DiagnosisText,
            &h.AttendingPhysician,
            &h.DischargeType,
        ); err != nil {
            return nil, err
        }
        h.SourceHospital = a.hospitalCode
        hospitalizations = append(hospitalizations, h)
    }

    return hospitalizations, nil
}

// Real-time: Poll za nove prijeme/otpuste
func (a *HeliantAdapter) StartEventPolling(ctx context.Context) error {
    ticker := time.NewTicker(a.syncInterval)
    defer ticker.Stop()

    var lastChecked time.Time = time.Now().Add(-a.syncInterval)

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // Check for new admissions
            admissions, err := a.fetchNewAdmissions(ctx, lastChecked)
            if err != nil {
                // Log error, continue
                continue
            }

            for _, adm := range admissions {
                a.eventChan <- health.HealthEvent{
                    Type:      health.EventAdmission,
                    Timestamp: adm.AdmissionDate,
                    Data:      adm,
                }
            }

            // Check for new discharges
            discharges, err := a.fetchNewDischarges(ctx, lastChecked)
            if err != nil {
                continue
            }

            for _, dis := range discharges {
                a.eventChan <- health.HealthEvent{
                    Type:      health.EventDischarge,
                    Timestamp: dis.DischargeDate,
                    Data:      dis,
                }
            }

            lastChecked = time.Now()
        }
    }
}

func (a *HeliantAdapter) Events() <-chan health.HealthEvent {
    return a.eventChan
}
```

---

## 6. Integracija sa Socijalnom kartom

### 6.1 Korišćenje postojeće Servisne magistrale

```go
// internal/adapters/social/socialcard/client.go

package socialcard

import (
    "context"
    "crypto/tls"
    "encoding/json"
    "net/http"

    "github.com/serbia-gov/platform/internal/adapters/social"
)

type SocialCardClient struct {
    baseURL    string
    httpClient *http.Client
    cert       tls.Certificate
}

func NewSocialCardClient(cfg SocialCardConfig) (*SocialCardClient, error) {
    cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
    if err != nil {
        return nil, err
    }

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
    }

    return &SocialCardClient{
        baseURL: cfg.BaseURL,
        httpClient: &http.Client{
            Transport: &http.Transport{
                TLSClientConfig: tlsConfig,
            },
        },
        cert: cert,
    }, nil
}

func (c *SocialCardClient) FetchBeneficiaryStatus(ctx context.Context, jmbg string) (*social.BeneficiaryStatus, error) {
    // Poziv prema Servisnoj magistrali
    req, err := http.NewRequestWithContext(ctx, "GET",
        c.baseURL+"/api/v1/beneficiary/"+jmbg+"/status", nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var status social.BeneficiaryStatus
    if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
        return nil, err
    }

    return &status, nil
}

func (c *SocialCardClient) FetchFamilyComposition(ctx context.Context, jmbg string) (*social.FamilyUnit, error) {
    req, err := http.NewRequestWithContext(ctx, "GET",
        c.baseURL+"/api/v1/beneficiary/"+jmbg+"/family", nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var family social.FamilyUnit
    if err := json.NewDecoder(resp.Body).Decode(&family); err != nil {
        return nil, err
    }

    return &family, nil
}
```

### 6.2 Proširenje konteksta predmeta

```go
// internal/coordination/context_enrichment.go

package coordination

import (
    "context"

    "github.com/serbia-gov/platform/internal/adapters/health"
    "github.com/serbia-gov/platform/internal/adapters/social"
    "github.com/serbia-gov/platform/internal/case/domain"
)

type ContextEnricher struct {
    healthAdapter health.HealthAdapter
    socialAdapter social.SocialProtectionAdapter
}

// EnrichCaseContext obogaćuje predmet sa podacima iz svih izvora
func (e *ContextEnricher) EnrichCaseContext(ctx context.Context, c *domain.Case) (*EnrichedCase, error) {
    enriched := &EnrichedCase{
        Case: c,
    }

    // Za svakog učesnika, preuzmi kontekst
    for _, p := range c.Participants {
        if p.PersonJMBG != nil {
            jmbg := string(*p.PersonJMBG)

            // Zdravstveni podaci
            healthCtx, err := e.fetchHealthContext(ctx, jmbg)
            if err == nil {
                enriched.HealthContexts[jmbg] = healthCtx
            }

            // Socijalni status
            socialCtx, err := e.fetchSocialContext(ctx, jmbg)
            if err == nil {
                enriched.SocialContexts[jmbg] = socialCtx
            }
        }
    }

    return enriched, nil
}

func (e *ContextEnricher) fetchHealthContext(ctx context.Context, jmbg string) (*HealthContext, error) {
    // Poslednje hospitalizacije (90 dana)
    hosps, _ := e.healthAdapter.FetchHospitalizations(jmbg,
        time.Now().AddDate(0, -3, 0), time.Now())

    // Aktivni recepti
    prescriptions, _ := e.healthAdapter.FetchPrescriptions(jmbg, true)

    // Poslednji laboratorijski rezultati
    labs, _ := e.healthAdapter.FetchLabResults(jmbg,
        time.Now().AddDate(0, -1, 0), time.Now())

    return &HealthContext{
        RecentHospitalizations: hosps,
        ActivePrescriptions:    prescriptions,
        RecentLabResults:       labs,
    }, nil
}

func (e *ContextEnricher) fetchSocialContext(ctx context.Context, jmbg string) (*SocialContext, error) {
    // Status korisnika
    status, _ := e.socialAdapter.FetchBeneficiaryStatus(jmbg)

    // Porodična situacija
    family, _ := e.socialAdapter.FetchFamilyComposition(jmbg)

    // Otvoreni predmeti CSR
    cases, _ := e.socialAdapter.FetchOpenCases(jmbg)

    // Procena rizika
    risk, _ := e.socialAdapter.FetchRiskAssessment(jmbg)

    return &SocialContext{
        BeneficiaryStatus: status,
        FamilyUnit:        family,
        OpenCases:         cases,
        RiskAssessment:    risk,
    }, nil
}
```

---

## 7. Security model za adaptere

### 7.1 Autentifikacija i autorizacija

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    ADAPTER SECURITY MODEL                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   EDGE ADAPTER                           CENTRAL PLATFORM                │
│                                                                          │
│   ┌──────────────────┐                   ┌──────────────────┐           │
│   │   Certificate    │                   │  Trust Authority  │           │
│   │   (X.509)        │◄─── mTLS ────────►│                  │           │
│   │                  │                   │  - Verify cert   │           │
│   │   - Agency ID    │                   │  - Check revoke  │           │
│   │   - Adapter ID   │                   │  - Log access    │           │
│   │   - Permissions  │                   └──────────────────┘           │
│   └──────────────────┘                                                  │
│                                                                          │
│   DATA PROTECTION:                                                       │
│   ├── Encryption at rest (AES-256)                                      │
│   ├── Encryption in transit (TLS 1.3)                                   │
│   ├── No PII in logs                                                    │
│   ├── Audit trail for all access                                        │
│   └── GDPR/ZZPL compliant                                               │
│                                                                          │
│   ACCESS CONTROL:                                                        │
│   ├── Adapter can only access own agency data                           │
│   ├── Cross-agency = explicit sharing in platform                       │
│   ├── Emergency override with audit                                     │
│   └── Time-limited access tokens                                        │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 7.2 Data minimization

```go
// internal/adapters/privacy/filter.go

package privacy

import "github.com/serbia-gov/platform/internal/case/domain"

// DataMinimizer filtrira podatke prema principu minimizacije
type DataMinimizer struct {
    accessLevel domain.AccessLevel
}

func (m *DataMinimizer) FilterHealthContext(ctx *HealthContext) *HealthContext {
    filtered := &HealthContext{}

    switch m.accessLevel {
    case domain.AccessLevelFull:
        return ctx // Sve
    case domain.AccessLevelStandard:
        // Bez detaljnih dijagnoza
        filtered.RecentHospitalizations = m.filterHospitalizations(ctx.RecentHospitalizations)
        filtered.ActivePrescriptions = nil // Bez lekova
    case domain.AccessLevelMinimal:
        // Samo činjenica da postoji hospitalizacija
        filtered.HasRecentHospitalization = len(ctx.RecentHospitalizations) > 0
    }

    return filtered
}

func (m *DataMinimizer) FilterSocialContext(ctx *SocialContext) *SocialContext {
    filtered := &SocialContext{}

    switch m.accessLevel {
    case domain.AccessLevelFull:
        return ctx
    case domain.AccessLevelStandard:
        filtered.BeneficiaryStatus = ctx.BeneficiaryStatus
        filtered.RiskLevel = ctx.RiskAssessment.Level // Samo nivo, ne detalji
    case domain.AccessLevelMinimal:
        filtered.IsBeneficiary = ctx.BeneficiaryStatus != nil
    }

    return filtered
}
```

---

## 8. Deployment plan

### 8.1 Faze implementacije

| Faza | Opseg | Trajanje |
|------|-------|----------|
| **Pilot** | 1 bolnica + 1 CSR (isti grad) | 8 nedelja |
| **Proširenje A** | 3 bolnice + 5 CSR (region) | 12 nedelja |
| **Proširenje B** | Svi KC + svi CSR | 24 nedelje |
| **Nacionalno** | Svi DZ, gerontologija, policija | 48 nedelja |

### 8.2 Pilot lokacija: Kikinda

Predlog za pilot:
- **Bolnica**: Opšta bolnica Kikinda (Heliant HIS)
- **CSR**: Centar za socijalni rad Kikinda
- **DZ**: Dom zdravlja Kikinda

Razlozi:
1. Dokumentovani problemi iz pisama
2. Pozitivan primer suda (spremnost na saradnju)
3. Upravljiva veličina za pilot
4. Autor projekta iz Kikinde (lokalno znanje)

### 8.3 Resursi za adapter razvoj

| Adapter | Tehnologija legacy | Kompleksnost | Procena |
|---------|-------------------|--------------|---------|
| Heliant HIS | SQL Server | Srednja | 4 nedelje |
| InfoMedis HIS | Oracle | Visoka | 6 nedelja |
| CSR lokalni | Various | Visoka | 6 nedelja |
| Socijalna karta | REST API | Niska | 2 nedelje |
| MUP sistemi | Unknown | Nepoznata | TBD |

---

## 9. Metrike uspeha

### 9.1 Tehničke metrike

| Metrika | Cilj | Merenje |
|---------|------|---------|
| Latency (p95) | < 500ms | Prometheus |
| Availability | > 99.5% | Uptime monitoring |
| Data freshness | < 5 min | Sync lag |
| Error rate | < 0.1% | Error logs |

### 9.2 Poslovne metrike

| Metrika | Baseline | Cilj | Merenje |
|---------|----------|------|---------|
| Vreme do kompletne istorije | 2-5 dana | < 1 sat | User survey |
| Broj fizičkih putovanja | 3-4 po predmetu | 0-1 | Case tracking |
| Vreme koordinacije hitnih | Nedefinisnao | < 30 min | Event logs |
| Greške zbog nedostatka podataka | TBD | -50% | Incident reports |

---

## 10. Otvorena pitanja

1. **MUP sistemi** - Potreban pristup dokumentaciji i API specifikacijama
2. **Pravni okvir** - Potrebna analiza pravnog osnova za razmenu zdravstvenih podataka
3. **Finansiranje** - Troškovi razvoja adaptera i održavanja
4. **Kapaciteti** - Tim za razvoj i podršku
5. **Koordinacija** - Saglasnost institucija za pilot

---

## Izvori

### Zdravstvo
- [eZdravlje portal](https://www.srbija.gov.rs/vest/en/183151/access-to-personal-medical-data-through-ehealth-portal-mobile-application.php)
- [Serbian HIS improvements 2021-2024](https://health-policy-systems.biomedcentral.com/articles/10.1186/s12961-025-01337-5)
- [HIS Maturity Assessment Serbia](https://www.ghspjournal.org/content/12/5/e2400083)
- [HL7 FHIR standard](https://www.hl7.org/fhir/overview.html)
- [HL7 Europe FHIR guides](https://www.hl7europe.org/new-hl7-europe-fhir-implementation-guides-to-support-the-european-health-data-space/)

### Socijalna zaštita
- [Socijalna karta - Zakon](https://www.paragraf.rs/propisi/zakon-o-socijalnoj-karti.html)
- [Digital surveillance of social protection in Serbia](https://digitalfreedomfund.org/digital-surveillance-of-social-protection-in-serbia/)
- [Anti Socijalne Karte - A11](https://antisocijalnekarte.org/)
- [Serbia Social Card Implementation - China-CEE](https://china-cee.eu/2024/04/10/serbia-political-briefing-two-years-of-the-implementation-of-the-law-on-social-card/)

### eUprava i infrastruktura
- [Državni data centar Kragujevac](https://www.ite.gov.rs/tekst/en/35/government-data-centredr-location-in-kragujevac.php)
- [Smart Serbia platforma](https://www.ite.gov.rs/vest/en/687/smart-serbia-new-government-mass-data-processing-platform-presented-in-the-state-data-centre-in-kragujevac.php)
- [Data centar ekspanzija (e& enterprise)](https://www.datacenterdynamics.com/en/news/serbia-signs-mou-with-e-enterprise-to-triple-national-data-center-capacity/)
- [EDGe projekat](https://www.ite.gov.rs/tekst/en/312/edge.php)
- [eUprava portal](https://www.ite.gov.rs/tekst/en/12/euprava-portal.php)

### Međunarodni modeli
- [X-Road (Estonija)](https://x-road.global/)
- [Kanta (Finska)](https://www.kanta.fi/en/)
- [Wijkteams (Holandija)](https://www.rijksoverheid.nl/)

### Bezbednost i revizija
- [Serbia Data Protection](https://www.dlapiperdataprotection.com/index.html?t=law&c=RS)
- [Serbia GDPR Strategy](https://www.schoenherr.eu/content/bridging-the-gap-between-serbian-regulations-and-the-gdpr-serbia-s-data-protection-strategy-unveiled/)

### Interni dokumenti
- [Inicijalno istraživanje IT sistema](../docs/it-sistem-srb-research-init.md)

---

*Dokument pripremljen kao tehnička specifikacija za adapter arhitekturu*
