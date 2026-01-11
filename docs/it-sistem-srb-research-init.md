# Serbia's fragmented IT ecosystem leaves vulnerable populations without coordinated protection

Serbia's public sector IT systems for social protection, healthcare, and police function largely as **isolated digital islands**—well-developed individually but critically disconnected when vulnerable citizens need coordinated emergency response. The €1.4 billion invested in the SOZIS social protection software since 2020 serves only Centers for Social Work, leaving residential care institutions and cross-ministry coordination unsupported. Meanwhile, the €27 million invested in Serbia's 112 unified emergency number remains largely unspent, with the system still not operational as of January 2026. This fragmentation directly impacts the most vulnerable: there is no national 24/7 social emergency hotline, no automated data exchange between health and social services, and after-hours coordination depends entirely on manual phone calls between agencies.

## The social protection IT stack prioritizes benefits over integration

The social protection information system architecture consists of three primary components, none of which fully integrate with healthcare or police systems:

**SOZIS (Sistem za zaštitu i automatizaciju instrumenata socijalne zaštite)** launched in December 2021 serves as the primary software for Serbia's 171 Centers for Social Work. The Ministry of Labor invested over **1.417 billion dinars (€12.09 million)** between 2020-2024, with development by a consortium led by Asseco SEE. Despite 27 version updates, social workers report the system slows their work rather than accelerating it. The Belgrade CSR union strike in February 2025 cited SOZIS dysfunction as a contributing factor, alongside caseloads of up to **250 cases per worker** compared to approximately 30 in EU countries.

**The Social Card (Socijalna Karta)** registry, operational since March 2022 with **656 million dinars** in investment, collects approximately 135 data points per beneficiary from external sources including MUP, Tax Administration, and the Pension Fund. The system generates monthly notifications to CSR workers about status changes. However, automatic data matching has led to controversial outcomes: over **44,000 people** lost cash social assistance after implementation, with critics noting beneficiaries were excluded "without their participation in procedures" and without considering actual circumstances.

**Critical gap for residential care:** Neither SOZIS nor the Social Card directly connects residential care institutions such as Dom za stare i penzionere Kikinda. These facilities report through a separate GIZ-developed system to Republican and Provincial Institutes for Social Protection 1-2 times annually—entirely disconnected from the real-time case management systems used by CSRs. The data flow architecture reveals the fundamental limitation:

| Level | System | Connection Status |
|-------|--------|-------------------|
| Ministry (MINRZS) | SOZIS central, Social Card | Central hub |
| Provincial (Vojvodina) | Pokrajinski zavod | Reporting only |
| Municipal (CSRs) | SOZIS client | Connected |
| Residential care | GIZ reporting software | **Disconnected** |

## Healthcare digitalization advances but operates independently from social services

Serbia's healthcare IT ecosystem demonstrates stronger central coordination than social protection, with the **IZIS (Integrisani zdravstveni informacioni sistem)** framework operational since 2008 and continuously expanded. The system now supports a comprehensive suite of e-health services that healthcare workers and citizens use daily.

**Operational e-health services as of January 2026:**
- **eRecept** (since November 2017): Electronic prescriptions with 2-6 month renewable prescriptions for chronic patients
- **eZakazivanje/MojDoktor** portal: National appointment scheduling with mobile app support
- **eBolovanje** (launched March 2025): Fully digital sick leave documentation between physicians, employers, and RFZO
- **eKarton** (full implementation from January 2025): Unified health records across public and private providers

The RFZO maintains **237 servers** with approximately **70,000 healthcare workers** having system access. The infrastructure is ISO 27001 certified for information security, though a 2021 audit revealed all users were initially assigned identical passwords—a vulnerability subsequently addressed.

**The health-social coordination gap** remains the critical weakness for vulnerable populations. Despite regulations requiring social protection providers to facilitate healthcare access, no automated data exchange exists between health facilities and CSRs. The Public Health Strategy 2018-2026 acknowledges the need for inter-sectoral coordination, but implementation relies on administrative channels rather than integrated systems. When a domestic violence victim presents at a hospital or a vulnerable elderly person is discharged, there is no automatic notification to social services—coordination depends on manual referrals that frequently fail outside business hours.

## Police systems prioritize citizen services over inter-agency integration

The Ministry of Interior (MUP) has emerged as a "pioneer" in citizen-facing e-government services, with over **2.6 million registered users** on the eUprava portal accessing services including online appointment booking, criminal record certificates, and driver's license renewals. The AFIS (Automated Fingerprint Identification System) has doubled criminal identification rates, and Regional Criminal Forensic Centers in Novi Sad, Niš, and Užice provide distributed capacity.

**The 112 emergency number remains inoperative despite €27+ million in funding.** EU funding of €1.5 million (2019-2020) and a Chinese donation of €25.6 million (2022) have not resulted in a functioning unified emergency system. A tragic March 2024 incident underscored the consequences: a 22-year-old woman died waiting while emergency services from Grocka and Kaluđerica disputed jurisdictional responsibility. Citizens must still call separate numbers—192 for police, 193 for fire, 194 for medical emergencies—with no unified dispatch coordination.

Inter-agency protocols for domestic violence and child protection do exist on paper. The **Law on Prevention of Domestic Violence (2017, amended 2023)** mandates Coordination Groups meeting every 15 days, comprising prosecutors, police, and CSR representatives. Police can issue 48-hour emergency protective orders, and the law requires immediate CSR notification. However, the protocol assumes same-day physical document delivery and working-hours availability—assumptions that fail for after-hours emergencies.

## Inter-system coordination relies on protocols rather than technology

Serbia has developed legitimate interoperability infrastructure that functions primarily for administrative document exchange:

The **eZUP system** connects over **400 public institutions** with more than **10,000 employees** processing tens of thousands of exchanges weekly. The **Servisna magistrala organa** (Government Service Bus), mandated by the 2018 Law on Electronic Administration, enables query-based access to public registries. These systems successfully eliminate citizen trips to collect certificates—but they were designed for document exchange, not real-time emergency coordination.

**What happens when police receive a social emergency call at 2 AM:**
1. Police dispatch responds and assesses immediate safety
2. If children or vulnerable adults are at risk, police attempt to reach CSR on-call duty worker by phone
3. CSR availability depends on local arrangements—some have duty staff until 17:00, others rely on on-call workers reachable by mobile
4. If the on-call worker doesn't answer, police may need to contact multiple numbers or wait until morning
5. No system automatically alerts CSR case managers about police interventions in their active cases
6. Health facilities treating victims have no automated connection to either police or CSR records

The **Pravilnik on CSR Organization** requires 24-hour emergency access but permits this through three mechanisms: reorganized hours, duty shifts, or on-call readiness from home. Implementation varies dramatically by municipality, with no national standard for response time guarantees.

## State audits reveal fundamental security and governance failures

The September 2025 **State Audit Institution (DRI) performance audit** of the Central Registry of Compulsory Social Insurance (CROSO) exposed critical vulnerabilities that likely exist across social protection IT systems:

- **No information security management system** has been established
- A generic administrator account is shared by **2 CROSO employees and 3 external vendor employees**
- The maintenance service provider has **direct access to production databases** containing all insured persons' data
- **No event log monitoring** procedures exist
- Backup systems have never been tested
- The system enables **retroactive backdating** of registration records

The DRI's CSR audit found work "not organized in accordance with principles of responsible governance," with only **1,671 professional workers serving 750,000 beneficiaries**—leaving insufficient time for thorough case management. Serbia has had no Social Protection Strategy since 2010; a strategy drafted in 2018 was never adopted.

**EU assessments paint a consistent picture.** The European Commission's 2024 Serbia Report found the country "moderately prepared with **no progress made**" on public administration reform and "**limited progress**" on digital transformation. The 2025 Report noted Serbia still lacks a comprehensive court case management system. Serbia's average EU readiness grade stands at **3.11 out of 5**, with 9 negotiation chapters showing no progress.

## Cybersecurity incidents reveal systemic vulnerability

Recent incidents demonstrate how technical weaknesses translate into real harm:

The **December 2023 ransomware attack** on state energy company EPS by Russian-speaking group Qilin resulted in 34GB of data published on the dark web, including employee ID cards and personal contracts. Neither EPS nor the government confirmed what was compromised. The National CERT stated it has "no authority or ability to determine whether certain data belong to an institution."

**The NoviSpy spyware scandal** (December 2024, documented by Amnesty International) revealed Serbian police and BIA have been using domestically developed spyware installed on individuals' devices during detention, targeting political activists, journalists, and civil society members. During 2024 student protests, pro-government media published passport photographs and personal data of protesters—direct violations of data protection law that demonstrate how systems meant to protect can be weaponized.

## What Estonia, Finland, and the Netherlands do differently

**Estonia's X-Road** provides the clearest technical model for Serbia. Operational since 2001, the system processes **2.2 billion transactions annually** across **52,000 organizations** while maintaining complete decentralization—each agency keeps its own database, exchanging data only when needed via encrypted, authenticated connections. Key principles:

- **Once-only data provision**: Citizens provide information once; systems must reuse it
- **No central data repository**: Reduces breach impact and political misuse potential
- **All transactions logged**: Complete audit trail of who accessed what, when
- **Open source**: Available for any country to adopt (MIT License since 2016)

**Finland's Kanta system** demonstrates health-social integration. Mandatory since 2023 for social welfare services, Kanta enables patient/client data to flow across organizational boundaries with consent management built in. Finland's 2023 reform transferred health, social, and rescue services from municipalities to regional Wellbeing Services Counties—creating structural integration that Serbia's fragmented municipal responsibilities cannot replicate without similar reform.

**The Netherlands' wijkteams (neighborhood social teams)** offer a service delivery model Serbia could pilot without major IT investment. These interdisciplinary teams co-locate social workers, healthcare professionals, and youth care specialists with police liaison—creating a "one-stop-shop" for citizens needing assistance. Over **450 care partnerships** exist nationally, with the **Buurtzorg model** demonstrating how self-managing teams can deliver superior outcomes with minimal administrative overhead.

| Country | Central coordination | Real-time integration | Social-health-police | 24/7 emergency social |
|---------|---------------------|----------------------|---------------------|---------------------|
| Estonia | RIA authority | X-Road operational | Via X-Road queries | Municipal services |
| Finland | Kela + THL | Kanta mandatory | Integrated since 2023 | Regional counties |
| Netherlands | Agency for Digitization | Wijkteams local | Veilig Thuis centers | SAMUR-style pilots |
| **Serbia** | Fragmented | **Not implemented** | **Protocol-based only** | **Not available** |

## What an integrated system would require

Achieving functional cross-agency coordination for vulnerable populations would require Serbia to address three levels simultaneously:

**Technical requirements:**
- Implement X-Road-style secure data exchange connecting SOZIS, IZIS, and MUP systems
- Create real-time notification capabilities for emergency interventions across agencies
- Develop unified citizen identifier that works across all systems (building on existing JMBG)
- Establish tested, monitored backup systems and proper information security management

**Governance requirements:**
- Establish central interoperability authority with cross-ministerial mandate
- Mandate participation by all relevant agencies (not just encourage it)
- Create clear data sharing agreements specifying what can be exchanged under what circumstances
- Build municipal capacity to implement and maintain connections (the "eUprava za sve" project reaching only 10 of 40 applicant municipalities suggests demand far exceeds support)

**Service delivery requirements:**
- Establish 24/7 social emergency hotline with national coverage
- Pilot integrated neighborhood teams combining social, health, and police liaison
- Complete 112 unified emergency number implementation with real dispatch coordination
- Standardize CSR after-hours coverage with response time guarantees

## Conclusion: the advocacy path forward

Serbia's IT ecosystem paradox—ranked 2nd in Europe on the World Bank's GovTech Maturity Index while State Auditors find no information security management in social insurance systems—reveals the gap between showcase achievements and operational reality. The systems that work well serve administrative convenience; the coordination that vulnerable populations need during emergencies remains absent.

For advocacy focused on inter-service coordination, the most actionable immediate targets are:

1. **Demand 112 implementation accountability**: €27+ million spent without a functioning system requires explanation and timeline commitment
2. **Press for 24/7 social emergency coverage**: The legal requirement exists; implementation varies—name and shame municipalities with inadequate coverage
3. **Expose the residential care disconnection**: Institutions caring for the most vulnerable have no real-time connection to the systems tracking their residents
4. **Use the DRI findings**: The September 2025 CROSO audit provides official documentation of security failures that should apply pressure for reform
5. **Advocate for pilot integrated teams**: The Netherlands' wijkteam model requires minimal IT investment—just co-location and coordination protocols

The European Commission has been documenting "no progress" for years. EU accession pressure alone has proven insufficient. Domestic advocacy combining the documented failures with specific international alternatives may prove more effective than waiting for gradual institutional improvement.