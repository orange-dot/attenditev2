import { useState } from 'react'
import {
  ShieldCheck,
  Eye,
  EyeOff,
  Lock,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  Building2,
  FileWarning,
  Bot,
  Filter,
} from 'lucide-react'

interface DepseudonymizationRequest {
  id: string
  pseudonym_id: string
  requestor_id: string
  requestor_agency: string
  purpose: string
  legal_basis: string
  justification: string
  status: 'pending' | 'approved' | 'rejected' | 'expired'
  requested_at: string
  expires_at: string
  approved_by?: string
  approved_at?: string
}

interface PIIViolation {
  id: string
  timestamp: string
  field: string
  location: string
  blocked: boolean
  masked_value: string
  request_path: string
  request_method: string
}

interface AIAccessRequest {
  id: string
  ai_system_id: string
  requested_level: number
  purpose: string
  status: 'pending' | 'approved' | 'rejected'
  requested_at: string
  expires_at: string
}

export function Privacy() {
  const [activeTab, setActiveTab] = useState<'overview' | 'depseudo' | 'violations' | 'ai'>('overview')

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Privatnost i Pseudonimizacija</h1>
        <p className="text-gray-600">
          Upravljanje privatnošću podataka, de-pseudonimizacija i kontrola pristupa
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="flex gap-4">
          {[
            { id: 'overview', label: 'Pregled', icon: ShieldCheck },
            { id: 'depseudo', label: 'De-pseudonimizacija', icon: Eye },
            { id: 'violations', label: 'PII Povrede', icon: FileWarning },
            { id: 'ai', label: 'AI Pristup', icon: Bot },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as typeof activeTab)}
              className={`flex items-center gap-2 px-4 py-3 border-b-2 transition-colors ${
                activeTab === tab.id
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              <tab.icon className="h-4 w-4" />
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      {activeTab === 'overview' && <PrivacyOverview />}
      {activeTab === 'depseudo' && <DepseudonymizationTab />}
      {activeTab === 'violations' && <ViolationsTab />}
      {activeTab === 'ai' && <AIAccessTab />}
    </div>
  )
}

function PrivacyOverview() {
  return (
    <div className="space-y-6">
      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Pseudonimizirani subjekti"
          value="12,847"
          icon={EyeOff}
          color="blue"
          trend="+234 danas"
        />
        <StatCard
          title="Aktivni zahtevi"
          value="3"
          icon={Clock}
          color="yellow"
          trend="cekaju odobrenje"
        />
        <StatCard
          title="Blokirane povrede"
          value="47"
          icon={ShieldCheck}
          color="green"
          trend="ovaj mesec"
        />
        <StatCard
          title="AI pristupi"
          value="2"
          icon={Bot}
          color="purple"
          trend="aktivna"
        />
      </div>

      {/* Privacy Architecture Info */}
      <div className="card">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Lock className="h-5 w-5 text-primary-600" />
          Arhitektura Privatnosti
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="space-y-4">
            <div className="p-4 bg-blue-50 rounded-lg border border-blue-200">
              <h3 className="font-medium text-blue-800 mb-2">Lokalna Ustanova</h3>
              <ul className="text-sm text-blue-700 space-y-1">
                <li>• Cuva mapiranje JMBG → PseudonymID</li>
                <li>• HMAC-SHA256 pseudonimizacija</li>
                <li>• HSM za kljuceve (produkcija)</li>
                <li>• Lokalna baza nikad ne napusta ustanovu</li>
              </ul>
            </div>

            <div className="p-4 bg-green-50 rounded-lg border border-green-200">
              <h3 className="font-medium text-green-800 mb-2">Centralni Sistem</h3>
              <ul className="text-sm text-green-700 space-y-1">
                <li>• Nikad ne vidi JMBG, ime, adresu</li>
                <li>• Samo PseudonymID identifikatori</li>
                <li>• Privacy Guard blokira PII</li>
                <li>• Agregirani podaci za AI</li>
              </ul>
            </div>
          </div>

          <div className="space-y-4">
            <div className="p-4 bg-yellow-50 rounded-lg border border-yellow-200">
              <h3 className="font-medium text-yellow-800 mb-2">De-pseudonimizacija</h3>
              <ul className="text-sm text-yellow-700 space-y-1">
                <li>• Zahteva pravni osnov</li>
                <li>• Odobrenje supervizora</li>
                <li>• Vremenski ogranicen token</li>
                <li>• Kompletan audit trail</li>
              </ul>
            </div>

            <div className="p-4 bg-purple-50 rounded-lg border border-purple-200">
              <h3 className="font-medium text-purple-800 mb-2">AI Nivoi Pristupa</h3>
              <ul className="text-sm text-purple-700 space-y-1">
                <li>• Nivo 0: Samo agregirani podaci</li>
                <li>• Nivo 1: Pseudonimizirani zapisi</li>
                <li>• Nivo 2: Linkabilni (hitni slucajevi)</li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="card">
        <h2 className="text-lg font-semibold mb-4">Nedavna Aktivnost</h2>
        <div className="space-y-3">
          <ActivityItem
            icon={EyeOff}
            iconColor="blue"
            title="Novi pseudonim kreiran"
            description="PSE-a1b2c3d4... za ustanovu CSR-KG-001"
            time="pre 5 minuta"
          />
          <ActivityItem
            icon={ShieldCheck}
            iconColor="green"
            title="PII pokusaj blokiran"
            description="JMBG detektovan u response body, redaktovan"
            time="pre 12 minuta"
          />
          <ActivityItem
            icon={CheckCircle}
            iconColor="green"
            title="De-pseudo zahtev odobren"
            description="Zahtev #DP-2024-0047 odobren od supervizora"
            time="pre 1 sat"
          />
          <ActivityItem
            icon={Bot}
            iconColor="purple"
            title="AI pristup zatrazen"
            description="anomaly-detector trazi Nivo 1 pristup"
            time="pre 2 sata"
          />
        </div>
      </div>
    </div>
  )
}

function DepseudonymizationTab() {
  const [statusFilter, setStatusFilter] = useState<string>('')

  // Mock data - in real app, fetch from API
  const requests: DepseudonymizationRequest[] = [
    {
      id: 'dp-001',
      pseudonym_id: 'PSE-a1b2c3d4e5f6',
      requestor_id: 'worker-123',
      requestor_agency: 'CSR Kragujevac',
      purpose: 'Hitna zastita deteta',
      legal_basis: 'child_protection',
      justification: 'Dete je u neposrednoj opasnosti, potrebna identifikacija porodice',
      status: 'pending',
      requested_at: '2024-01-15T10:30:00Z',
      expires_at: '2024-01-16T10:30:00Z',
    },
    {
      id: 'dp-002',
      pseudonym_id: 'PSE-x9y8z7w6v5u4',
      requestor_id: 'worker-456',
      requestor_agency: 'Policija Beograd',
      purpose: 'Istraga nasilja u porodici',
      legal_basis: 'law_enforcement',
      justification: 'Sudski nalog #SN-2024-789 za pristup podacima osumnjicenog',
      status: 'approved',
      requested_at: '2024-01-14T14:00:00Z',
      expires_at: '2024-01-15T14:00:00Z',
      approved_by: 'supervisor-001',
      approved_at: '2024-01-14T15:30:00Z',
    },
    {
      id: 'dp-003',
      pseudonym_id: 'PSE-m1n2o3p4q5r6',
      requestor_id: 'worker-789',
      requestor_agency: 'Dom zdravlja Nis',
      purpose: 'Medicinska istorija',
      legal_basis: 'subject_consent',
      justification: 'Pacijent je potpisao saglasnost za uvid u istoriju',
      status: 'rejected',
      requested_at: '2024-01-13T09:00:00Z',
      expires_at: '2024-01-14T09:00:00Z',
      approved_by: 'supervisor-002',
      approved_at: '2024-01-13T11:00:00Z',
    },
  ]

  const filteredRequests = statusFilter
    ? requests.filter((r) => r.status === statusFilter)
    : requests

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-gray-500" />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="input w-48"
          >
            <option value="">Svi statusi</option>
            <option value="pending">Na cekanju</option>
            <option value="approved">Odobreni</option>
            <option value="rejected">Odbijeni</option>
            <option value="expired">Istekli</option>
          </select>
        </div>
      </div>

      {/* Requests */}
      <div className="space-y-4">
        {filteredRequests.map((request) => (
          <DepseudoRequestCard key={request.id} request={request} />
        ))}
      </div>

      {/* Info */}
      <div className="card bg-yellow-50 border-yellow-200">
        <div className="flex items-start gap-3">
          <AlertTriangle className="h-5 w-5 text-yellow-600 mt-0.5" />
          <div>
            <h3 className="font-medium text-yellow-800">Vazno</h3>
            <p className="text-sm text-yellow-700 mt-1">
              De-pseudonimizacija otkriva pravi identitet osobe. Svaki pristup se evidentira
              i moze se koristiti kao dokaz u slucaju zloupotrebe. Token za pristup vazi
              maksimalno 1 sat i moze se koristiti najvise 3 puta.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

function DepseudoRequestCard({ request }: { request: DepseudonymizationRequest }) {
  const statusConfig = {
    pending: { icon: Clock, color: 'yellow', label: 'Na cekanju' },
    approved: { icon: CheckCircle, color: 'green', label: 'Odobreno' },
    rejected: { icon: XCircle, color: 'red', label: 'Odbijeno' },
    expired: { icon: Clock, color: 'gray', label: 'Isteklo' },
  }

  const legalBasisLabels: Record<string, string> = {
    court_order: 'Sudski nalog',
    life_threat: 'Pretnja po zivot',
    child_protection: 'Zastita deteta',
    law_enforcement: 'Istraga',
    subject_consent: 'Saglasnost subjekta',
  }

  const config = statusConfig[request.status]
  const StatusIcon = config.icon

  return (
    <div className="card">
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-4">
          <div className={`p-2 rounded-lg bg-${config.color}-100`}>
            <StatusIcon className={`h-5 w-5 text-${config.color}-600`} />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-medium">{request.purpose}</span>
              <span className={`badge bg-${config.color}-100 text-${config.color}-700`}>
                {config.label}
              </span>
            </div>
            <p className="text-sm text-gray-600 mt-1">
              Pseudonym: <span className="font-mono">{request.pseudonym_id}</span>
            </p>
            <p className="text-sm text-gray-600">
              Pravni osnov: <span className="font-medium">{legalBasisLabels[request.legal_basis]}</span>
            </p>
            <p className="text-sm text-gray-500 mt-2">{request.justification}</p>
          </div>
        </div>

        <div className="text-right text-sm">
          <div className="flex items-center gap-1 text-gray-500">
            <Building2 className="h-3 w-3" />
            {request.requestor_agency}
          </div>
          <div className="text-gray-400 mt-1">
            {new Date(request.requested_at).toLocaleString('sr-RS')}
          </div>
        </div>
      </div>

      {request.status === 'pending' && (
        <div className="mt-4 pt-4 border-t border-gray-100 flex justify-end gap-2">
          <button className="btn btn-secondary text-red-600 border-red-200 hover:bg-red-50">
            <XCircle className="h-4 w-4" />
            Odbij
          </button>
          <button className="btn btn-primary">
            <CheckCircle className="h-4 w-4" />
            Odobri
          </button>
        </div>
      )}
    </div>
  )
}

function ViolationsTab() {
  // Mock data
  const violations: PIIViolation[] = [
    {
      id: 'v-001',
      timestamp: '2024-01-15T11:23:45Z',
      field: 'jmbg',
      location: 'response_body:/api/v1/cases/123',
      blocked: true,
      masked_value: '0101990******',
      request_path: '/api/v1/cases/123',
      request_method: 'GET',
    },
    {
      id: 'v-002',
      timestamp: '2024-01-15T10:15:30Z',
      field: 'email',
      location: 'response_body:/api/v1/enrichment',
      blocked: true,
      masked_value: 'jo***@gmail.com',
      request_path: '/api/v1/enrichment',
      request_method: 'POST',
    },
    {
      id: 'v-003',
      timestamp: '2024-01-15T09:45:00Z',
      field: 'phone',
      location: 'request_body:/api/v1/notifications',
      blocked: true,
      masked_value: '***-***-4567',
      request_path: '/api/v1/notifications',
      request_method: 'POST',
    },
  ]

  return (
    <div className="space-y-4">
      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="card bg-green-50 border-green-200">
          <div className="flex items-center gap-3">
            <ShieldCheck className="h-8 w-8 text-green-600" />
            <div>
              <div className="text-2xl font-bold text-green-700">47</div>
              <div className="text-sm text-green-600">Blokirano ovog meseca</div>
            </div>
          </div>
        </div>
        <div className="card bg-yellow-50 border-yellow-200">
          <div className="flex items-center gap-3">
            <AlertTriangle className="h-8 w-8 text-yellow-600" />
            <div>
              <div className="text-2xl font-bold text-yellow-700">3</div>
              <div className="text-sm text-yellow-600">Detektovano danas</div>
            </div>
          </div>
        </div>
        <div className="card bg-blue-50 border-blue-200">
          <div className="flex items-center gap-3">
            <EyeOff className="h-8 w-8 text-blue-600" />
            <div>
              <div className="text-2xl font-bold text-blue-700">100%</div>
              <div className="text-sm text-blue-600">Stopa blokiranja</div>
            </div>
          </div>
        </div>
      </div>

      {/* Violations List */}
      <div className="card">
        <h3 className="font-medium mb-4">Nedavne PII Povrede</h3>
        <div className="space-y-3">
          {violations.map((v) => (
            <div key={v.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <div className="flex items-center gap-3">
                <div className={`p-2 rounded ${v.blocked ? 'bg-green-100' : 'bg-red-100'}`}>
                  {v.blocked ? (
                    <ShieldCheck className="h-4 w-4 text-green-600" />
                  ) : (
                    <AlertTriangle className="h-4 w-4 text-red-600" />
                  )}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium uppercase text-sm">{v.field}</span>
                    <span className="badge bg-gray-200 text-gray-600">{v.request_method}</span>
                  </div>
                  <div className="text-sm text-gray-500">
                    <span className="font-mono">{v.request_path}</span>
                  </div>
                  <div className="text-xs text-gray-400 mt-1">
                    Maskirano: <span className="font-mono">{v.masked_value}</span>
                  </div>
                </div>
              </div>
              <div className="text-sm text-gray-500">
                {new Date(v.timestamp).toLocaleString('sr-RS')}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function AIAccessTab() {
  // Mock data
  const aiRequests: AIAccessRequest[] = [
    {
      id: 'ai-001',
      ai_system_id: 'anomaly-detector',
      requested_level: 1,
      purpose: 'Detekcija anomalija u medicinskoj dokumentaciji',
      status: 'approved',
      requested_at: '2024-01-14T08:00:00Z',
      expires_at: '2024-01-15T08:00:00Z',
    },
    {
      id: 'ai-002',
      ai_system_id: 'risk-assessment',
      requested_level: 0,
      purpose: 'Procena rizika za CSR slucajeve',
      status: 'approved',
      requested_at: '2024-01-10T12:00:00Z',
      expires_at: '2024-01-17T12:00:00Z',
    },
    {
      id: 'ai-003',
      ai_system_id: 'pattern-analyzer',
      requested_level: 2,
      purpose: 'Hitna analiza obrazaca zlostavljanja',
      status: 'pending',
      requested_at: '2024-01-15T09:00:00Z',
      expires_at: '2024-01-16T09:00:00Z',
    },
  ]

  const levelLabels = ['Agregirano', 'Pseudonimizirano', 'Linkabilno']
  const levelColors = ['blue', 'yellow', 'red']

  return (
    <div className="space-y-4">
      {/* Access Levels Explanation */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="card border-blue-200 bg-blue-50">
          <h4 className="font-medium text-blue-800">Nivo 0: Agregirano</h4>
          <p className="text-sm text-blue-700 mt-1">
            Samo statisticki podaci, bez individualnih zapisa
          </p>
        </div>
        <div className="card border-yellow-200 bg-yellow-50">
          <h4 className="font-medium text-yellow-800">Nivo 1: Pseudonimizirano</h4>
          <p className="text-sm text-yellow-700 mt-1">
            Individualni zapisi sa pseudonimima, zahteva odobrenje
          </p>
        </div>
        <div className="card border-red-200 bg-red-50">
          <h4 className="font-medium text-red-800">Nivo 2: Linkabilno</h4>
          <p className="text-sm text-red-700 mt-1">
            Pristup pravim identitetima - samo hitni slucajevi
          </p>
        </div>
      </div>

      {/* AI Systems */}
      <div className="card">
        <h3 className="font-medium mb-4">AI Sistemi i Pristupi</h3>
        <div className="space-y-3">
          {aiRequests.map((req) => (
            <div key={req.id} className="flex items-center justify-between p-4 border rounded-lg">
              <div className="flex items-center gap-3">
                <Bot className="h-6 w-6 text-purple-600" />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{req.ai_system_id}</span>
                    <span className={`badge bg-${levelColors[req.requested_level]}-100 text-${levelColors[req.requested_level]}-700`}>
                      Nivo {req.requested_level}: {levelLabels[req.requested_level]}
                    </span>
                  </div>
                  <p className="text-sm text-gray-600">{req.purpose}</p>
                </div>
              </div>
              <div className="text-right">
                <div className={`badge ${
                  req.status === 'approved' ? 'bg-green-100 text-green-700' :
                  req.status === 'pending' ? 'bg-yellow-100 text-yellow-700' :
                  'bg-red-100 text-red-700'
                }`}>
                  {req.status === 'approved' ? 'Aktivan' :
                   req.status === 'pending' ? 'Na cekanju' : 'Odbijen'}
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  Istice: {new Date(req.expires_at).toLocaleDateString('sr-RS')}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

// Helper Components

function StatCard({
  title,
  value,
  icon: Icon,
  color,
  trend,
}: {
  title: string
  value: string
  icon: React.ElementType
  color: 'blue' | 'yellow' | 'green' | 'purple'
  trend: string
}) {
  const colorClasses = {
    blue: 'bg-blue-50 text-blue-600',
    yellow: 'bg-yellow-50 text-yellow-600',
    green: 'bg-green-50 text-green-600',
    purple: 'bg-purple-50 text-purple-600',
  }

  return (
    <div className="card">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-600">{title}</p>
          <p className="text-2xl font-bold mt-1">{value}</p>
          <p className="text-xs text-gray-500 mt-1">{trend}</p>
        </div>
        <div className={`p-3 rounded-lg ${colorClasses[color]}`}>
          <Icon className="h-6 w-6" />
        </div>
      </div>
    </div>
  )
}

function ActivityItem({
  icon: Icon,
  iconColor,
  title,
  description,
  time,
}: {
  icon: React.ElementType
  iconColor: string
  title: string
  description: string
  time: string
}) {
  return (
    <div className="flex items-start gap-3 p-3 rounded-lg hover:bg-gray-50">
      <div className={`p-2 rounded-lg bg-${iconColor}-100`}>
        <Icon className={`h-4 w-4 text-${iconColor}-600`} />
      </div>
      <div className="flex-1 min-w-0">
        <p className="font-medium text-sm">{title}</p>
        <p className="text-sm text-gray-500 truncate">{description}</p>
      </div>
      <span className="text-xs text-gray-400 whitespace-nowrap">{time}</span>
    </div>
  )
}
