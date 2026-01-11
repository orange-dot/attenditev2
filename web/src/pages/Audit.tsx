import { useState, useEffect } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import {
  Search,
  Filter,
  Shield,
  CheckCircle,
  XCircle,
  AlertTriangle,
  RefreshCw,
  Calendar,
  X,
  Link,
} from 'lucide-react'
import { api } from '../api/client'
import type { AuditEntry, AuditVerifyEntry } from '../types'

export function Audit() {
  const [search, setSearch] = useState('')
  const [actionFilter, setActionFilter] = useState<string>('')
  const [resourceFilter, setResourceFilter] = useState<string>('')
  const [dateFrom, setDateFrom] = useState<string>('')
  const [dateTo, setDateTo] = useState<string>('')
  const [showVerifyModal, setShowVerifyModal] = useState(false)

  const { data: auditLog, isLoading, refetch } = useQuery({
    queryKey: ['audit', search, actionFilter, resourceFilter, dateFrom, dateTo],
    queryFn: () =>
      api.audit.list({
        ...(search && { actor_id: search }),
        ...(actionFilter && { action: actionFilter }),
        ...(resourceFilter && { resource_type: resourceFilter }),
        ...(dateFrom && { from: dateFrom }),
        ...(dateTo && { to: dateTo }),
        limit: '100',
      }),
  })

  const verifyChain = useMutation({
    mutationFn: () => api.audit.verify(true),
    onSuccess: () => setShowVerifyModal(true),
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Audit Log</h1>
          <p className="text-gray-600">Nepromenjivi zapis svih operacija u sistemu</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => refetch()}
            className="btn btn-secondary"
          >
            <RefreshCw className="h-4 w-4" />
            Osveži
          </button>
          <button
            onClick={() => verifyChain.mutate()}
            disabled={verifyChain.isPending}
            className="btn btn-primary"
          >
            <Shield className="h-4 w-4" />
            {verifyChain.isPending ? 'Verifikacija...' : 'Verifikuj lanac'}
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="card">
        <div className="flex items-center gap-2 mb-4">
          <Filter className="h-4 w-4 text-gray-500" />
          <span className="font-medium text-gray-700">Filteri</span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div>
            <label className="block text-sm text-gray-600 mb-1">Pretraga (Actor ID)</label>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                placeholder="UUID aktera..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="input pl-10 font-mono text-sm"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm text-gray-600 mb-1">Akcija</label>
            <select
              value={actionFilter}
              onChange={(e) => setActionFilter(e.target.value)}
              className="input"
            >
              <option value="">Sve akcije</option>
              <optgroup label="CRUD">
                <option value="CREATE">CREATE</option>
                <option value="READ">READ</option>
                <option value="UPDATE">UPDATE</option>
                <option value="DELETE">DELETE</option>
              </optgroup>
              <optgroup label="Operacije">
                <option value="SHARE">SHARE</option>
                <option value="SIGN">SIGN</option>
                <option value="OPEN">OPEN</option>
                <option value="CLOSE">CLOSE</option>
              </optgroup>
              <optgroup label="Simulacija">
                <option value="simulation.started">Simulacija pokrenuta</option>
                <option value="simulation.data_request">Zahtev za podacima</option>
                <option value="simulation.data_response">Odgovor sa podacima</option>
                <option value="simulation.completed">Simulacija završena</option>
              </optgroup>
            </select>
          </div>

          <div>
            <label className="block text-sm text-gray-600 mb-1">Tip resursa</label>
            <select
              value={resourceFilter}
              onChange={(e) => setResourceFilter(e.target.value)}
              className="input"
            >
              <option value="">Svi tipovi</option>
              <option value="AGENCY">Agencija</option>
              <option value="WORKER">Radnik</option>
              <option value="CASE">Slučaj</option>
              <option value="DOCUMENT">Dokument</option>
              <option value="simulation">Simulacija</option>
            </select>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="block text-sm text-gray-600 mb-1">Od</label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="input"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-600 mb-1">Do</label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="input"
              />
            </div>
          </div>
        </div>
      </div>

      {/* Audit Entries */}
      {isLoading ? (
        <div className="text-center py-12 text-gray-500">Učitavanje...</div>
      ) : auditLog?.data && auditLog.data.length > 0 ? (
        <div className="space-y-2">
          {auditLog.data.map((entry: AuditEntry) => (
            <AuditEntryCard key={entry.id} entry={entry} />
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-gray-500">
          Nema audit zapisa za prikazivanje
        </div>
      )}

      {/* Pagination Info */}
      {auditLog?.total !== undefined && (
        <div className="text-sm text-gray-500 text-center">
          Prikazano {auditLog.data?.length || 0} od {auditLog.total} zapisa
        </div>
      )}

      {/* Info Card */}
      <div className="card bg-blue-50 border-blue-200">
        <div className="flex items-start gap-3">
          <AlertTriangle className="h-5 w-5 text-blue-600 mt-0.5" />
          <div>
            <h3 className="font-medium text-blue-800">O Audit Logu</h3>
            <p className="text-sm text-blue-700 mt-1">
              Audit log koristi hash-chain strukturu gde svaki zapis sadrži hash prethodnog.
              Ovo omogućava detekciju bilo kakve manipulacije podacima. Redovna verifikacija
              osigurava integritet kompletne istorije sistema.
            </p>
          </div>
        </div>
      </div>

      {/* Verification Modal */}
      {showVerifyModal && verifyChain.data && (
        <VerificationModal
          data={verifyChain.data}
          onClose={() => setShowVerifyModal(false)}
        />
      )}
    </div>
  )
}

function VerificationModal({
  data,
  onClose
}: {
  data: { valid: boolean; checked: number; violations?: string[]; entries?: AuditVerifyEntry[] }
  onClose: () => void
}) {
  const [animatedIndex, setAnimatedIndex] = useState(-1)
  const [isAnimating, setIsAnimating] = useState(true)
  const entries = data.entries || []

  useEffect(() => {
    if (!isAnimating || entries.length === 0) return

    const timer = setInterval(() => {
      setAnimatedIndex((prev) => {
        if (prev >= entries.length - 1) {
          setIsAnimating(false)
          clearInterval(timer)
          return prev
        }
        return prev + 1
      })
    }, 150)

    return () => clearInterval(timer)
  }, [entries.length, isAnimating])

  const handleReplay = () => {
    setAnimatedIndex(-1)
    setIsAnimating(true)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-2xl max-w-3xl w-full max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center gap-3">
            <div className={`p-2 rounded-lg ${data.valid ? 'bg-green-100' : 'bg-red-100'}`}>
              {data.valid ? (
                <Shield className="h-6 w-6 text-green-600" />
              ) : (
                <XCircle className="h-6 w-6 text-red-600" />
              )}
            </div>
            <div>
              <h2 className="text-lg font-bold text-gray-900">
                Verifikacija Hash-Chain lanca
              </h2>
              <p className="text-sm text-gray-500">
                {data.valid
                  ? `Svih ${data.checked} zapisa uspešno verifikovano`
                  : `Pronađeno ${data.violations?.length || 0} narušenja integriteta`
                }
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <X className="h-5 w-5 text-gray-500" />
          </button>
        </div>

        {/* Status Banner */}
        <div className={`px-4 py-3 ${data.valid ? 'bg-green-50' : 'bg-red-50'}`}>
          <div className="flex items-center gap-2">
            {data.valid ? (
              <>
                <CheckCircle className="h-5 w-5 text-green-600" />
                <span className="font-medium text-green-800">
                  Integritet lanca je očuvan - nijedan zapis nije menjan
                </span>
              </>
            ) : (
              <>
                <XCircle className="h-5 w-5 text-red-600" />
                <span className="font-medium text-red-800">
                  UPOZORENJE: Detektovana manipulacija podacima!
                </span>
              </>
            )}
          </div>
          {data.violations && data.violations.length > 0 && (
            <ul className="mt-2 text-sm text-red-700 list-disc list-inside">
              {data.violations.map((v, i) => (
                <li key={i}>{v}</li>
              ))}
            </ul>
          )}
        </div>

        {/* Chain Visualization */}
        <div className="flex-1 overflow-y-auto p-4">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-medium text-gray-700">Vizualizacija lanca</h3>
            <button
              onClick={handleReplay}
              disabled={isAnimating}
              className="text-sm text-primary-600 hover:text-primary-700 flex items-center gap-1"
            >
              <RefreshCw className={`h-4 w-4 ${isAnimating ? 'animate-spin' : ''}`} />
              Ponovi animaciju
            </button>
          </div>

          <div className="space-y-1">
            {entries.map((entry, index) => {
              const isVisible = index <= animatedIndex
              const isCurrentlyAnimating = index === animatedIndex && isAnimating

              return (
                <div key={entry.id}>
                  {/* Entry Block */}
                  <div
                    className={`
                      transition-all duration-300 transform
                      ${isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 -translate-y-4'}
                      ${isCurrentlyAnimating ? 'scale-105' : 'scale-100'}
                    `}
                  >
                    <div
                      className={`
                        p-3 rounded-lg border-2 transition-colors
                        ${!isVisible ? 'border-gray-200 bg-gray-50' :
                          entry.valid
                            ? 'border-green-300 bg-green-50'
                            : 'border-red-300 bg-red-50'
                        }
                        ${isCurrentlyAnimating ? 'ring-2 ring-primary-400 ring-offset-2' : ''}
                      `}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className={`
                            w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold
                            ${!isVisible ? 'bg-gray-200 text-gray-500' :
                              entry.valid
                                ? 'bg-green-200 text-green-700'
                                : 'bg-red-200 text-red-700'
                            }
                          `}>
                            {entry.sequence}
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="font-medium text-gray-900">{entry.action}</span>
                              {isVisible && (
                                entry.valid ? (
                                  <CheckCircle className="h-4 w-4 text-green-600" />
                                ) : (
                                  <XCircle className="h-4 w-4 text-red-600" />
                                )
                              )}
                            </div>
                            <div className="text-xs text-gray-500 font-mono mt-1">
                              Hash: {entry.hash.slice(0, 16)}...
                            </div>
                          </div>
                        </div>
                        {isVisible && (
                          <div className={`text-xs px-2 py-1 rounded ${
                            entry.valid
                              ? 'bg-green-100 text-green-700'
                              : 'bg-red-100 text-red-700'
                          }`}>
                            {entry.valid ? 'Validan' : 'Nevalidan!'}
                          </div>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Chain Link */}
                  {index < entries.length - 1 && (
                    <div className={`
                      flex justify-center py-1 transition-all duration-300
                      ${index < animatedIndex ? 'opacity-100' : 'opacity-0'}
                    `}>
                      <div className="flex flex-col items-center">
                        <div className={`w-0.5 h-3 ${
                          entries[index + 1]?.valid !== false
                            ? 'bg-green-400'
                            : 'bg-red-400'
                        }`} />
                        <Link className={`h-4 w-4 ${
                          entries[index + 1]?.valid !== false
                            ? 'text-green-500'
                            : 'text-red-500'
                        }`} />
                        <div className={`w-0.5 h-3 ${
                          entries[index + 1]?.valid !== false
                            ? 'bg-green-400'
                            : 'bg-red-400'
                        }`} />
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>

          {entries.length === 0 && (
            <div className="text-center py-8 text-gray-500">
              Nema zapisa za prikaz
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t bg-gray-50 rounded-b-xl">
          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-600">
              <span className="font-medium">{data.checked}</span> zapisa verifikovano
              {!isAnimating && animatedIndex >= 0 && (
                <span className="ml-2 text-green-600">
                  Animacija završena
                </span>
              )}
            </div>
            <button
              onClick={onClose}
              className="btn btn-primary"
            >
              Zatvori
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function AuditEntryCard({ entry }: { entry: AuditEntry }) {
  const actionColors: Record<string, string> = {
    CREATE: 'bg-green-100 text-green-700',
    READ: 'bg-blue-100 text-blue-700',
    UPDATE: 'bg-yellow-100 text-yellow-700',
    DELETE: 'bg-red-100 text-red-700',
    SHARE: 'bg-purple-100 text-purple-700',
    SIGN: 'bg-indigo-100 text-indigo-700',
    OPEN: 'bg-cyan-100 text-cyan-700',
    CLOSE: 'bg-orange-100 text-orange-700',
    'simulation.started': 'bg-primary-100 text-primary-700',
    'simulation.data_request': 'bg-blue-100 text-blue-700',
    'simulation.data_response': 'bg-green-100 text-green-700',
    'simulation.completed': 'bg-primary-100 text-primary-700',
  }

  const resourceIcons: Record<string, string> = {
    AGENCY: '🏢',
    WORKER: '👤',
    CASE: '📁',
    DOCUMENT: '📄',
    simulation: '🔄',
  }

  return (
    <div className="card hover:shadow-sm transition-shadow">
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-3">
          <div className="text-2xl">{resourceIcons[entry.resource_type] || '📋'}</div>
          <div>
            <div className="flex items-center gap-2">
              <span className={`badge ${actionColors[entry.action] || 'bg-gray-100 text-gray-700'}`}>
                {entry.action}
              </span>
              <span className="text-sm text-gray-600">{entry.resource_type}</span>
            </div>
            {entry.resource_id && (
              <p className="text-sm text-gray-500 mt-1">
                Resurs: <span className="font-mono">{entry.resource_id.slice(0, 12)}...</span>
              </p>
            )}
            {entry.actor_id && (
              <p className="text-sm text-gray-500">
                Akter: <span className="font-mono">{entry.actor_id.slice(0, 12)}...</span>
              </p>
            )}
          </div>
        </div>

        <div className="text-right">
          <div className="flex items-center gap-1 text-sm text-gray-500">
            <Calendar className="h-3 w-3" />
            {new Date(entry.timestamp).toLocaleString('sr-RS')}
          </div>
          {entry.hash && (
            <p className="text-xs text-gray-400 font-mono mt-1">
              #{entry.hash.length > 8 ? entry.hash.slice(0, 8) : entry.hash}
            </p>
          )}
        </div>
      </div>

      {entry.metadata && Object.keys(entry.metadata).length > 0 && (
        <div className="mt-3 pt-3 border-t border-gray-100">
          <details className="text-sm">
            <summary className="cursor-pointer text-gray-500 hover:text-gray-700">
              Metapodaci
            </summary>
            <pre className="mt-2 p-2 bg-gray-50 rounded text-xs overflow-auto">
              {JSON.stringify(entry.metadata, null, 2)}
            </pre>
          </details>
        </div>
      )}
    </div>
  )
}
