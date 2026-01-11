import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  FolderOpen,
  Plus,
  Search,
  Filter,
  X,
  Clock,
  AlertTriangle,
  CheckCircle,
  Play,
  Share2,
} from 'lucide-react'
import { api } from '../api/client'
import type { Case, CreateCaseRequest, CaseEvent } from '../types'

export function Cases() {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [priorityFilter, setPriorityFilter] = useState<string>('')
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [selectedCase, setSelectedCase] = useState<Case | null>(null)

  const { data: cases, isLoading } = useQuery({
    queryKey: ['cases', search, statusFilter, priorityFilter],
    queryFn: () =>
      api.cases.list({
        ...(search && { search }),
        ...(statusFilter && { status: statusFilter }),
        ...(priorityFilter && { priority: priorityFilter }),
      }),
  })

  const createCase = useMutation({
    mutationFn: (data: CreateCaseRequest) => api.cases.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cases'] })
      setShowCreateModal(false)
    },
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Slučajevi</h1>
          <p className="text-gray-600">Upravljanje slučajevima i koordinacija između agencija</p>
        </div>
        <button onClick={() => setShowCreateModal(true)} className="btn btn-primary">
          <Plus className="h-4 w-4" />
          Novi slučaj
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-4">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Pretraži slučajeve..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="input pl-10"
          />
        </div>

        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-gray-400" />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="input w-auto"
          >
            <option value="">Svi statusi</option>
            <option value="DRAFT">Nacrt</option>
            <option value="OPEN">Otvoren</option>
            <option value="IN_PROGRESS">U toku</option>
            <option value="CLOSED">Zatvoren</option>
          </select>

          <select
            value={priorityFilter}
            onChange={(e) => setPriorityFilter(e.target.value)}
            className="input w-auto"
          >
            <option value="">Svi prioriteti</option>
            <option value="LOW">Nizak</option>
            <option value="MEDIUM">Srednji</option>
            <option value="HIGH">Visok</option>
            <option value="CRITICAL">Kritičan</option>
          </select>
        </div>
      </div>

      {/* Cases List */}
      {isLoading ? (
        <div className="text-center py-12 text-gray-500">Učitavanje...</div>
      ) : cases?.data && cases.data.length > 0 ? (
        <div className="space-y-4">
          {cases.data.map((caseItem: Case) => (
            <CaseCard
              key={caseItem.id}
              caseItem={caseItem}
              onSelect={() => setSelectedCase(caseItem)}
            />
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-gray-500">
          {search || statusFilter || priorityFilter
            ? 'Nema rezultata pretrage'
            : 'Nema registrovanih slučajeva'}
        </div>
      )}

      {/* Pagination Info */}
      {cases?.total !== undefined && (
        <div className="text-sm text-gray-500 text-center">
          Prikazano {cases.data?.length || 0} od {cases.total} slučajeva
        </div>
      )}

      {/* Create Case Modal */}
      {showCreateModal && (
        <CreateCaseModal
          onClose={() => setShowCreateModal(false)}
          onSubmit={(data) => createCase.mutate(data)}
          isLoading={createCase.isPending}
        />
      )}

      {/* Case Detail Modal */}
      {selectedCase && (
        <CaseDetailModal
          caseItem={selectedCase}
          onClose={() => setSelectedCase(null)}
          onUpdate={() => {
            queryClient.invalidateQueries({ queryKey: ['cases'] })
          }}
        />
      )}
    </div>
  )
}

function CaseCard({ caseItem, onSelect }: { caseItem: Case; onSelect: () => void }) {
  const statusConfig: Record<string, { color: string; icon: React.ElementType; label: string }> = {
    DRAFT: { color: 'bg-gray-100 text-gray-700', icon: Clock, label: 'Nacrt' },
    OPEN: { color: 'bg-blue-100 text-blue-700', icon: FolderOpen, label: 'Otvoren' },
    IN_PROGRESS: { color: 'bg-yellow-100 text-yellow-700', icon: Play, label: 'U toku' },
    CLOSED: { color: 'bg-green-100 text-green-700', icon: CheckCircle, label: 'Zatvoren' },
  }

  const priorityConfig: Record<string, { color: string; label: string }> = {
    LOW: { color: 'bg-gray-100 text-gray-600', label: 'Nizak' },
    MEDIUM: { color: 'bg-blue-100 text-blue-600', label: 'Srednji' },
    HIGH: { color: 'bg-orange-100 text-orange-600', label: 'Visok' },
    CRITICAL: { color: 'bg-red-100 text-red-600', label: 'Kritičan' },
  }

  const status = statusConfig[caseItem.status] || statusConfig.DRAFT
  const priority = priorityConfig[caseItem.priority] || priorityConfig.MEDIUM
  const StatusIcon = status.icon

  return (
    <div
      onClick={onSelect}
      className="card cursor-pointer hover:shadow-md transition-shadow"
    >
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-3">
          <div className="rounded-lg bg-green-100 p-2">
            <FolderOpen className="h-5 w-5 text-green-600" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">{caseItem.title}</h3>
            <p className="text-sm text-gray-500">
              {caseItem.type} • Kreiran: {new Date(caseItem.created_at).toLocaleDateString('sr-RS')}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className={`badge ${priority.color}`}>
            {caseItem.priority === 'HIGH' || caseItem.priority === 'CRITICAL' ? (
              <AlertTriangle className="h-3 w-3 mr-1" />
            ) : null}
            {priority.label}
          </span>
          <span className={`badge ${status.color}`}>
            <StatusIcon className="h-3 w-3 mr-1" />
            {status.label}
          </span>
        </div>
      </div>

      {caseItem.description && (
        <p className="mt-3 text-sm text-gray-600 line-clamp-2">{caseItem.description}</p>
      )}

      <div className="mt-4 pt-3 border-t border-gray-100 flex items-center gap-4 text-sm text-gray-500">
        <span className="font-mono text-xs">{caseItem.id.slice(0, 8)}...</span>
        {caseItem.lead_agency_id && (
          <span>Vodeća agencija: {caseItem.lead_agency_id.slice(0, 8)}...</span>
        )}
      </div>
    </div>
  )
}

function CreateCaseModal({
  onClose,
  onSubmit,
  isLoading,
}: {
  onClose: () => void
  onSubmit: (data: CreateCaseRequest) => void
  isLoading: boolean
}) {
  const [form, setForm] = useState<CreateCaseRequest>({
    title: '',
    description: '',
    type: 'CHILD_PROTECTION',
    priority: 'MEDIUM',
    lead_agency_id: '',
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">Novi slučaj</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            onSubmit(form)
          }}
          className="p-4 space-y-4"
        >
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Naslov
            </label>
            <input
              type="text"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              className="input"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Opis
            </label>
            <textarea
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              className="input"
              rows={3}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Tip
              </label>
              <select
                value={form.type}
                onChange={(e) => setForm({ ...form, type: e.target.value })}
                className="input"
              >
                <option value="CHILD_PROTECTION">Zaštita dece</option>
                <option value="DOMESTIC_VIOLENCE">Nasilje u porodici</option>
                <option value="ELDER_CARE">Briga o starijima</option>
                <option value="DISABILITY_SUPPORT">Podrška osobama sa invaliditetom</option>
                <option value="OTHER">Ostalo</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Prioritet
              </label>
              <select
                value={form.priority}
                onChange={(e) => setForm({ ...form, priority: e.target.value })}
                className="input"
              >
                <option value="LOW">Nizak</option>
                <option value="MEDIUM">Srednji</option>
                <option value="HIGH">Visok</option>
                <option value="CRITICAL">Kritičan</option>
              </select>
            </div>
          </div>

          <div className="flex gap-3 pt-4">
            <button type="button" onClick={onClose} className="btn btn-secondary flex-1">
              Otkaži
            </button>
            <button type="submit" disabled={isLoading} className="btn btn-primary flex-1">
              {isLoading ? 'Kreiranje...' : 'Kreiraj'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function CaseDetailModal({
  caseItem,
  onClose,
  onUpdate,
}: {
  caseItem: Case
  onClose: () => void
  onUpdate: () => void
}) {
  const queryClient = useQueryClient()

  const { data: events } = useQuery({
    queryKey: ['case-events', caseItem.id],
    queryFn: () => api.cases.events(caseItem.id),
  })

  const openCase = useMutation({
    mutationFn: () => api.cases.open(caseItem.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cases'] })
      onUpdate()
    },
  })

  const startCase = useMutation({
    mutationFn: () => api.cases.start(caseItem.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cases'] })
      onUpdate()
    },
  })

  const closeCase = useMutation({
    mutationFn: (resolution: string) => api.cases.close(caseItem.id, resolution),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cases'] })
      onUpdate()
    },
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-3xl mx-4 max-h-[80vh] overflow-hidden flex flex-col">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">{caseItem.title}</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-4 overflow-auto flex-1">
          {/* Case Info */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <div>
              <span className="text-sm text-gray-500">ID</span>
              <p className="font-mono text-sm">{caseItem.id.slice(0, 8)}...</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">Status</span>
              <p>{caseItem.status}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">Prioritet</span>
              <p>{caseItem.priority}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">Tip</span>
              <p>{caseItem.type}</p>
            </div>
          </div>

          {caseItem.description && (
            <div className="mb-6">
              <span className="text-sm text-gray-500">Opis</span>
              <p className="mt-1">{caseItem.description}</p>
            </div>
          )}

          {/* Actions */}
          <div className="flex flex-wrap gap-2 mb-6 p-4 bg-gray-50 rounded-lg">
            {caseItem.status === 'DRAFT' && (
              <button
                onClick={() => openCase.mutate()}
                disabled={openCase.isPending}
                className="btn btn-primary"
              >
                <FolderOpen className="h-4 w-4" />
                {openCase.isPending ? 'Otvaranje...' : 'Otvori slučaj'}
              </button>
            )}
            {caseItem.status === 'OPEN' && (
              <button
                onClick={() => startCase.mutate()}
                disabled={startCase.isPending}
                className="btn btn-primary"
              >
                <Play className="h-4 w-4" />
                {startCase.isPending ? 'Pokretanje...' : 'Započni rad'}
              </button>
            )}
            {(caseItem.status === 'OPEN' || caseItem.status === 'IN_PROGRESS') && (
              <>
                <button
                  onClick={() => {
                    const resolution = prompt('Unesite razlog zatvaranja:')
                    if (resolution) closeCase.mutate(resolution)
                  }}
                  disabled={closeCase.isPending}
                  className="btn btn-secondary"
                >
                  <CheckCircle className="h-4 w-4" />
                  {closeCase.isPending ? 'Zatvaranje...' : 'Zatvori slučaj'}
                </button>
                <button className="btn btn-secondary">
                  <Share2 className="h-4 w-4" />
                  Podeli sa agencijom
                </button>
              </>
            )}
          </div>

          {/* Timeline */}
          <div className="border-t pt-4">
            <h3 className="font-semibold mb-4">Istorija događaja</h3>
            {events?.data && events.data.length > 0 ? (
              <div className="space-y-3">
                {events.data.map((event: CaseEvent) => (
                  <div
                    key={event.id}
                    className="flex items-start gap-3 p-3 bg-gray-50 rounded-lg"
                  >
                    <div className="h-2 w-2 rounded-full bg-primary-500 mt-2" />
                    <div className="flex-1">
                      <p className="font-medium text-sm">{event.event_type}</p>
                      <p className="text-sm text-gray-500">
                        {new Date(event.timestamp).toLocaleString('sr-RS')}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500">Nema zabeleženih događaja</p>
            )}
          </div>
        </div>

        <div className="p-4 border-t">
          <button onClick={onClose} className="btn btn-secondary w-full">
            Zatvori
          </button>
        </div>
      </div>
    </div>
  )
}
