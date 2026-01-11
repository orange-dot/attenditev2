# Upotreba AI u Sistemu Socijalne i Zdravstvene Zaštite

> Verzija: 1.0 | Datum: 2026-01-10
> ARGUS | Udruženje građana "Lišeni Svega"

---

## Sažetak

Ovaj dokument opisuje predloženu upotrebu veštačke inteligencije (AI) u sistemu koordinacije socijalne i zdravstvene zaštite. Fokus je na:

1. **Detekciji anomalija** - automatsko prepoznavanje opasnih grešaka u dokumentaciji
2. **Asistenciji lekarima** - AI kao pomoćnik u dijagnostici, ne zamena za lekara
3. **Decentralizovano hostovanje** - svaka ustanova može imati svoj LLM; podaci se čuvaju lokalno i šalju na analizu samo kada je potrebno, bez centralnog skladištenja

---

## 1. Detekcija Anomalija

### Problem

Tokom naše analize dokumentovali smo primere sistemskih grešaka koje su opasne po zdravlje i život pacijenata:

| Tip greške | Primer iz prakse | Rizik |
|------------|------------------|-------|
| **Nemoguća uputstva** | Uputstvo slepom pacijentu (dijagnostifikovana retinopatija oba oka) da "čita vrednosti i zapisuje" | Pacijent ne može da prati terapiju |
| **Nekonzistentnost dijagnoza-preporuka** | Zaključak da je postupanje "u skladu sa dobrom praksom" pri glikemiji od 0.7 mmol/L i izjavi da "nega nije obezbeđena" | Lažno oslobađanje od odgovornosti |
| **Konflikt podataka** | Otpusna lista kaže "sestra vodi računa" dok pacijent živi sam | Pacijent ostaje bez nege |

### Rešenje: AI Detekcija

AI sistem može u realnom vremenu analizirati medicinsku dokumentaciju i detektovati:

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI ANOMALY DETECTION                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [Ulaz: Medicinska dokumentacija]                               │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────────────────────────────┐                   │
│  │  1. Ekstrakcija dijagnoza i uputstava   │                   │
│  │  2. Provera logičke konzistentnosti     │                   │
│  │  3. Unakrsna validacija sa protokolima  │                   │
│  │  4. Detekcija kontradiktornih podataka  │                   │
│  └─────────────────────────────────────────┘                   │
│           │                                                     │
│           ▼                                                     │
│  [Izlaz: Alert ako postoji anomalija]                          │
│                                                                 │
│  Primeri alerta:                                                │
│  ⚠️ "Uputstvo zahteva vid, pacijent ima Dx retinopatije"       │
│  ⚠️ "Glikemija 0.7 mmol/L zahteva hospitalizaciju po Batuta"   │
│  ⚠️ "Otpusna lista navodi negu, socijalni status: živi sam"    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Implementacija

**Model:** DeepSeek-R1 (MIT licenca) sa fine-tuningom na srpskim medicinskim protokolima

**Zašto DeepSeek-R1:**
- Chain-of-thought reasoning - objašnjava ZAŠTO je nešto anomalija
- MIT licenca - potpuno slobodna upotreba
- Već korišćen u healthcare (Fangzhou Inc.)
- Može se fine-tunovati na lokalnim podacima

---

## 2. AI Asistencija Lekarima

### Princip: Kolaboracija, ne zamena

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│   LEKAR                              AI                         │
│   ┌─────┐                         ┌─────┐                      │
│   │     │  ◄─── Preporuke ───────│     │                      │
│   │     │                         │     │                      │
│   │     │  ─── Odluka ──────────►│     │                      │
│   └─────┘                         └─────┘                      │
│      │                               │                          │
│      │                               │                          │
│   Kliničko                     Kompletna                       │
│   iskustvo                     med. istorija                   │
│   Kontekst                     Pattern matching                │
│   Empatija                     24/7 dostupnost                 │
│   FINALNA ODLUKA               Bez umora                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Dokazana efikasnost

| Metrika | Vrednost | Izvor |
|---------|----------|-------|
| Lekari koji koriste AI (SAD, 2024) | 66% (+78% od 2023) | [AMA](https://www.ama-assn.org/practice-management/digital-health/2-3-physicians-are-using-health-ai-78-2023) |
| AI tačnost - plućni noduli | 94% vs 65% (radiolozi) | [WEF](https://www.weforum.org/stories/2024/09/ai-diagnostics-health-outcomes/) |
| AI senzitivnost - karcinom dojke | 90% vs 78% (radiolozi) | Istraživanje Južna Koreja |
| FDA odobreni AI medicinski uređaji | ~950 (avg 2024) | [NCBI](https://www.ncbi.nlm.nih.gov/books/NBK613808/) |
| Smanjenje vremena dokumentacije | do 70% | [Mayo Clinic](https://www.mcpdigitalhealth.org/article/S2949-7612(24)00041-5/fulltext) |

### Ključna poruka

> **Ni AI sam, ni lekar sam - zajedno su najefikasniji.**

AI ima pristup kompletnoj medicinskoj istoriji i može identifikovati obrasce koje čovek ne može. Lekar donosi konačne odluke koristeći kliničko iskustvo i kontekst.

---

## 3. Predloženi Open-Source Modeli

### Tier 1: Optimalna kombinacija za produkciju

| Model | Namena | Licenca | Performanse |
|-------|--------|---------|-------------|
| **OpenBioLLM-70B** | Medicinski reasoning | Apache 2.0 | Nadmašuje GPT-4 i Med-PaLM (86.06% na 9 benchmarkova) |
| **DeepSeek-R1** | Opšti reasoning + detekcija anomalija | MIT | Chain-of-thought, već u healthcare upotrebi |
| **GLM-4.5V** | Analiza medicinskih slika | Open | Vision-language, 12B active params |

### Zašto ovi modeli?

**OpenBioLLM-70B:**
- Baziran na Meta Llama-3
- Specifično treniran na biomedicinskim podacima
- [Hugging Face](https://huggingface.co/blog/aaditya/openbiollm)
- Nadmašuje GPT-4 na medicinskim zadacima

**DeepSeek-R1:**
- MIT licenca - bez ograničenja
- Već deployovan u healthcare (Fangzhou Inc., februar 2025)
- [PMC članak](https://pmc.ncbi.nlm.nih.gov/articles/PMC11836063/)
- Može se fine-tunovati na srpskim protokolima (Batuta, itd.)

**GLM-4.5V:**
- Vision-language model
- Može analizirati RTG, CT, MRI slike
- [Zhipu AI](https://www.siliconflow.com/articles/en/best-open-source-LLM-for-medical-diagonisis)
- 106B total / 12B active (MoE arhitektura)

### Tier 2: Lakša varijanta (manje resursa)

| Model | Parametri | Zahtevi | Napomena |
|-------|-----------|---------|----------|
| PMC-LLaMA | 13B | 1x A100 40GB | Nadmašuje ChatGPT |
| Hippo (Hippocrates) | 7B | RTX 4090 | Nadmašuje 70B modele |

---

## 4. Infrastruktura - Decentralizovana Arhitektura

### Ključni principi

1. **Podaci se čuvaju lokalno** - svaka ustanova čuva svoje podatke
2. **Nema centralnog skladišta** - Data Centar Kragujevac NE čuva medicinske podatke
3. **LLM kao servis** - podaci se šalju na analizu i odmah brišu
4. **Opcija lokalnog LLM-a** - veće ustanove mogu imati sopstveni model

---

### Opcija A: Ustanova SA sopstvenim LLM-om

Za veće ustanove (klinički centri, velike bolnice) koje imaju IT kapacitete:

```
┌─────────────────────────────────────────────────────────────────┐
│                    KLINIČKI CENTAR VOJVODINE                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐                    │
│  │   Lokalni LLM   │    │  Lokalna baza   │                    │
│  │  (Hippo-7B ili  │◄──►│    podataka     │                    │
│  │   PMC-LLaMA)    │    │                 │                    │
│  └─────────────────┘    └─────────────────┘                    │
│           │                                                     │
│           │ Samo metapodaci / agregirani izveštaji             │
│           ▼                                                     │
│  ┌─────────────────────────────────────────┐                   │
│  │         Nacionalna mreža (X-Road)       │                   │
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  ═══════════════════════════════════════════                   │
│  PUNI PODACI NIKADA NE NAPUŠTAJU USTANOVU                      │
│  ═══════════════════════════════════════════                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Hardware za lokalnu instalaciju:**

| Model | GPU | RAM | Cena (approx) |
|-------|-----|-----|---------------|
| Hippo-7B | RTX 4090 (24GB) | 32GB | ~2,500 EUR |
| PMC-LLaMA-13B | A100 40GB | 64GB | ~15,000 EUR |

---

### Opcija B: Ustanova BEZ sopstvenog LLM-a

Za manje ustanove (domovi zdravlja, CSR-ovi, gerontološki centri):

```
┌─────────────────────────────────────────────────────────────────┐
│                      DOM ZDRAVLJA KIKINDA                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐                                           │
│  │  Lokalna baza   │  Podaci ostaju OVDE                       │
│  │    podataka     │                                           │
│  └────────┬────────┘                                           │
│           │                                                     │
│           │ (1) Zahtev za analizu                              │
│           │     [šalje se SAMO dokument za analizu]            │
│           ▼                                                     │
└───────────┼─────────────────────────────────────────────────────┘
            │
            │ Enkriptovani kanal (mTLS)
            ▼
┌───────────────────────────────────────────────────────────────────┐
│                    DATA CENTAR KRAGUJEVAC                         │
├───────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────────┐    ┌─────────────────┐                      │
│  │  OpenBioLLM-70B │    │   DeepSeek-R1   │                      │
│  └────────┬────────┘    └────────┬────────┘                      │
│           │                      │                                │
│           └──────────┬───────────┘                                │
│                      │                                            │
│              ┌───────▼───────┐                                    │
│              │   PROCESIRA   │                                    │
│              │   ODMAH BRIŠE │◄── NEMA ČUVANJA!                   │
│              └───────┬───────┘                                    │
│                      │                                            │
│           (2) Vraća SAMO rezultat analize                        │
│                      │                                            │
└──────────────────────┼────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                      DOM ZDRAVLJA KIKINDA                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐                                           │
│  │  Lokalna baza   │◄── Rezultat se čuva LOKALNO               │
│  │    podataka     │                                           │
│  └─────────────────┘                                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

### Tok podataka - detaljan prikaz

```
┌────────────────────────────────────────────────────────────────────┐
│                         TOK PODATAKA                               │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  1. LOKALNO ČUVANJE                                               │
│     ┌─────────────────────────────────────────────────────────┐   │
│     │ Ustanova čuva SVE podatke u svojoj bazi                 │   │
│     │ - Medicinska dokumentacija                               │   │
│     │ - Socijalni kartoni                                      │   │
│     │ - Istorija lečenja                                       │   │
│     └─────────────────────────────────────────────────────────┘   │
│                              │                                     │
│                              ▼                                     │
│  2. ZAHTEV ZA ANALIZU (opciono)                                   │
│     ┌─────────────────────────────────────────────────────────┐   │
│     │ Kada je potrebna AI analiza:                             │   │
│     │ - Šalje se SAMO dokument koji treba analizirati          │   │
│     │ - Enkriptovan prenos (mTLS + AES-256)                    │   │
│     │ - Bez identifikacionih podataka ako nije neophodno       │   │
│     └─────────────────────────────────────────────────────────┘   │
│                              │                                     │
│                              ▼                                     │
│  3. OBRADA U MEMORIJI                                             │
│     ┌─────────────────────────────────────────────────────────┐   │
│     │ Data Centar Kragujevac:                                  │   │
│     │ - Prima dokument                                         │   │
│     │ - Procesira u RAM-u (NIKAD na disk)                      │   │
│     │ - Generiše analizu/alert                                 │   │
│     │ - ODMAH BRIŠE ulazne podatke                             │   │
│     └─────────────────────────────────────────────────────────┘   │
│                              │                                     │
│                              ▼                                     │
│  4. REZULTAT NAZAD                                                │
│     ┌─────────────────────────────────────────────────────────┐   │
│     │ Vraća se SAMO:                                           │   │
│     │ - Rezultat analize (npr. "Anomalija detektovana")        │   │
│     │ - Preporuke                                              │   │
│     │ - Referenca na protokol                                  │   │
│     │                                                          │   │
│     │ NE vraća se:                                             │   │
│     │ - Originalni dokument (već ga ustanova ima)              │   │
│     │ - Kopija podataka                                        │   │
│     └─────────────────────────────────────────────────────────┘   │
│                              │                                     │
│                              ▼                                     │
│  5. LOKALNO ČUVANJE REZULTATA                                     │
│     ┌─────────────────────────────────────────────────────────┐   │
│     │ Ustanova čuva rezultat analize u svojoj bazi            │   │
│     │ Centralni sistem NEMA KOPIJU                             │   │
│     └─────────────────────────────────────────────────────────┘   │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

---

### Zašto ovako? Prednosti decentralizacije

| Aspekt | Centralizovano (tradicionalno) | Decentralizovano (naš predlog) |
|--------|-------------------------------|--------------------------------|
| **Rizik curenja podataka** | Visok - sve na jednom mestu | Nizak - podaci distribuirani |
| **Single point of failure** | Da | Ne |
| **Usklađenost sa GDPR** | Komplikovano | Jednostavno - podaci ostaju kod kontrolora |
| **Troškovi storage-a** | Ogromni (centralno) | Distribuirani (lokalno) |
| **Latencija** | Veća (sve ide preko centra) | Manja (lokalni pristup) |
| **Autonomija ustanova** | Niska | Visoka |

---

### Hardware zahtevi

**Data Centar Kragujevac (centralni LLM servis):**

| Komponenta | Specifikacija | Napomena |
|------------|---------------|----------|
| GPU | 4x NVIDIA A100 80GB | Za OpenBioLLM-70B + DeepSeek-R1 |
| RAM | 512GB | Obrada u memoriji, bez diska |
| Storage | 500GB NVMe SSD | Samo za modele, NE za podatke |
| Network | 100Gbps | Niska latencija |

**Lokalna instalacija (opciono za veće ustanove):**

| Tier | GPU | RAM | Model | Cena |
|------|-----|-----|-------|------|
| Mini | RTX 4090 | 32GB | Hippo-7B | ~2,500 EUR |
| Standard | A100 40GB | 64GB | PMC-LLaMA-13B | ~15,000 EUR |
| Full | 2x A100 80GB | 256GB | OpenBioLLM-70B | ~50,000 EUR |

---

### Bezbednost

- **Podaci ostaju u ustanovi** - centralni sistem NEMA kopiju
- **Obrada u memoriji** - ulazni podaci se nikad ne pišu na disk
- **Enkripcija in transit** - mTLS + AES-256
- **Audit log** - svaki zahtev za analizu se beleži (bez sadržaja)
- **Zero-knowledge arhitektura** - centralni sistem "ne zna" šta je analizirao nakon obrade
- **Pravo na zaborav** - automatski implementirano (nema šta da se briše)

---

## 5. Primeri Upotrebe

### Primer 1: Detekcija nemogućeg uputstva

**Ulaz:** Otpusna lista sa:
- Dijagnoza: H36.0 (Retinopathia diabetica, OU)
- Uputstvo: "Da upiše sve glikemije manje od 3,5"

**AI Analiza:**
```
⚠️ ANOMALIJA DETEKTOVANA

Dijagnoza H36.0 (dijabetička retinopatija oba oka) ukazuje na
teško oštećenje vida ili slepoću.

Uputstvo zahteva da pacijent:
1. ČITA vrednosti na glukometru
2. ZAPISUJE vrednosti

Ovo je fizički nemoguće za pacijenta sa ovom dijagnozom.

PREPORUKA: Obezbediti asistenciju ili govorni glukometar.
```

### Primer 2: Nekonzistentnost u proceni

**Ulaz:** Izveštaj savetnika sa:
- Činjenica: Glikemija 0.7 mmol/L
- Činjenica: Lekar izjavio "nega NIJE obezbeđena"
- Zaključak: "Postupanje u skladu sa dobrom praksom"

**AI Analiza:**
```
⚠️ LOGIČKA NEKONZISTENTNOST

Prema Vodiču Batuta za prehospitalna urgentna stanja:
- Glikemija < 2.2 mmol/L = OZBILJNA hipoglikemija
- Hipoglikemija sa dugim insulinom ZAHTEVA hospitalizaciju

Činjenice u izveštaju:
- Glikemija: 0.7 mmol/L (3x ispod kritičnog praga)
- Nega: NIJE obezbeđena (izjava lekara)
- Odluka: Nije hospitalizovan

Zaključak "dobra praksa" je u kontradikciji sa:
1. Protokolom Batuta
2. Izjavom samog lekara

PREPORUKA: Revidirati zaključak.
```

### Primer 3: Konflikt između dokumenata

**Ulaz:**
- Otpusna lista: "Sestra vodi računa o bolesniku"
- Socijalni karton: "Živi sam, nema srodnika u gradu"

**AI Analiza:**
```
⚠️ KONFLIKT PODATAKA

Otpusna lista navodi: "Sestra vodi računa o bolesniku"
Socijalni karton navodi: "Živi sam"

Ovi podaci su u direktnoj kontradikciji.

PREPORUKA: Verifikovati stvarno stanje pre otpusta.
```

---

## 6. Sledeći Koraci

1. **Pilot projekat** - Testiranje na anonimiziranim podacima
2. **Fine-tuning** - Prilagođavanje modela srpskim protokolima
3. **Integracija** - Povezivanje sa postojećim sistemima (ISS, eZdravlje)
4. **Evaluacija** - Merenje tačnosti i korisnosti
5. **Skaliranje** - Proširenje na sve CSR-ove i zdravstvene ustanove

---

## Reference

- [OpenBioLLM - Hugging Face](https://huggingface.co/blog/aaditya/openbiollm)
- [DeepSeek in Healthcare - PMC](https://pmc.ncbi.nlm.nih.gov/articles/PMC11836063/)
- [Meditron - GitHub](https://github.com/epfLLM/meditron)
- [AMA AI Survey 2024](https://www.ama-assn.org/practice-management/digital-health/2-3-physicians-are-using-health-ai-78-2023)
- [WEF AI Diagnostics](https://www.weforum.org/stories/2024/09/ai-diagnostics-health-outcomes/)
- [Mayo Clinic AI Documentation](https://www.mcpdigitalhealth.org/article/S2949-7612(24)00041-5/fulltext)
- [FDA AI Medical Devices - NCBI](https://www.ncbi.nlm.nih.gov/books/NBK613808/)

---

*Dokument pripremio: ARGUS, Udruženje građana "Lišeni Svega"*
*Uz asistenciju: Claude Opus 4.5 (Anthropic)*
*Licenca: AGPL v3*
