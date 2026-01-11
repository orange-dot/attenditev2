# Tehničko obrazloženje arhitekture platforme

> Verzija: 1.0 | Datum: 2026-01-10

Ovaj dokument objašnjava tehničke odluke iza platforme za interoperabilnost javnih službi, sa fokusom na bezbednost, održivost i transparentnost.

---

## 1. Zašto Open Source za vladine sisteme?

### 1.1 Kerkhoffsov princip - temelj moderne bezbednosti

U kriptografiji postoji princip star 150 godina koji je temelj **SVE** moderne bezbednosti. Auguste Kerckhoffs je 1883. godine u *La Cryptographie Militaire* postavio pravilo:

> **"Sistem mora biti siguran čak i ako sve o njemu, osim ključa, postane javno znanje."**

Američki matematičar Claude Shannon (otac teorije informacija) preformulisao je ovo kao:

> **"Neprijatelj poznaje sistem."** (*"The enemy knows the system."*)

Ovo je poznato kao **Shannonova maksima** i predstavlja suprotnost tzv. "bezbednosti kroz nepoznatost" (*security through obscurity*).

### 1.2 Schneierova analogija sa sefom

Bruce Schneier, autor knjige *Applied Cryptography* (1996), dao je možda najbolju ilustraciju ovog principa:

> *"Ako uzmem pismo, zaključam ga u sef, sakrijem sef negde u Njujorku, i onda vam kažem da pročitate pismo - to nije bezbednost. To je nejasnoća (obscurity).*
>
> *S druge strane, ako uzmem pismo i zaključam ga u sef, a onda vam dam taj sef zajedno sa specifikacijama dizajna sefa i sto identičnih sefova sa njihovim kombinacijama, tako da vi i najbolji obijači sefova na svetu možete proučavati mehanizam zaključavanja - a vi i dalje ne možete otvoriti sef i pročitati pismo - TO je bezbednost."*

### 1.3 Logika protiv "security through obscurity"

**Ako bi samo otkrivanje nekog detalja sistema učinilo sistem nesigurnim - onda taj sistem NIJE siguran.**

Zašto? Jer kako ćete sačuvati taj detalj tajnim? Trebao bi vam drugi kriptosistem! A cela poenta prvog sistema je bila čuvanje tajni. To je cirkularni problem.

**Moderna primena:** HTTPS, SSL, AES, RSA - **SVI** su javno objavljeni standardi. Njihova bezbednost ne zavisi od tajnosti algoritma, već isključivo od tajnosti ključa. Milioni stručnjaka ih pregledaju, testiraju, napadaju - i upravo zato su sigurni.

### 1.4 Šta to znači za vladine sisteme?

| Pristup | Realnost |
|---------|----------|
| Tajna arhitektura | Lažna bezbednost - napadač će je ionako otkriti (reverse engineering, insider leak, itd.) |
| Javna arhitektura + tajni ključevi | Prava bezbednost - pregledana od strane hiljada nezavisnih stručnjaka |

**Open source omogućava nezavisnu reviziju** - greške se otkrivaju pre nego što ih napadači iskoriste. Zatvoreni sistemi te greške skrivaju - ali samo od javnosti, ne od napadača.

---

## 2. AGPL v3 licenca - zaštita javnih resursa

### 2.1 Zašto baš AGPL?

AGPL v3 licenca specifično štiti vladine interese:

| Karakteristika | Objašnjenje |
|----------------|-------------|
| **Sloboda korišćenja** | Neograničeno, bez plaćanja |
| **Sloboda modifikacije** | Može se prilagoditi lokalnim potrebama |
| **Obaveza otvaranja** | Sve modifikacije moraju biti javne |
| **Copyleft klauzula** | Ako neko koristi kod u svom sistemu, i taj sistem mora biti otvoren |
| **Network klauzula** | Čak i ako se koristi samo preko mreže (SaaS), izvorni kod mora biti dostupan |

### 2.2 Zaštita od privatizacije javnih resursa

AGPL sprečava scenario koji se često dešava:
1. Privatna firma uzme javno finansiran open source kod
2. Napravi zatvorenu (proprietary) verziju sa modifikacijama
3. Prodaje tu verziju nazad državi

Sa AGPL licencom, **sve modifikacije moraju ostati javne**. Javni novac → javni kod.

---

## 3. Go programski jezik

### 3.1 Zašto Go?

Platforma je implementirana u **Go programskom jeziku** (Google, 2009), koji je specifično dizajniran za velike infrastrukturne sisteme.

Go koriste: **Google, Cloudflare, Uber, Twitch, Dropbox, Docker, Kubernetes**

### 3.2 "Bez magije" filozofija

Go je dizajniran sa radikalnom jednostavnošću. Dok drugi jezici dodaju funkcionalnosti tokom vremena, Go ih namerno ograničava.

| Jezik | Veličina specifikacije |
|-------|------------------------|
| Java | 750 stranica |
| C# | 517 stranica |
| **Go** | **50 stranica** |

**Šta to znači u praksi:**
- Svaki Go programer razume svaku liniju Go koda
- Nema "magičnih" frameworka koji rade stvari iza scene
- Kod radi tačno ono što piše - nema skrivenih ponašanja
- Princip: *"Manje je više"* (*"Less is more"*)

### 3.3 Bezbednosne karakteristike jezika

Go na nivou kompajlera forsira pravila koja su u drugim jezicima samo preporuke:

| Karakteristika | Bezbednosni benefit |
|----------------|---------------------|
| **Zabranjene ciklične zavisnosti** | Sprečava skrivene ranjivosti i kompleksnost |
| **Zabranjene nekorišćene promenljive** | Čistiji, pregledniji kod - lakše uočiti greške |
| **Zabranjeni nekorišćeni importi** | Manja površina napada |
| **Nema implicitnih konverzija tipova** | Sprečava čitavu klasu grešaka |
| **Eksplicitna obrada grešaka** | Programer MORA da razmisli o svakoj grešci |

Iz zvaničnog Go bloga:
> *"Eksplicitno rukovanje greškama primorava programera da razmisli o greškama - i da ih obradi - kada se dogode. Rezultujući kod može biti duži, ali jasnoća i jednostavnost takvog koda nadoknađuje njegovu opširnost."*

### 3.4 Operativne prednosti

- **Standardna biblioteka pokriva ~90% potreba** → manje eksternih zavisnosti → manja površina napada
- **Kompajlira se u jedan binarni fajl** → jednostavna distribucija, manje pokretnih delova
- **Cross-compilation** → isti kod radi na Linux, Windows, macOS
- **Ugrađena konkurentnost** → efikasno korišćenje resursa servera

---

## 4. Modularni monolit arhitektura

### 4.1 Šta je modularni monolit?

Modularni monolit je arhitekturni obrazac koji kombinuje:
- **Jednostavnost monolita** - jedna aplikacija, jedan deployment
- **Fleksibilnost mikroservisa** - jasno razdvojeni moduli sa definisanim granicama

### 4.2 Zašto ne mikroservisi?

Mikroservisi uvode značajnu kompleksnost:

| Problem | Objašnjenje |
|---------|-------------|
| Distribuirana komunikacija | Mrežni pozivi umesto memorijskih |
| Konzistentnost podataka | Distribuirane transakcije su teške |
| Operativna kompleksnost | Service discovery, load balancing, circuit breakers |
| Debugging | Traganje grešaka kroz više servisa |
| Deployment | Koordinacija verzija između servisa |

**Shopify** je počeo kao monolit, evoluirao u modularni monolit, i izvlači mikroservise samo za specifične potrebe (checkout, fraud detection).

### 4.3 "Best of both worlds"

| Karakteristika | Modularni monolit |
|----------------|-------------------|
| **Deployment** | Jedan artefakt - jednostavno |
| **Komunikacija** | In-memory pozivi - brzo |
| **Razvoj** | Jedan codebase - lakše debugovanje |
| **Modularnost** | Jasne granice između modula |
| **Migracija** | Prirodan put ka mikroservisima ako zatreba |

### 4.4 Struktura naše platforme

```
platform/
├── internal/
│   ├── agency/       # Modul: Agencije i radnici
│   ├── cases/        # Modul: Upravljanje slučajevima
│   ├── documents/    # Modul: Dokumenti i verzije
│   ├── audit/        # Modul: Audit log (append-only)
│   ├── federation/   # Modul: Međuagencijska komunikacija
│   └── ai/           # Modul: AI detekcija anomalija
```

Svaki modul:
- Ima sopstvenu bazu (logički razdvojene šeme)
- Komunicira sa drugima kroz definisane interfejse
- Može se nezavisno testirati
- Može se izvući u mikroservis ako performanse to zahtevaju

### 4.5 Preporuke po veličini tima

Iz industrijskih podataka:

| Veličina tima | Preporučena arhitektura |
|---------------|-------------------------|
| 1-10 developera | Monolit |
| 10-50 developera | **Modularni monolit** ← naša preporuka |
| 50+ developera | Mikroservisi postaju opravdani |

Za pilot projekat sa ograničenim resursima, modularni monolit je optimalan izbor.

---

## 5. Rezime tehničkih odluka

| Odluka | Obrazloženje |
|--------|--------------|
| **Open Source** | Kerkhoffsov princip - prava bezbednost, nezavisna revizija |
| **AGPL v3** | Zaštita javnih resursa od privatizacije |
| **Go jezik** | Jednostavnost, bezbednost, performanse |
| **Modularni monolit** | Balans između jednostavnosti i fleksibilnosti |
| **PostgreSQL** | Pouzdana, open source baza sa naprednim funkcijama |
| **NATS JetStream** | Lightweight event bus za asinhrone operacije |

---

## Izvori

- [Bruce Schneier - Applied Cryptography, Preface](https://www.schneier.com/books/applied-cryptography-2preface/)
- [Kerckhoffs's principle - Wikipedia](https://en.wikipedia.org/wiki/Kerckhoffs%27s_principle)
- [Go at Google: Language Design in the Service of Software Engineering](https://go.dev/talks/2012/splash.article)
- [The Origins and Design Philosophy of Go Language](https://leapcell.io/blog/the-origins-and-design-philosophy-of-go-language)
- [What Is a Modular Monolith? - Milan Jovanović](https://www.milanjovanovic.tech/blog/what-is-a-modular-monolith)
- [Modular Monoliths vs. Microservices](https://adriankodja.com/modular-monoliths-vs-microservices)
- [Microservices vs. Modular Monoliths in 2025](https://www.javacodegeeks.com/2025/12/microservices-vs-modular-monoliths-in-2025-when-each-approach-wins.html)
