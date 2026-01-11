"""
AI Mock Service - Simulacija LLM za detekciju anomalija

Ovaj servis simulira ponasanje medicinskog AI modela za potrebe MVP demo-a.
Vraca predefinisane odgovore bazirane na pattern matching-u ulaznog teksta.

Za produkciju: Zameniti sa OpenBioLLM-70B, DeepSeek-R1, ili Hippo-7B
"""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import Optional, List
from enum import Enum
import re
from datetime import datetime

app = FastAPI(
    title="AI Mock Service",
    description="Simulacija medicinskog AI za detekciju anomalija",
    version="1.0.0"
)

# CORS za demo UI
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


class SeverityLevel(str, Enum):
    INFO = "info"
    WARNING = "warning"
    CRITICAL = "critical"


class AnomalyType(str, Enum):
    IMPOSSIBLE_INSTRUCTION = "impossible_instruction"
    LOGICAL_INCONSISTENCY = "logical_inconsistency"
    DATA_CONFLICT = "data_conflict"
    PROTOCOL_VIOLATION = "protocol_violation"


class AnalysisRequest(BaseModel):
    document_text: str
    document_type: Optional[str] = "medical"
    patient_context: Optional[dict] = None


class Anomaly(BaseModel):
    type: AnomalyType
    severity: SeverityLevel
    title: str
    description: str
    evidence: List[str]
    recommendation: str
    protocol_reference: Optional[str] = None


class AnalysisResponse(BaseModel):
    request_id: str
    timestamp: str
    anomalies_found: int
    anomalies: List[Anomaly]
    processing_time_ms: int
    model_used: str
    confidence: float


# Predefinisani obrasci za detekciju
DETECTION_PATTERNS = [
    {
        "patterns": [
            r"retinopat",
            r"slep",
            r"vid.*ostecen",
            r"H36\.0",
            r"dijabeti.*retinopat"
        ],
        "instruction_patterns": [
            r"upis",
            r"pis(ati|uje|e|i)",
            r"[cč]ita",
            r"bele[zž]i",
            r"evidentira",
            r"meri(ti)?.*glikemij",
            r"javi(ti)?.*lekar"
        ],
        "anomaly": {
            "type": AnomalyType.IMPOSSIBLE_INSTRUCTION,
            "severity": SeverityLevel.CRITICAL,
            "title": "Nemoguće uputstvo - zahteva vid kod slepog pacijenta",
            "description": "Dokumentacija sadrži uputstvo koje zahteva vid (čitanje/pisanje vrednosti), "
                          "ali pacijent ima dijagnostifikovano teško oštećenje vida ili slepoću (retinopatija).",
            "evidence": [
                "Dijagnoza: H36.0 (Retinopathia diabetica) ili ekvivalent",
                "Uputstvo zahteva vizuelnu aktivnost (čitanje, pisanje, beleženje)"
            ],
            "recommendation": "Obezbediti asistenciju treće osobe ili govorni glukometar sa audio povratnom informacijom. "
                            "Alternativno: CGM (kontinuirani monitoring glukoze) sa alarmima.",
            "protocol_reference": "Vodič za dijabetes Batuta 2023, Sekcija 4.2 - Prilagođavanje terapije"
        }
    },
    {
        "patterns": [
            r"glikemij.*[0-2]\.[0-9]",
            r"glukoz.*[0-2]\.[0-9]",
            r"[0-2]\.[0-9]\s*mmol",
            r"hipoglikemij"
        ],
        "conclusion_patterns": [
            r"dobr.*praks",
            r"u skladu",
            r"pravilno",
            r"adekvatn",
            r"bez propust"
        ],
        "anomaly": {
            "type": AnomalyType.LOGICAL_INCONSISTENCY,
            "severity": SeverityLevel.CRITICAL,
            "title": "Logička nekonzistentnost - opasna hipoglikemija opisana kao 'dobra praksa'",
            "description": "Dokumentacija navodi kritično nisku glikemiju (< 2.2 mmol/L je ozbiljna hipoglikemija), "
                          "ali zaključak tvrdi da je postupanje bilo u skladu sa dobrom praksom.",
            "evidence": [
                "Glikemija ispod kritičnog praga (< 2.2 mmol/L)",
                "Zaključak pozitivno ocenjuje postupanje"
            ],
            "recommendation": "Revidirati zaključak. Prema Vodiču Batuta, glikemija < 2.2 mmol/L zahteva "
                            "hitnu intervenciju. Ako je pacijent na dugom insulinu, neophodna je hospitalizacija.",
            "protocol_reference": "Vodič Batuta za prehospitalna urgentna stanja, Hipoglikemija"
        }
    },
    {
        "patterns": [
            r"nega.*nije.*obezbe[dđ]",
            r"nije.*obezbe[dđ].*nega",
            r"bez.*nege",
            r"nega.*nedostup"
        ],
        "conclusion_patterns": [
            r"dobr.*praks",
            r"u skladu",
            r"pravilno",
            r"adekvatn"
        ],
        "anomaly": {
            "type": AnomalyType.LOGICAL_INCONSISTENCY,
            "severity": SeverityLevel.CRITICAL,
            "title": "Kontradikcija - 'nega nije obezbeđena' + 'dobra praksa'",
            "description": "Dokumentacija eksplicitno navodi da nega nije obezbeđena, "
                          "ali zaključak ocenjuje postupanje kao ispravno.",
            "evidence": [
                "Izjava: 'nega nije obezbeđena' (ili ekvivalent)",
                "Zaključak: pozitivna ocena postupanja"
            ],
            "recommendation": "Uskladiti zaključak sa činjeničnim stanjem. Ako nega zaista nije bila "
                            "obezbeđena, to ne može biti 'dobra praksa' prema bilo kom standardu.",
            "protocol_reference": None
        }
    },
    {
        "patterns": [
            r"sestra.*vodi.*ra[cč]un",
            r"[cč]lan.*porodic.*brin",
            r"srodnik.*obezbe[dđ]",
            r"suprug.*poma[zž]"
        ],
        "conflict_patterns": [
            r"[zž]ivi\s+sam",
            r"nema.*srodnik",
            r"usamljen",
            r"bez.*porodic",
            r"samo.*[zž]ivi"
        ],
        "anomaly": {
            "type": AnomalyType.DATA_CONFLICT,
            "severity": SeverityLevel.WARNING,
            "title": "Konflikt podataka - navedena nega vs. socijalni status",
            "description": "Medicinska dokumentacija navodi da će se član porodice/srodnik brinuti o pacijentu, "
                          "ali socijalni podaci ukazuju da pacijent živi sam ili nema dostupne srodnike.",
            "evidence": [
                "Otpusna lista/plan: srodnik će se brinuti",
                "Socijalni karton: živi sam/nema srodnika u mestu"
            ],
            "recommendation": "Verifikovati stvarno stanje pre otpusta. Ako pacijent zaista živi sam, "
                            "organizovati kućnu negu ili razmotriti produženi boravak.",
            "protocol_reference": None
        }
    },
    {
        "patterns": [
            r"otpu[sš]t.*sa.*gluk",
            r"otpu[sš]t.*sa.*glikemij"
        ],
        "value_patterns": [
            r"[0-3]\.[0-9]\s*mmol",
            r"glu.*[0-3]\.[0-9]"
        ],
        "anomaly": {
            "type": AnomalyType.PROTOCOL_VIOLATION,
            "severity": SeverityLevel.CRITICAL,
            "title": "Kršenje protokola - otpust sa kritičnom glikemijom",
            "description": "Pacijent je otpušten sa glikemijom koja je ispod bezbednog praga. "
                          "Prema protokolu, glikemija mora biti stabilizovana pre otpusta.",
            "evidence": [
                "Vrednost glikemije pri otpustu ispod 4.0 mmol/L"
            ],
            "recommendation": "Pacijent sa glikemijom < 4.0 mmol/L ne bi trebalo da bude otpušten "
                            "dok se vrednosti ne stabilizuju iznad 5.0 mmol/L tokom najmanje 2 sata.",
            "protocol_reference": "ADA Standards of Care 2024, Sekcija 6 - Hospitalizovani pacijenti"
        }
    }
]


def detect_anomalies(text: str, context: Optional[dict] = None) -> List[Anomaly]:
    """
    Analizira tekst i detektuje anomalije na osnovu predefinisanih obrazaca.

    U produkciji: Ovo bi bio poziv ka LLM modelu (OpenBioLLM, DeepSeek, itd.)
    """
    text_lower = text.lower()
    anomalies = []

    for pattern_set in DETECTION_PATTERNS:
        # Proveri da li postoji osnovna dijagnoza/stanje
        base_match = any(re.search(p, text_lower) for p in pattern_set["patterns"])

        if not base_match:
            continue

        # Proveri da li postoji konfliktno uputstvo/zaključak
        conflict_key = None
        for key in ["instruction_patterns", "conclusion_patterns", "conflict_patterns", "value_patterns"]:
            if key in pattern_set:
                conflict_key = key
                break

        if conflict_key and any(re.search(p, text_lower) for p in pattern_set[conflict_key]):
            anomaly_data = pattern_set["anomaly"]
            anomalies.append(Anomaly(
                type=anomaly_data["type"],
                severity=anomaly_data["severity"],
                title=anomaly_data["title"],
                description=anomaly_data["description"],
                evidence=anomaly_data["evidence"],
                recommendation=anomaly_data["recommendation"],
                protocol_reference=anomaly_data.get("protocol_reference")
            ))

    # Dodaj kontekst iz socijalnog kartona ako postoji
    if context and "social_status" in context:
        social = context["social_status"]
        if social.get("lives_alone") and "sestra" in text_lower:
            # Već pokriveno gore, ali ovo je primer kako bi se kontekst koristio
            pass

    return anomalies


@app.get("/health")
async def health():
    """Health check endpoint"""
    return {"status": "healthy", "service": "ai-mock", "version": "1.0.0"}


@app.post("/api/v1/analyze", response_model=AnalysisResponse)
async def analyze_document(request: AnalysisRequest):
    """
    Analizira medicinski dokument i detektuje anomalije.

    U produkciji bi ovo koristilo pravi LLM model.
    Za demo, koristi pattern matching sa predefinisanim odgovorima.
    """
    import time
    import uuid

    start_time = time.time()

    anomalies = detect_anomalies(request.document_text, request.patient_context)

    processing_time = int((time.time() - start_time) * 1000)

    # Simuliraj malo duže procesiranje za realističnost
    if processing_time < 100:
        import asyncio
        await asyncio.sleep(0.1)
        processing_time = 100 + processing_time

    return AnalysisResponse(
        request_id=str(uuid.uuid4()),
        timestamp=datetime.utcnow().isoformat() + "Z",
        anomalies_found=len(anomalies),
        anomalies=anomalies,
        processing_time_ms=processing_time,
        model_used="ai-mock-v1 (demo) | Production: OpenBioLLM-70B + DeepSeek-R1",
        confidence=0.95 if anomalies else 0.85
    )


@app.get("/api/v1/examples")
async def get_examples():
    """
    Vraća primere dokumenata za testiranje.
    Bazirano na stvarnim slučajevima iz dokumentacije.
    """
    return {
        "examples": [
            {
                "id": "blind_patient_instructions",
                "title": "Nemoguće uputstvo - slepi pacijent",
                "description": "Pacijent sa dijabetičkom retinopatijom oba oka dobija uputstvo da čita i zapisuje vrednosti glikemije",
                "document_text": """
OTPUSNA LISTA
Dijagnoza: E11.3 (Diabetes mellitus tip 2 sa oftalmološkim komplikacijama)
           H36.0 (Retinopathia diabetica, OU - oba oka)

Terapija pri otpustu:
- Insulin glargin 20 j SC uveče
- Metformin 1000mg 2x1

Uputstvo za pacijenta:
1. Meriti glikemiju 4x dnevno (ujutru natašte, pre ručka, pre večere, pred spavanje)
2. Da upiše sve glikemije manje od 3,5 mmol/L
3. Da upiše sve glikemije veće od 13,0 mmol/L
4. Javiti se lekaru ako glikemija padne ispod 3,5 ili poraste iznad 13,0

Kontrola za 30 dana.
                """,
                "expected_anomaly": "IMPOSSIBLE_INSTRUCTION"
            },
            {
                "id": "critical_hypoglycemia",
                "title": "Kritična hipoglikemija - logička nekonzistentnost",
                "description": "Izveštaj koji ocenjuje kao 'dobru praksu' postupanje gde je pacijent imao kritičnu hipoglikemiju",
                "document_text": """
IZVEŠTAJ SAVETNIKA ZA ZAŠTITU PRAVA PACIJENATA

Predmet: Pritužba na postupanje zdravstvene ustanove

Činjenično stanje:
- Pacijent dovezen sa glikemijom 0,7 mmol/L
- Lekar izjavio da "nega nije obezbeđena u kućnim uslovima"
- Pacijent otpušten istog dana
- Prepisan dugodjelujući insulin (glargin)

Ocena savetnika:
Na osnovu uvida u medicinsku dokumentaciju, utvrđeno je da je postupanje
zdravstvene ustanove bilo u skladu sa dobrom kliničkom praksom.

Pritužba se odbija kao neosnovana.
                """,
                "expected_anomaly": "LOGICAL_INCONSISTENCY"
            },
            {
                "id": "social_conflict",
                "title": "Konflikt podataka - sestra vs. živi sam",
                "description": "Otpusna lista navodi da će se sestra brinuti, ali pacijent živi sam",
                "document_text": """
OTPUSNA LISTA

Socijalna anamneza: Pacijent živi sam u stanu, nema srodnika u gradu.
Supruga preminula pre 3 godine. Deca žive u inostranstvu.

Plan nege nakon otpusta:
- Sestra će voditi računa o redovnom uzimanju terapije
- Kontrola kod izabranog lekara za 7 dana
- Kućna nega nije potrebna

Napomena: Pacijent je upoznat sa terapijom i otpušta se na kućno lečenje.
                """,
                "expected_anomaly": "DATA_CONFLICT"
            },
            {
                "id": "normal_document",
                "title": "Normalan dokument - bez anomalija",
                "description": "Primer ispravne medicinske dokumentacije",
                "document_text": """
OTPUSNA LISTA

Dijagnoza: E11.9 (Diabetes mellitus tip 2 bez komplikacija)

Terapija:
- Metformin 500mg 2x1

Preporuke:
- Dijeta sa smanjenim unosom ugljenih hidrata
- Fizička aktivnost 30 min dnevno
- Kontrola HbA1c za 3 meseca

Pacijent je edukovan o znacima hipoglikemije i hiperglikemije.
Porodica je uključena u plan lečenja.
                """,
                "expected_anomaly": None
            }
        ]
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=5000)
