import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Building2, Plus, Users, Search, X } from 'lucide-react'
import { api } from '../api/client'
import type { Agency, Worker, CreateAgencyRequest, CreateWorkerRequest } from '../types'

export function Agencies() {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [selectedAgency, setSelectedAgency] = useState<Agency | null>(null)
  const [showWorkerModal, setShowWorkerModal] = useState(false)

  const { data: agencies, isLoading } = useQuery({
    queryKey: ['agencies', search],
    queryFn: () => api.agencies.list(search ? { search } : undefined),
  })

  const createAgency = useMutation({
    mutationFn: (data: CreateAgencyRequest) => api.agencies.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agencies'] })
      setShowCreateModal(false)
    },
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Agencije</h1>
          <p className="text-gray-600">Upravljanje javnim službama u sistemu</p>
        </div>
        <button onClick={() => setShowCreateModal(true)} className="btn btn-primary">
          <Plus className="h-4 w-4" />
          Nova agencija
        </button>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          placeholder="Pretraži agencije..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="input pl-10"
        />
      </div>

      {/* Agencies Grid */}
      {isLoading ? (
        <div className="text-center py-12 text-gray-500">Učitavanje...</div>
      ) : agencies?.data && agencies.data.length > 0 ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {agencies.data.map((agency: Agency) => (
            <AgencyCard
              key={agency.id}
              agency={agency}
              onSelect={() => setSelectedAgency(agency)}
            />
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-gray-500">
          {search ? 'Nema rezultata pretrage' : 'Nema registrovanih agencija'}
        </div>
      )}

      {/* Create Agency Modal */}
      {showCreateModal && (
        <CreateAgencyModal
          onClose={() => setShowCreateModal(false)}
          onSubmit={(data) => createAgency.mutate(data)}
          isLoading={createAgency.isPending}
        />
      )}

      {/* Agency Detail Modal */}
      {selectedAgency && (
        <AgencyDetailModal
          agency={selectedAgency}
          onClose={() => setSelectedAgency(null)}
          onAddWorker={() => setShowWorkerModal(true)}
        />
      )}

      {/* Add Worker Modal */}
      {showWorkerModal && selectedAgency && (
        <AddWorkerModal
          agencyId={selectedAgency.id}
          onClose={() => setShowWorkerModal(false)}
          onSuccess={() => {
            setShowWorkerModal(false)
            queryClient.invalidateQueries({ queryKey: ['workers', selectedAgency.id] })
          }}
        />
      )}
    </div>
  )
}

function AgencyCard({ agency, onSelect }: { agency: Agency; onSelect: () => void }) {
  const typeColors: Record<string, string> = {
    SOCIAL_SERVICES: 'bg-blue-100 text-blue-700',
    HEALTHCARE: 'bg-green-100 text-green-700',
    EDUCATION: 'bg-purple-100 text-purple-700',
    POLICE: 'bg-red-100 text-red-700',
    JUDICIARY: 'bg-amber-100 text-amber-700',
    LOCAL_GOVERNMENT: 'bg-cyan-100 text-cyan-700',
    TAX: 'bg-orange-100 text-orange-700',
    EMERGENCY: 'bg-rose-100 text-rose-700',
    OTHER: 'bg-gray-100 text-gray-700',
  }

  const typeLabels: Record<string, string> = {
    SOCIAL_SERVICES: 'Socijalna zaštita',
    HEALTHCARE: 'Zdravstvo',
    EDUCATION: 'Obrazovanje',
    POLICE: 'Policija',
    JUDICIARY: 'Pravosuđe',
    LOCAL_GOVERNMENT: 'Lokalna samouprava',
    TAX: 'Poreska uprava',
    EMERGENCY: 'Hitne službe',
    OTHER: 'Ostalo',
  }

  return (
    <div
      onClick={onSelect}
      className="card cursor-pointer hover:shadow-md transition-shadow"
    >
      <div className="flex items-start gap-3">
        <div className="rounded-lg bg-blue-100 p-2">
          <Building2 className="h-5 w-5 text-blue-600" />
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="font-semibold text-gray-900 truncate">{agency.name}</h3>
          <p className="text-sm text-gray-500 truncate font-mono">{agency.code}</p>
        </div>
      </div>

      <div className="mt-4 flex items-center gap-2">
        <span className={`badge ${typeColors[agency.type] || typeColors.OTHER}`}>
          {typeLabels[agency.type] || agency.type}
        </span>
        {agency.status === 'active' && (
          <span className="badge bg-green-100 text-green-700">Aktivna</span>
        )}
      </div>

      <div className="mt-3 pt-3 border-t border-gray-100 flex items-center gap-2 text-sm text-gray-500">
        <Users className="h-4 w-4" />
        <span>Vidi detalje</span>
      </div>
    </div>
  )
}

function CreateAgencyModal({
  onClose,
  onSubmit,
  isLoading,
}: {
  onClose: () => void
  onSubmit: (data: CreateAgencyRequest) => void
  isLoading: boolean
}) {
  const [form, setForm] = useState<CreateAgencyRequest>({
    code: '',
    name: '',
    type: 'SOCIAL_SERVICES',
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">Nova agencija</h2>
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
              Kod agencije
            </label>
            <input
              type="text"
              value={form.code}
              onChange={(e) => setForm({ ...form, code: e.target.value.toUpperCase() })}
              className="input"
              placeholder="npr. CSR-BG"
              required
            />
            <p className="text-xs text-gray-500 mt-1">Jedinstveni identifikator (2-50 karaktera)</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Naziv
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="input"
              placeholder="npr. Centar za socijalni rad Beograd"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Tip
            </label>
            <select
              value={form.type}
              onChange={(e) => setForm({ ...form, type: e.target.value })}
              className="input"
            >
              <option value="SOCIAL_SERVICES">Socijalna zaštita</option>
              <option value="HEALTHCARE">Zdravstvo</option>
              <option value="EDUCATION">Obrazovanje</option>
              <option value="POLICE">Policija</option>
              <option value="JUDICIARY">Pravosuđe</option>
              <option value="LOCAL_GOVERNMENT">Lokalna samouprava</option>
              <option value="TAX">Poreska uprava</option>
              <option value="EMERGENCY">Hitne službe</option>
              <option value="OTHER">Ostalo</option>
            </select>
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

function AgencyDetailModal({
  agency,
  onClose,
  onAddWorker,
}: {
  agency: Agency
  onClose: () => void
  onAddWorker: () => void
}) {
  const { data: workers } = useQuery({
    queryKey: ['workers', agency.id],
    queryFn: () => api.agencies.workers(agency.id),
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl mx-4 max-h-[80vh] overflow-hidden flex flex-col">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">{agency.name}</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-4 overflow-auto flex-1">
          {/* Agency Info */}
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <span className="text-sm text-gray-500">Kod</span>
              <p className="font-mono text-sm font-semibold">{agency.code}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">Tip</span>
              <p>{agency.type}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">ID</span>
              <p className="font-mono text-sm">{agency.id.slice(0, 8)}...</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">Status</span>
              <p className={agency.status === 'active' ? 'text-green-600' : 'text-gray-500'}>
                {agency.status === 'active' ? 'Aktivna' : agency.status}
              </p>
            </div>
          </div>

          {/* Workers */}
          <div className="border-t pt-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold">Radnici</h3>
              <button onClick={onAddWorker} className="btn btn-secondary text-sm">
                <Plus className="h-4 w-4" />
                Dodaj radnika
              </button>
            </div>

            {workers?.data && workers.data.length > 0 ? (
              <div className="space-y-2">
                {workers.data.map((worker: Worker) => (
                  <div
                    key={worker.id}
                    className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
                  >
                    <div>
                      <p className="font-medium">{worker.first_name} {worker.last_name}</p>
                      <p className="text-sm text-gray-500">{worker.position || worker.department || 'Bez pozicije'}</p>
                      <p className="text-xs text-gray-400 font-mono">{worker.employee_id}</p>
                    </div>
                    <span
                      className={`badge ${
                        worker.status === 'active'
                          ? 'bg-green-100 text-green-700'
                          : 'bg-gray-100 text-gray-700'
                      }`}
                    >
                      {worker.status === 'active' ? 'Aktivan' : worker.status}
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500">Nema registrovanih radnika</p>
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

function AddWorkerModal({
  agencyId,
  onClose,
  onSuccess,
}: {
  agencyId: string
  onClose: () => void
  onSuccess: () => void
}) {
  const [form, setForm] = useState<CreateWorkerRequest>({
    employee_id: '',
    first_name: '',
    last_name: '',
    email: '',
    position: 'Socijalni radnik',
    department: '',
    roles: ['CASE_WORKER'],
  })

  const createWorker = useMutation({
    mutationFn: (data: CreateWorkerRequest) => api.agencies.createWorker(agencyId, data),
    onSuccess,
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-4 border-b sticky top-0 bg-white">
          <h2 className="text-lg font-semibold">Novi radnik</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            createWorker.mutate(form)
          }}
          className="p-4 space-y-4"
        >
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Matični broj zaposlenog
            </label>
            <input
              type="text"
              value={form.employee_id}
              onChange={(e) => setForm({ ...form, employee_id: e.target.value })}
              className="input"
              placeholder="npr. EMP-001"
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Ime
              </label>
              <input
                type="text"
                value={form.first_name}
                onChange={(e) => setForm({ ...form, first_name: e.target.value })}
                className="input"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Prezime
              </label>
              <input
                type="text"
                value={form.last_name}
                onChange={(e) => setForm({ ...form, last_name: e.target.value })}
                className="input"
                required
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Email
            </label>
            <input
              type="email"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className="input"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Pozicija
            </label>
            <input
              type="text"
              value={form.position}
              onChange={(e) => setForm({ ...form, position: e.target.value })}
              className="input"
              placeholder="npr. Socijalni radnik"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Odeljenje
            </label>
            <input
              type="text"
              value={form.department}
              onChange={(e) => setForm({ ...form, department: e.target.value })}
              className="input"
              placeholder="npr. Služba za zaštitu dece"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Uloga u sistemu
            </label>
            <select
              value={form.roles?.[0] || 'CASE_WORKER'}
              onChange={(e) => setForm({ ...form, roles: [e.target.value] })}
              className="input"
            >
              <option value="CASE_WORKER">Socijalni radnik</option>
              <option value="SUPERVISOR">Supervizor</option>
              <option value="ADMIN">Administrator</option>
            </select>
          </div>

          <div className="flex gap-3 pt-4">
            <button type="button" onClick={onClose} className="btn btn-secondary flex-1">
              Otkaži
            </button>
            <button
              type="submit"
              disabled={createWorker.isPending}
              className="btn btn-primary flex-1"
            >
              {createWorker.isPending ? 'Dodavanje...' : 'Dodaj'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
