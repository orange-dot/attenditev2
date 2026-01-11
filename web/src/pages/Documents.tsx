import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  FileText,
  Plus,
  Search,
  Filter,
  X,
  Download,
  History,
  PenTool,
  Archive,
  Share2,
} from 'lucide-react'
import { api } from '../api/client'
import type { Document, CreateDocumentRequest, DocumentVersion, SignatureRequest } from '../types'

export function Documents() {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [selectedDocument, setSelectedDocument] = useState<Document | null>(null)

  const { data: documents, isLoading } = useQuery({
    queryKey: ['documents', search, statusFilter, typeFilter],
    queryFn: () =>
      api.documents.list({
        ...(search && { search }),
        ...(statusFilter && { status: statusFilter }),
        ...(typeFilter && { type: typeFilter }),
      }),
  })

  const createDocument = useMutation({
    mutationFn: (data: CreateDocumentRequest) => api.documents.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['documents'] })
      setShowCreateModal(false)
    },
  })

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Dokumenti</h1>
          <p className="text-gray-600">Upravljanje dokumentima i potpisima</p>
        </div>
        <button onClick={() => setShowCreateModal(true)} className="btn btn-primary">
          <Plus className="h-4 w-4" />
          Novi dokument
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-4">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Pretraži dokumente..."
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
            <option value="ACTIVE">Aktivan</option>
            <option value="ARCHIVED">Arhiviran</option>
          </select>

          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
            className="input w-auto"
          >
            <option value="">Svi tipovi</option>
            <option value="MEDICAL_REPORT">Medicinski izveštaj</option>
            <option value="SOCIAL_REPORT">Socijalni izveštaj</option>
            <option value="LEGAL_DOCUMENT">Pravni dokument</option>
            <option value="ASSESSMENT">Procena</option>
            <option value="OTHER">Ostalo</option>
          </select>
        </div>
      </div>

      {/* Documents List */}
      {isLoading ? (
        <div className="text-center py-12 text-gray-500">Učitavanje...</div>
      ) : documents?.data && documents.data.length > 0 ? (
        <div className="space-y-4">
          {documents.data.map((doc: Document) => (
            <DocumentCard
              key={doc.id}
              document={doc}
              onSelect={() => setSelectedDocument(doc)}
            />
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-gray-500">
          {search || statusFilter || typeFilter
            ? 'Nema rezultata pretrage'
            : 'Nema registrovanih dokumenata'}
        </div>
      )}

      {/* Pagination Info */}
      {documents?.total !== undefined && (
        <div className="text-sm text-gray-500 text-center">
          Prikazano {documents.data?.length || 0} od {documents.total} dokumenata
        </div>
      )}

      {/* Create Document Modal */}
      {showCreateModal && (
        <CreateDocumentModal
          onClose={() => setShowCreateModal(false)}
          onSubmit={(data) => createDocument.mutate(data)}
          isLoading={createDocument.isPending}
        />
      )}

      {/* Document Detail Modal */}
      {selectedDocument && (
        <DocumentDetailModal
          document={selectedDocument}
          onClose={() => setSelectedDocument(null)}
          onUpdate={() => {
            queryClient.invalidateQueries({ queryKey: ['documents'] })
          }}
        />
      )}
    </div>
  )
}

function DocumentCard({ document, onSelect }: { document: Document; onSelect: () => void }) {
  const statusConfig: Record<string, { color: string; label: string }> = {
    DRAFT: { color: 'bg-gray-100 text-gray-700', label: 'Nacrt' },
    ACTIVE: { color: 'bg-green-100 text-green-700', label: 'Aktivan' },
    ARCHIVED: { color: 'bg-yellow-100 text-yellow-700', label: 'Arhiviran' },
  }

  const typeLabels: Record<string, string> = {
    MEDICAL_REPORT: 'Medicinski izveštaj',
    SOCIAL_REPORT: 'Socijalni izveštaj',
    LEGAL_DOCUMENT: 'Pravni dokument',
    ASSESSMENT: 'Procena',
    OTHER: 'Ostalo',
  }

  const status = statusConfig[document.status] || statusConfig.DRAFT

  return (
    <div
      onClick={onSelect}
      className="card cursor-pointer hover:shadow-md transition-shadow"
    >
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-3">
          <div className="rounded-lg bg-purple-100 p-2">
            <FileText className="h-5 w-5 text-purple-600" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">{document.title}</h3>
            <p className="text-sm text-gray-500">
              {typeLabels[document.type] || document.type} • v{document.version}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className={`badge ${status.color}`}>{status.label}</span>
        </div>
      </div>

      {(() => {
        const desc = document.metadata?.description
        return typeof desc === 'string' ? (
          <p className="mt-3 text-sm text-gray-600 line-clamp-2">{desc}</p>
        ) : null
      })()}

      <div className="mt-4 pt-3 border-t border-gray-100 flex items-center justify-between text-sm text-gray-500">
        <span className="font-mono text-xs">{document.id.slice(0, 8)}...</span>
        <span>
          Kreiran: {new Date(document.created_at).toLocaleDateString('sr-RS')}
        </span>
      </div>
    </div>
  )
}

function CreateDocumentModal({
  onClose,
  onSubmit,
  isLoading,
}: {
  onClose: () => void
  onSubmit: (data: CreateDocumentRequest) => void
  isLoading: boolean
}) {
  const [form, setForm] = useState<CreateDocumentRequest>({
    title: '',
    type: 'MEDICAL_REPORT',
    content: '',
    case_id: '',
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg mx-4">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">Novi dokument</h2>
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
              Tip dokumenta
            </label>
            <select
              value={form.type}
              onChange={(e) => setForm({ ...form, type: e.target.value })}
              className="input"
            >
              <option value="MEDICAL_REPORT">Medicinski izveštaj</option>
              <option value="SOCIAL_REPORT">Socijalni izveštaj</option>
              <option value="LEGAL_DOCUMENT">Pravni dokument</option>
              <option value="ASSESSMENT">Procena</option>
              <option value="OTHER">Ostalo</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Sadržaj
            </label>
            <textarea
              value={form.content}
              onChange={(e) => setForm({ ...form, content: e.target.value })}
              className="input font-mono text-sm"
              rows={8}
              placeholder="Unesite sadržaj dokumenta..."
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              ID slučaja (opciono)
            </label>
            <input
              type="text"
              value={form.case_id}
              onChange={(e) => setForm({ ...form, case_id: e.target.value })}
              className="input font-mono"
              placeholder="UUID slučaja"
            />
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

function DocumentDetailModal({
  document,
  onClose,
  onUpdate,
}: {
  document: Document
  onClose: () => void
  onUpdate: () => void
}) {
  const queryClient = useQueryClient()
  const [showRequestSignature, setShowRequestSignature] = useState(false)

  const { data: versions } = useQuery({
    queryKey: ['document-versions', document.id],
    queryFn: () => api.documents.versions(document.id),
  })

  const archiveDocument = useMutation({
    mutationFn: () => api.documents.archive(document.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['documents'] })
      onUpdate()
    },
  })

  const requestSignature = useMutation({
    mutationFn: (data: SignatureRequest) => api.documents.requestSignature(document.id, data),
    onSuccess: () => {
      setShowRequestSignature(false)
      queryClient.invalidateQueries({ queryKey: ['documents'] })
    },
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-3xl mx-4 max-h-[80vh] overflow-hidden flex flex-col">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">{document.title}</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-4 overflow-auto flex-1">
          {/* Document Info */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <div>
              <span className="text-sm text-gray-500">ID</span>
              <p className="font-mono text-sm">{document.id.slice(0, 8)}...</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">Status</span>
              <p>{document.status}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">Tip</span>
              <p>{document.type}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500">Verzija</span>
              <p>v{document.version}</p>
            </div>
          </div>

          {/* Content Preview */}
          <div className="mb-6">
            <span className="text-sm text-gray-500">Sadržaj</span>
            <div className="mt-2 p-4 bg-gray-50 rounded-lg">
              <pre className="text-sm whitespace-pre-wrap font-mono">
                {document.content || 'Nema sadržaja'}
              </pre>
            </div>
          </div>

          {/* Actions */}
          <div className="flex flex-wrap gap-2 mb-6 p-4 bg-gray-50 rounded-lg">
            <button className="btn btn-secondary">
              <Download className="h-4 w-4" />
              Preuzmi
            </button>
            <button
              onClick={() => setShowRequestSignature(true)}
              className="btn btn-secondary"
            >
              <PenTool className="h-4 w-4" />
              Zatraži potpis
            </button>
            <button className="btn btn-secondary">
              <Share2 className="h-4 w-4" />
              Podeli
            </button>
            {document.status !== 'ARCHIVED' && (
              <button
                onClick={() => archiveDocument.mutate()}
                disabled={archiveDocument.isPending}
                className="btn btn-secondary"
              >
                <Archive className="h-4 w-4" />
                {archiveDocument.isPending ? 'Arhiviranje...' : 'Arhiviraj'}
              </button>
            )}
          </div>

          {/* Version History */}
          <div className="border-t pt-4">
            <div className="flex items-center gap-2 mb-4">
              <History className="h-4 w-4 text-gray-500" />
              <h3 className="font-semibold">Istorija verzija</h3>
            </div>
            {versions?.data && versions.data.length > 0 ? (
              <div className="space-y-2">
                {versions.data.map((version: DocumentVersion) => (
                  <div
                    key={version.version}
                    className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
                  >
                    <div>
                      <p className="font-medium text-sm">Verzija {version.version}</p>
                      <p className="text-xs text-gray-500">
                        {new Date(version.created_at).toLocaleString('sr-RS')}
                      </p>
                    </div>
                    <span className="font-mono text-xs text-gray-400">
                      {version.hash?.slice(0, 12)}...
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500">Samo trenutna verzija</p>
            )}
          </div>
        </div>

        <div className="p-4 border-t">
          <button onClick={onClose} className="btn btn-secondary w-full">
            Zatvori
          </button>
        </div>

        {/* Request Signature Modal */}
        {showRequestSignature && (
          <RequestSignatureModal
            onClose={() => setShowRequestSignature(false)}
            onSubmit={(data) => requestSignature.mutate(data)}
            isLoading={requestSignature.isPending}
          />
        )}
      </div>
    </div>
  )
}

function RequestSignatureModal({
  onClose,
  onSubmit,
  isLoading,
}: {
  onClose: () => void
  onSubmit: (data: SignatureRequest) => void
  isLoading: boolean
}) {
  const [form, setForm] = useState<SignatureRequest>({
    signer_id: '',
    reason: '',
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">Zatraži potpis</h2>
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
              ID potpisnika
            </label>
            <input
              type="text"
              value={form.signer_id}
              onChange={(e) => setForm({ ...form, signer_id: e.target.value })}
              className="input font-mono"
              placeholder="UUID radnika"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Razlog
            </label>
            <textarea
              value={form.reason}
              onChange={(e) => setForm({ ...form, reason: e.target.value })}
              className="input"
              rows={3}
              placeholder="Zašto je potpis potreban?"
            />
          </div>

          <div className="flex gap-3 pt-4">
            <button type="button" onClick={onClose} className="btn btn-secondary flex-1">
              Otkaži
            </button>
            <button type="submit" disabled={isLoading} className="btn btn-primary flex-1">
              {isLoading ? 'Slanje...' : 'Zatraži'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
