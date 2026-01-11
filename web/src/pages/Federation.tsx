import { useQuery } from '@tanstack/react-query'
import {
  Globe,
  Building2,
  Shield,
  CheckCircle,
  XCircle,
  Server,
  Link as LinkIcon,
  Clock,
  Key,
} from 'lucide-react'
import { api } from '../api/client'

import type { FederatedAgency } from '../types'

interface FederationService {
  name: string
  endpoint: string
  status: string
  version?: string
}

export function Federation() {
  const { data: agencies, isLoading: loadingAgencies } = useQuery({
    queryKey: ['federation-agencies'],
    queryFn: () => api.federation.agencies(),
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Federacija</h1>
          <p className="text-gray-600">Pilot projekat: Kikinda - Povezane javne službe</p>
        </div>
        <div className="flex items-center gap-2 px-4 py-2 bg-accent-50 rounded-lg border border-accent-200">
          <Globe className="h-5 w-5 text-accent-600" />
          <span className="text-sm font-medium text-accent-700">Kikinda Pilot</span>
        </div>
      </div>

      {/* Status Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatusCard
          icon={Shield}
          label="Trust Authority"
          value="Aktivan"
          color="green"
          description="Centralni CA server"
        />
        <StatusCard
          icon={Building2}
          label="Povezane agencije"
          value={agencies?.data?.length ?? '-'}
          color="blue"
          description="Registrovane u federaciji"
        />
        <StatusCard
          icon={Server}
          label="Servisi"
          value="Online"
          color="green"
          description="Svi servisi operativni"
        />
      </div>

      {/* Architecture Diagram */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center gap-2">
          <Globe className="h-5 w-5 text-gray-500" />
          Arhitektura federacije
        </h2>

        <div className="bg-gray-50 rounded-lg p-6">
          <div className="flex flex-col items-center">
            {/* Trust Authority */}
            <div className="bg-amber-100 border-2 border-amber-400 rounded-lg p-4 text-center w-64">
              <Key className="h-8 w-8 text-amber-600 mx-auto mb-2" />
              <p className="font-semibold text-amber-800">Trust Authority</p>
              <p className="text-xs text-amber-600">Izdaje sertifikate</p>
            </div>

            {/* Connection lines */}
            <div className="flex items-center justify-center my-4 w-full">
              <div className="flex-1 border-t-2 border-dashed border-gray-300"></div>
              <div className="px-4">
                <LinkIcon className="h-5 w-5 text-gray-400" />
              </div>
              <div className="flex-1 border-t-2 border-dashed border-gray-300"></div>
            </div>

            {/* Kikinda Pilot Agencies */}
            <div className="flex flex-wrap justify-center gap-4">
              <AgencyNode name="CSR Kikinda" type="SOCIAL_WELFARE" status="online" />
              <AgencyNode name="DZ Kikinda" type="HEALTHCARE" status="online" />
              <AgencyNode name="PU Kikinda" type="POLICE" status="online" />
              <AgencyNode name="OŠ Vuk Karadžić" type="EDUCATION" status="online" />
              <AgencyNode name="Opština Kikinda" type="GOVERNMENT" status="online" />
            </div>

            {/* Legend */}
            <div className="mt-6 flex items-center gap-6 text-sm text-gray-500">
              <div className="flex items-center gap-2">
                <div className="h-3 w-3 rounded-full bg-green-500"></div>
                <span>Online</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-3 w-3 rounded-full bg-yellow-500"></div>
                <span>Sinhronizacija</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-3 w-3 rounded-full bg-red-500"></div>
                <span>Offline</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Federated Agencies List */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Povezane agencije</h2>

        {loadingAgencies ? (
          <div className="text-center py-8 text-gray-500">Učitavanje...</div>
        ) : agencies?.data && agencies.data.length > 0 ? (
          <div className="space-y-4">
            {agencies.data.map((agency: FederatedAgency) => (
              <FederatedAgencyCard key={agency.id} agency={agency} />
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-gray-500">
            Nema registrovanih agencija u federaciji
          </div>
        )}
      </div>

      {/* Services Status */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Servisni katalog</h2>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <ServiceCard
            name="Case Coordination"
            endpoint="/api/v1/cases"
            status="active"
            version="1.0"
          />
          <ServiceCard
            name="Document Exchange"
            endpoint="/api/v1/documents"
            status="active"
            version="1.0"
          />
          <ServiceCard
            name="AI Analysis"
            endpoint="/api/v1/ai"
            status="active"
            version="1.0"
          />
          <ServiceCard
            name="Audit Trail"
            endpoint="/api/v1/audit"
            status="active"
            version="1.0"
          />
        </div>
      </div>

      {/* Security Info */}
      <div className="card bg-blue-50 border-blue-200">
        <div className="flex items-start gap-3">
          <Shield className="h-5 w-5 text-blue-600 mt-0.5" />
          <div>
            <h3 className="font-medium text-blue-800">Sigurnosni model</h3>
            <ul className="text-sm text-blue-700 mt-2 space-y-1">
              <li>• mTLS (Mutual TLS) za svu inter-agencijsku komunikaciju</li>
              <li>• X.509 sertifikati izdati od centralnog CA</li>
              <li>• Automatska rotacija sertifikata svakih 90 dana</li>
              <li>• Federativni identitet - svaka agencija upravlja svojim korisnicima</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  )
}

function StatusCard({
  icon: Icon,
  label,
  value,
  color,
  description,
}: {
  icon: React.ElementType
  label: string
  value: string | number
  color: 'green' | 'blue' | 'yellow' | 'red'
  description: string
}) {
  const colorClasses = {
    green: 'bg-green-100 text-green-600',
    blue: 'bg-blue-100 text-blue-600',
    yellow: 'bg-yellow-100 text-yellow-600',
    red: 'bg-red-100 text-red-600',
  }

  return (
    <div className="card">
      <div className="flex items-center gap-3">
        <div className={`rounded-lg p-2 ${colorClasses[color]}`}>
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <p className="text-sm text-gray-600">{label}</p>
          <p className="text-xl font-semibold text-gray-900">{value}</p>
          <p className="text-xs text-gray-500">{description}</p>
        </div>
      </div>
    </div>
  )
}

function AgencyNode({
  name,
  type,
  status,
}: {
  name: string
  type: string
  status: 'online' | 'syncing' | 'offline'
}) {
  const statusColors = {
    online: 'border-green-400 bg-green-50',
    syncing: 'border-yellow-400 bg-yellow-50',
    offline: 'border-red-400 bg-red-50',
  }

  const typeColors: Record<string, string> = {
    SOCIAL_WELFARE: 'text-blue-600',
    HEALTHCARE: 'text-green-600',
    EDUCATION: 'text-purple-600',
    POLICE: 'text-slate-600',
    GOVERNMENT: 'text-red-600',
  }

  return (
    <div className={`border-2 rounded-lg p-3 text-center w-40 ${statusColors[status]}`}>
      <Building2 className={`h-6 w-6 mx-auto mb-1 ${typeColors[type] || 'text-gray-600'}`} />
      <p className="font-medium text-sm text-gray-800">{name}</p>
      <div className="flex items-center justify-center gap-1 mt-1">
        <div
          className={`h-2 w-2 rounded-full ${
            status === 'online' ? 'bg-green-500' : status === 'syncing' ? 'bg-yellow-500' : 'bg-red-500'
          }`}
        ></div>
        <span className="text-xs text-gray-500 capitalize">{status}</span>
      </div>
    </div>
  )
}

// Get agency type from code
function getAgencyType(code: string): { type: string; color: string } {
  // Nacionalni nivo
  if (code === 'VLADA-RS') return { type: 'Vlada', color: 'text-red-700 bg-red-100' }
  if (code === 'RZS') return { type: 'Statistika', color: 'text-indigo-600 bg-indigo-100' }
  if (code.startsWith('MIN')) return { type: 'Ministarstvo', color: 'text-red-600 bg-red-50' }
  if (code === 'MUP') return { type: 'Ministarstvo', color: 'text-red-600 bg-red-50' }

  // Pravosuđe
  if (code.startsWith('SUD')) return { type: 'Sud', color: 'text-violet-600 bg-violet-100' }

  // Lokalna samouprava
  if (code.startsWith('OU')) return { type: 'Opština', color: 'text-orange-600 bg-orange-100' }

  // Socijalna zaštita
  if (code.startsWith('CSR')) return { type: 'Socijalna zaštita', color: 'text-blue-600 bg-blue-100' }
  if (code.startsWith('GC')) return { type: 'Gerontologija', color: 'text-amber-600 bg-amber-100' }
  if (code.startsWith('NSZ')) return { type: 'Zapošljavanje', color: 'text-cyan-600 bg-cyan-100' }

  // Zdravstvo
  if (code.startsWith('OB')) return { type: 'Bolnica', color: 'text-emerald-600 bg-emerald-100' }
  if (code.startsWith('DZ')) return { type: 'Dom zdravlja', color: 'text-green-600 bg-green-100' }
  if (code.startsWith('APO')) return { type: 'Apoteka', color: 'text-teal-600 bg-teal-100' }

  // Bezbednost
  if (code.startsWith('PU') && !code.includes('DU')) return { type: 'Policija', color: 'text-slate-600 bg-slate-100' }

  // Obrazovanje
  if (code.startsWith('PU-DU')) return { type: 'Predškolsko', color: 'text-pink-600 bg-pink-100' }
  if (code.startsWith('OS') || code.startsWith('GIM')) return { type: 'Obrazovanje', color: 'text-purple-600 bg-purple-100' }

  return { type: 'Ostalo', color: 'text-gray-600 bg-gray-100' }
}

function FederatedAgencyCard({ agency }: { agency: FederatedAgency }) {
  const agencyType = getAgencyType(agency.code)

  return (
    <div className="p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-3">
          <div className={`rounded-lg p-2 ${agencyType.color.split(' ')[1]}`}>
            <Building2 className={`h-5 w-5 ${agencyType.color.split(' ')[0]}`} />
          </div>
          <div>
            <h3 className="font-medium text-gray-900">{agency.name}</h3>
            <div className="flex items-center gap-2 mt-1">
              <span className={`text-xs px-2 py-0.5 rounded-full ${agencyType.color}`}>
                {agencyType.type}
              </span>
              <span className="text-xs text-gray-400 font-mono">{agency.code}</span>
            </div>
            {agency.gateway_url && (
              <p className="text-xs text-gray-400 font-mono mt-1">{agency.gateway_url}</p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          {agency.status === 'active' ? (
            <span className="badge bg-green-100 text-green-700">
              <CheckCircle className="h-3 w-3 mr-1" />
              Aktivna
            </span>
          ) : agency.status === 'suspended' ? (
            <span className="badge bg-yellow-100 text-yellow-700">
              <Clock className="h-3 w-3 mr-1" />
              Suspendovana
            </span>
          ) : (
            <span className="badge bg-red-100 text-red-700">
              <XCircle className="h-3 w-3 mr-1" />
              Opozvana
            </span>
          )}
        </div>
      </div>

      {agency.last_seen_at && (
        <div className="mt-3 pt-3 border-t border-gray-100 flex items-center gap-2 text-sm text-gray-500">
          <Clock className="h-4 w-4" />
          <span>Registrovana: {new Date(agency.registered_at).toLocaleString('sr-RS')}</span>
        </div>
      )}
    </div>
  )
}

function ServiceCard({
  name,
  endpoint,
  status,
  version,
}: FederationService) {
  return (
    <div className="p-4 border border-gray-200 rounded-lg">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Server className="h-4 w-4 text-gray-500" />
          <span className="font-medium text-gray-900">{name}</span>
        </div>
        {status === 'active' ? (
          <span className="badge bg-green-100 text-green-700">Aktivan</span>
        ) : (
          <span className="badge bg-gray-100 text-gray-700">Neaktivan</span>
        )}
      </div>
      <div className="mt-2 text-sm">
        <p className="text-gray-500 font-mono">{endpoint}</p>
        {version && <p className="text-gray-400 text-xs mt-1">v{version}</p>}
      </div>
    </div>
  )
}
