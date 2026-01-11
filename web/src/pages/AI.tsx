import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import {
  Brain,
  AlertTriangle,
  AlertCircle,
  Info,
  CheckCircle,
  Loader2,
  FileText,
  Server,
  Database,
  Lock,
  Zap,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { api } from '../api/client'
import type { AnalysisResponse, Anomaly, AIExample } from '../types'

export function AI() {
  const [documentText, setDocumentText] = useState('')
  const [analysisResult, setAnalysisResult] = useState<AnalysisResponse | null>(null)
  const [activeTab, setActiveTab] = useState<'analyze' | 'architecture'>('analyze')

  const { data: examples } = useQuery({
    queryKey: ['ai-examples'],
    queryFn: () => api.ai.examples(),
  })

  const analyzeMutation = useMutation({
    mutationFn: (text: string) => api.ai.analyze({ document_text: text, document_type: 'medical' }),
    onSuccess: (data) => setAnalysisResult(data as AnalysisResponse),
  })

  const loadExample = (example: AIExample) => {
    setDocumentText(example.document_text.trim())
    setAnalysisResult(null)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">AI Detekcija Anomalija</h1>
          <p className="text-gray-600">Automatska analiza medicinske dokumentacije</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setActiveTab('analyze')}
            className={`btn ${activeTab === 'analyze' ? 'btn-primary' : 'btn-secondary'}`}
          >
            <Brain className="h-4 w-4" />
            Analiza
          </button>
          <button
            onClick={() => setActiveTab('architecture')}
            className={`btn ${activeTab === 'architecture' ? 'btn-primary' : 'btn-secondary'}`}
          >
            <Server className="h-4 w-4" />
            Arhitektura
          </button>
        </div>
      </div>

      {activeTab === 'analyze' ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {/* Input Panel */}
          <div className="card">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Medicinski Dokument</h2>

            <textarea
              value={documentText}
              onChange={(e) => setDocumentText(e.target.value)}
              placeholder="Unesite tekst medicinske dokumentacije za analizu..."
              className="input min-h-[300px] font-mono text-sm"
            />

            {/* Examples */}
            <div className="mt-4">
              <p className="text-sm font-medium text-gray-700 mb-2">Test primeri:</p>
              <div className="flex flex-wrap gap-2">
                {(examples?.examples as AIExample[] || []).map((example) => (
                  <button
                    key={example.id}
                    onClick={() => loadExample(example)}
                    className="btn btn-secondary text-xs"
                  >
                    {example.title}
                  </button>
                ))}
              </div>
            </div>

            {/* Actions */}
            <div className="mt-4 flex gap-3">
              <button
                onClick={() => analyzeMutation.mutate(documentText)}
                disabled={!documentText.trim() || analyzeMutation.isPending}
                className="btn btn-primary"
              >
                {analyzeMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Analiziranje...
                  </>
                ) : (
                  <>
                    <Zap className="h-4 w-4" />
                    Analiziraj
                  </>
                )}
              </button>
              <button
                onClick={() => {
                  setDocumentText('')
                  setAnalysisResult(null)
                }}
                className="btn btn-secondary"
              >
                Obriši
              </button>
            </div>
          </div>

          {/* Results Panel */}
          <div className="card">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Rezultat Analize</h2>

            {!analysisResult ? (
              <div className="flex flex-col items-center justify-center h-[300px] text-gray-500">
                <FileText className="h-12 w-12 mb-3 opacity-50" />
                <p>Unesite dokument i kliknite "Analiziraj"</p>
              </div>
            ) : analysisResult.anomalies_found === 0 ? (
              <div className="flex flex-col items-center justify-center h-[300px]">
                <CheckCircle className="h-16 w-16 text-green-500 mb-3" />
                <h3 className="text-lg font-semibold text-green-700">Nema detektovanih anomalija</h3>
                <p className="text-sm text-gray-500 mt-1">
                  Dokument ne sadrži prepoznatljive logičke nekonzistentnosti.
                </p>
              </div>
            ) : (
              <div className="space-y-4">
                {analysisResult.anomalies.map((anomaly, index) => (
                  <AnomalyCard key={index} anomaly={anomaly} />
                ))}
              </div>
            )}

            {/* Meta */}
            {analysisResult && (
              <div className="mt-6 pt-4 border-t border-gray-200">
                <div className="flex flex-wrap gap-4 text-xs text-gray-500">
                  <span>
                    <strong>Model:</strong> {analysisResult.model_used}
                  </span>
                  <span>
                    <strong>Vreme:</strong> {analysisResult.processing_time_ms}ms
                  </span>
                  <span>
                    <strong>Pouzdanost:</strong> {(analysisResult.confidence * 100).toFixed(0)}%
                  </span>
                </div>
              </div>
            )}
          </div>
        </div>
      ) : (
        <ArchitectureView />
      )}

      {/* Info Cards */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
        <div className="card">
          <div className="flex items-center gap-3 mb-3">
            <div className="rounded-lg bg-blue-100 p-2">
              <Brain className="h-5 w-5 text-blue-600" />
            </div>
            <h3 className="font-semibold text-gray-900">Open-Source Modeli</h3>
          </div>
          <p className="text-sm text-gray-600">
            OpenBioLLM-70B + DeepSeek-R1 za medicinsku analizu. Mogu se hostovati lokalno u državnom data centru.
          </p>
        </div>

        <div className="card">
          <div className="flex items-center gap-3 mb-3">
            <div className="rounded-lg bg-green-100 p-2">
              <Lock className="h-5 w-5 text-green-600" />
            </div>
            <h3 className="font-semibold text-gray-900">Zero-Knowledge</h3>
          </div>
          <p className="text-sm text-gray-600">
            Podaci se procesiraju u RAM-u i nikad ne pišu na disk. Centralni sistem ne čuva kopije dokumenata.
          </p>
        </div>

        <div className="card">
          <div className="flex items-center gap-3 mb-3">
            <div className="rounded-lg bg-purple-100 p-2">
              <Database className="h-5 w-5 text-purple-600" />
            </div>
            <h3 className="font-semibold text-gray-900">Decentralizovano</h3>
          </div>
          <p className="text-sm text-gray-600">
            Svaka ustanova čuva svoje podatke lokalno. Veće ustanove mogu imati sopstveni LLM.
          </p>
        </div>
      </div>
    </div>
  )
}

function AnomalyCard({ anomaly }: { anomaly: Anomaly }) {
  const severityConfig: Record<string, {
    bg: string
    border: string
    icon: LucideIcon
    iconColor: string
    badge: string
  }> = {
    critical: {
      bg: 'bg-red-50',
      border: 'border-red-200',
      icon: AlertTriangle,
      iconColor: 'text-red-500',
      badge: 'badge-danger',
    },
    warning: {
      bg: 'bg-yellow-50',
      border: 'border-yellow-200',
      icon: AlertCircle,
      iconColor: 'text-yellow-500',
      badge: 'badge-warning',
    },
    info: {
      bg: 'bg-blue-50',
      border: 'border-blue-200',
      icon: Info,
      iconColor: 'text-blue-500',
      badge: 'badge-info',
    },
  }

  const config = severityConfig[anomaly.severity] || severityConfig.info
  const Icon = config.icon

  const typeLabels: Record<string, string> = {
    impossible_instruction: 'Nemoguće uputstvo',
    logical_inconsistency: 'Logička nekonzistentnost',
    data_conflict: 'Konflikt podataka',
    protocol_violation: 'Kršenje protokola',
  }

  return (
    <div className={`rounded-lg border ${config.border} ${config.bg} overflow-hidden`}>
      <div className="p-4">
        <div className="flex items-start gap-3">
          <Icon className={`h-5 w-5 ${config.iconColor} mt-0.5`} />
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-1">
              <h4 className="font-semibold text-gray-900">{anomaly.title}</h4>
              <span className={`badge ${config.badge}`}>
                {anomaly.severity.toUpperCase()}
              </span>
            </div>
            <p className="text-xs text-gray-500 mb-2">{typeLabels[anomaly.type]}</p>
            <p className="text-sm text-gray-700">{anomaly.description}</p>

            {/* Evidence */}
            <div className="mt-3">
              <p className="text-xs font-medium text-gray-500 uppercase mb-1">Dokazi</p>
              <ul className="text-sm text-gray-600 space-y-1">
                {anomaly.evidence.map((item, i) => (
                  <li key={i} className="flex items-start gap-2">
                    <span className="text-gray-400">•</span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>

            {/* Recommendation */}
            <div className="mt-3 p-3 bg-green-50 rounded-lg border border-green-200">
              <p className="text-xs font-medium text-green-700 uppercase mb-1">Preporuka</p>
              <p className="text-sm text-green-800">{anomaly.recommendation}</p>
            </div>

            {anomaly.protocol_reference && (
              <p className="mt-2 text-xs text-gray-500">
                <strong>Referenca:</strong> {anomaly.protocol_reference}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function ArchitectureView() {
  return (
    <div className="space-y-6">
      {/* Option A: With Local LLM */}
      <div className="card">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          Opcija A: Ustanova SA sopstvenim LLM-om
        </h3>
        <p className="text-sm text-gray-600 mb-4">
          Za veće ustanove (klinički centri, velike bolnice) koje imaju IT kapacitete
        </p>
        <div className="bg-gray-50 rounded-lg p-6 font-mono text-sm">
          <pre className="text-gray-700 whitespace-pre-wrap">
{`┌─────────────────────────────────────────────────────────────────┐
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
│  │      Federativna mreža (mTLS/REST)      │                   │
│  └─────────────────────────────────────────┘                   │
│                                                                 │
│  ═══════════════════════════════════════════                   │
│  PUNI PODACI NIKADA NE NAPUŠTAJU USTANOVU                      │
│  ═══════════════════════════════════════════                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘`}
          </pre>
        </div>
        <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
          <div className="bg-blue-50 rounded-lg p-3">
            <p className="font-medium text-blue-800">Hardware</p>
            <p className="text-blue-600">RTX 4090 (24GB) + 32GB RAM</p>
          </div>
          <div className="bg-green-50 rounded-lg p-3">
            <p className="font-medium text-green-800">Cena</p>
            <p className="text-green-600">~2,500 EUR</p>
          </div>
          <div className="bg-purple-50 rounded-lg p-3">
            <p className="font-medium text-purple-800">Model</p>
            <p className="text-purple-600">Hippo-7B</p>
          </div>
        </div>
      </div>

      {/* Option B: Without Local LLM */}
      <div className="card">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          Opcija B: Ustanova BEZ sopstvenog LLM-a
        </h3>
        <p className="text-sm text-gray-600 mb-4">
          Za manje ustanove (domovi zdravlja, CSR-ovi, gerontološki centri)
        </p>
        <div className="bg-gray-50 rounded-lg p-6 font-mono text-sm">
          <pre className="text-gray-700 whitespace-pre-wrap">
{`┌─────────────────────────────────────────────────────────────────┐
│                      DOM ZDRAVLJA                               │
│  ┌─────────────────┐                                           │
│  │  Lokalna baza   │  Podaci ostaju OVDE                       │
│  └────────┬────────┘                                           │
│           │ (1) Zahtev za analizu                              │
│           │     [šalje se SAMO dokument za analizu]            │
└───────────┼─────────────────────────────────────────────────────┘
            │ Enkriptovani kanal (mTLS)
            ▼
┌───────────────────────────────────────────────────────────────────┐
│                    DATA CENTAR KRAGUJEVAC                         │
│  ┌─────────────────┐    ┌─────────────────┐                      │
│  │  OpenBioLLM-70B │    │   DeepSeek-R1   │                      │
│  └────────┬────────┘    └────────┬────────┘                      │
│           └──────────┬───────────┘                                │
│              ┌───────▼───────┐                                    │
│              │   PROCESIRA   │                                    │
│              │   ODMAH BRIŠE │◄── NEMA ČUVANJA!                   │
│              └───────┬───────┘                                    │
│           (2) Vraća SAMO rezultat analize                        │
└──────────────────────┼────────────────────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                      DOM ZDRAVLJA                               │
│  ┌─────────────────┐                                           │
│  │  Lokalna baza   │◄── Rezultat se čuva LOKALNO               │
│  └─────────────────┘                                           │
└─────────────────────────────────────────────────────────────────┘`}
          </pre>
        </div>
      </div>

      {/* Comparison Table */}
      <div className="card">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Poređenje</h3>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead>
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Aspekt</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Centralizovano</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Decentralizovano (naš predlog)</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">Rizik curenja podataka</td>
                <td className="px-4 py-3 text-sm text-red-600">Visok - sve na jednom mestu</td>
                <td className="px-4 py-3 text-sm text-green-600">Nizak - podaci distribuirani</td>
              </tr>
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">Single point of failure</td>
                <td className="px-4 py-3 text-sm text-red-600">Da</td>
                <td className="px-4 py-3 text-sm text-green-600">Ne</td>
              </tr>
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">GDPR usklađenost</td>
                <td className="px-4 py-3 text-sm text-yellow-600">Komplikovano</td>
                <td className="px-4 py-3 text-sm text-green-600">Jednostavno</td>
              </tr>
              <tr>
                <td className="px-4 py-3 text-sm text-gray-900">Autonomija ustanova</td>
                <td className="px-4 py-3 text-sm text-red-600">Niska</td>
                <td className="px-4 py-3 text-sm text-green-600">Visoka</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
