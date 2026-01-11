import { useQuery } from '@tanstack/react-query'
import { Building2, FolderOpen, FileText, Activity, Brain, ArrowRight } from 'lucide-react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'

export function Dashboard() {
  const { data: agencies } = useQuery({
    queryKey: ['agencies'],
    queryFn: () => api.agencies.list(),
  })

  const { data: cases } = useQuery({
    queryKey: ['cases'],
    queryFn: () => api.cases.list(),
  })

  const { data: documents } = useQuery({
    queryKey: ['documents'],
    queryFn: () => api.documents.list(),
  })

  const { data: audit } = useQuery({
    queryKey: ['audit'],
    queryFn: () => api.audit.list({ limit: '5' }),
  })

  const stats = [
    { name: 'Agencije', value: agencies?.total ?? '-', icon: Building2, href: '/agencies', color: 'bg-blue-500' },
    { name: 'Slučajevi', value: cases?.total ?? '-', icon: FolderOpen, href: '/cases', color: 'bg-green-500' },
    { name: 'Dokumenti', value: documents?.total ?? '-', icon: FileText, href: '/documents', color: 'bg-purple-500' },
    { name: 'Audit zapisi', value: audit?.total ?? '-', icon: Activity, href: '/audit', color: 'bg-orange-500' },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-600">Pregled sistema koordinacije javnih službi</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Link
            key={stat.name}
            to={stat.href}
            className="card hover:shadow-md transition-shadow"
          >
            <div className="flex items-center gap-4">
              <div className={`rounded-lg p-3 ${stat.color}`}>
                <stat.icon className="h-6 w-6 text-white" />
              </div>
              <div>
                <p className="text-sm text-gray-600">{stat.name}</p>
                <p className="text-2xl font-semibold text-gray-900">{stat.value}</p>
              </div>
            </div>
          </Link>
        ))}
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* AI Analysis Card */}
        <div className="card">
          <div className="flex items-center gap-3 mb-4">
            <div className="rounded-lg bg-purple-100 p-2">
              <Brain className="h-5 w-5 text-purple-600" />
            </div>
            <h2 className="text-lg font-semibold text-gray-900">AI Detekcija Anomalija</h2>
          </div>
          <p className="text-sm text-gray-600 mb-4">
            Analizirajte medicinsku dokumentaciju za automatsku detekciju logičkih nekonzistentnosti,
            nemogućih uputstava i kršenja protokola.
          </p>
          <Link to="/ai" className="btn btn-primary">
            Pokreni analizu
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>

        {/* Recent Activity */}
        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Poslednje aktivnosti</h2>
          {audit?.data && audit.data.length > 0 ? (
            <div className="space-y-3">
              {audit.data.slice(0, 5).map((entry: any) => (
                <div key={entry.id} className="flex items-center gap-3 text-sm">
                  <div className="h-2 w-2 rounded-full bg-primary-500" />
                  <span className="text-gray-600">{entry.action}</span>
                  <span className="text-gray-400">-</span>
                  <span className="text-gray-500">{entry.resource_type}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-500">Nema nedavnih aktivnosti</p>
          )}
          <Link
            to="/audit"
            className="mt-4 inline-flex items-center text-sm text-primary-600 hover:text-primary-700"
          >
            Pogledaj sve
            <ArrowRight className="ml-1 h-4 w-4" />
          </Link>
        </div>
      </div>

      {/* System Info */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">O Sistemu</h2>
        <div className="prose prose-sm text-gray-600">
          <p>
            <strong>Serbia Government Interoperability Platform</strong> omogućava koordinaciju između
            javnih službi - centara za socijalni rad, zdravstvenih ustanova i drugih agencija.
          </p>
          <p>
            Ključne funkcionalnosti:
          </p>
          <ul>
            <li>Upravljanje slučajevima sa real-time koordinacijom</li>
            <li>Federativna razmena podataka između agencija</li>
            <li>AI detekcija anomalija u medicinskoj dokumentaciji</li>
            <li>Nepromenjivi audit trail za sve operacije</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
