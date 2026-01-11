import { useState } from 'react'
import {
  Shield,
  Users,
  Key,
  Lock,
  Eye,
  Clock,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Fingerprint,
} from 'lucide-react'

// Role definitions from security-model.md
const systemRoles = [
  { id: 'platform_admin', name: 'Platform Admin', description: 'Potpuni pristup platformi', scope: 'Globalno' },
  { id: 'platform_operator', name: 'Platform Operator', description: 'Operacije, monitoring', scope: 'Globalno' },
  { id: 'security_auditor', name: 'Security Auditor', description: 'Read-only pristup reviziji', scope: 'Globalno' },
]

const agencyRoles = [
  { id: 'agency_admin', name: 'Admin ustanove', description: 'Upravljanje radnicima i podešavanjima', scope: 'Ustanova' },
  { id: 'agency_supervisor', name: 'Supervizor', description: 'Nadzor radnika, eskalacije', scope: 'Ustanova' },
  { id: 'case_worker', name: 'Radnik na predmetima', description: 'Obrada predmeta', scope: 'Ustanova' },
  { id: 'dispatch_operator', name: 'Dispečer', description: 'Dispečerska konzola', scope: 'Ustanova' },
  { id: 'field_unit', name: 'Terenska jedinica', description: 'Mobilni terenski radnik', scope: 'Ustanova' },
  { id: 'agency_viewer', name: 'Pregled ustanove', description: 'Read-only pristup', scope: 'Ustanova' },
]

// Permission matrix
const permissionMatrix = [
  { permission: 'case.create', platform_admin: true, agency_admin: true, supervisor: true, case_worker: true, citizen: true },
  { permission: 'case.read', platform_admin: true, agency_admin: true, supervisor: true, case_worker: true, citizen: false },
  { permission: 'case.update', platform_admin: true, agency_admin: true, supervisor: true, case_worker: true, citizen: false },
  { permission: 'case.delete', platform_admin: true, agency_admin: false, supervisor: false, case_worker: false, citizen: false },
  { permission: 'case.assign', platform_admin: true, agency_admin: true, supervisor: true, case_worker: false, citizen: false },
  { permission: 'case.transfer', platform_admin: true, agency_admin: true, supervisor: true, case_worker: false, citizen: false },
  { permission: 'document.create', platform_admin: true, agency_admin: true, supervisor: true, case_worker: true, citizen: true },
  { permission: 'document.sign', platform_admin: true, agency_admin: true, supervisor: true, case_worker: true, citizen: true },
  { permission: 'audit.read', platform_admin: true, agency_admin: false, supervisor: false, case_worker: false, citizen: false },
]

// Access levels
const accessLevels = [
  { level: 0, name: 'Bez pristupa', description: 'Pristup je opozvan', color: 'bg-gray-100 text-gray-800' },
  { level: 1, name: 'Čitanje', description: 'Pregled predmeta i dokumenata', color: 'bg-blue-100 text-blue-800' },
  { level: 2, name: 'Komentari', description: 'Dodavanje beleški i poruka', color: 'bg-green-100 text-green-800' },
  { level: 3, name: 'Doprinos', description: 'Ažuriranje, dodela svojih radnika', color: 'bg-yellow-100 text-yellow-800' },
  { level: 4, name: 'Potpuni', description: 'Sve osim prenosa vlasništva', color: 'bg-accent-100 text-accent-800' },
]

// Data classification
const dataClassification = [
  { level: 0, name: 'Javno', example: 'Kontakt informacije ustanove', access: 'Svi', icon: Eye, color: 'text-green-600' },
  { level: 1, name: 'Interno', example: 'Statistika predmeta', access: 'Autentifikovani radnici', icon: Users, color: 'text-blue-600' },
  { level: 2, name: 'Poverljivo', example: 'Detalji predmeta', access: 'Dodeljeni radnici', icon: Lock, color: 'text-yellow-600' },
  { level: 3, name: 'Ograničeno', example: 'Medicinski podaci', access: 'Need-to-know + revizija', icon: AlertTriangle, color: 'text-orange-600' },
  { level: 4, name: 'Tajno', example: 'Zaštita svedoka', access: 'Specijalno ovlašćenje', icon: Shield, color: 'text-red-600' },
]

// Mock current session
const currentSession = {
  id: 'sess-demo-001',
  user: 'Demo Korisnik',
  userType: 'worker',
  agency: 'CSR Kikinda',
  roles: ['case_worker'],
  permissions: ['case.create', 'case.read', 'case.update', 'document.create', 'document.sign'],
  mfaVerified: true,
  eidVerified: false,
  loginTime: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
  lastActivity: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
  expiresAt: new Date(Date.now() + 6 * 60 * 60 * 1000).toISOString(),
  ipAddress: '192.168.1.100',
}

// Session config from security-model.md
const sessionConfig = {
  accessTokenTTL: '15 minuta',
  refreshTokenTTL: '8 sati',
  idleTimeout: '30 minuta',
  absoluteTimeout: '12 sati',
  maxConcurrentSessions: 3,
}

type TabType = 'roles' | 'permissions' | 'access' | 'session'

export function Security() {
  const [activeTab, setActiveTab] = useState<TabType>('roles')

  const tabs = [
    { id: 'roles' as const, name: 'Uloge', icon: Users },
    { id: 'permissions' as const, name: 'Dozvole', icon: Key },
    { id: 'access' as const, name: 'Nivoi pristupa', icon: Lock },
    { id: 'session' as const, name: 'Sesija', icon: Fingerprint },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Bezbednost</h1>
          <p className="text-gray-600">Pregled bezbednosnog modela i trenutne sesije</p>
        </div>
        <div className="flex items-center gap-2 px-4 py-2 bg-primary-50 rounded-lg">
          <Shield className="h-5 w-5 text-primary-600" />
          <span className="text-sm font-medium text-primary-700">RBAC + ABAC Model</span>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-2 py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.id
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <tab.icon className="h-4 w-4" />
              {tab.name}
            </button>
          ))}
        </nav>
      </div>

      {/* Content */}
      {activeTab === 'roles' && (
        <div className="space-y-6">
          {/* System Roles */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Sistemske uloge</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead>
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Uloga</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Opis</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Opseg</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {systemRoles.map((role) => (
                    <tr key={role.id}>
                      <td className="px-4 py-3">
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-accent-100 text-accent-800">
                          {role.name}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-600">{role.description}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{role.scope}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Agency Roles */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Uloge u ustanovi</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead>
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Uloga</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Opis</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Opseg</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {agencyRoles.map((role) => (
                    <tr key={role.id}>
                      <td className="px-4 py-3">
                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-primary-100 text-primary-800">
                          {role.name}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-600">{role.description}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{role.scope}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'permissions' && (
        <div className="card">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Matrica dozvola</h3>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead>
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Dozvola</th>
                  <th className="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase">Platform Admin</th>
                  <th className="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase">Agency Admin</th>
                  <th className="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase">Supervizor</th>
                  <th className="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase">Radnik</th>
                  <th className="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase">Građanin</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {permissionMatrix.map((row) => (
                  <tr key={row.permission}>
                    <td className="px-4 py-3">
                      <code className="text-sm bg-gray-100 px-2 py-0.5 rounded">{row.permission}</code>
                    </td>
                    <td className="px-4 py-3 text-center">
                      {row.platform_admin ? (
                        <CheckCircle className="h-5 w-5 text-green-600 mx-auto" />
                      ) : (
                        <XCircle className="h-5 w-5 text-gray-300 mx-auto" />
                      )}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {row.agency_admin ? (
                        <CheckCircle className="h-5 w-5 text-green-600 mx-auto" />
                      ) : (
                        <XCircle className="h-5 w-5 text-gray-300 mx-auto" />
                      )}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {row.supervisor ? (
                        <CheckCircle className="h-5 w-5 text-green-600 mx-auto" />
                      ) : (
                        <XCircle className="h-5 w-5 text-gray-300 mx-auto" />
                      )}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {row.case_worker ? (
                        <CheckCircle className="h-5 w-5 text-green-600 mx-auto" />
                      ) : (
                        <XCircle className="h-5 w-5 text-gray-300 mx-auto" />
                      )}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {row.citizen ? (
                        <CheckCircle className="h-5 w-5 text-green-600 mx-auto" />
                      ) : (
                        <XCircle className="h-5 w-5 text-gray-300 mx-auto" />
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="mt-4 text-xs text-gray-500">
            * Neke dozvole su uslovne (npr. case.read za radnike važi samo za dodeljene predmete)
          </p>
        </div>
      )}

      {activeTab === 'access' && (
        <div className="space-y-6">
          {/* Access Levels */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Nivoi pristupa za deljenje</h3>
            <div className="space-y-3">
              {accessLevels.map((level) => (
                <div key={level.level} className="flex items-center gap-4 p-3 bg-gray-50 rounded-lg">
                  <div className="flex items-center justify-center w-8 h-8 rounded-full bg-gray-200 text-gray-700 font-bold text-sm">
                    {level.level}
                  </div>
                  <div className="flex-1">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${level.color}`}>
                      {level.name}
                    </span>
                    <p className="text-sm text-gray-600 mt-1">{level.description}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Data Classification */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Klasifikacija podataka</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead>
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Nivo</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Oznaka</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Primer</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Pristup</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {dataClassification.map((item) => (
                    <tr key={item.level}>
                      <td className="px-4 py-3 text-sm font-medium text-gray-900">{item.level}</td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <item.icon className={`h-4 w-4 ${item.color}`} />
                          <span className="text-sm font-medium">{item.name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-600">{item.example}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{item.access}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'session' && (
        <div className="space-y-6">
          {/* Current Session */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Trenutna sesija</h3>
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                <span className="w-2 h-2 bg-green-500 rounded-full mr-1.5 animate-pulse"></span>
                Aktivna
              </span>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="p-4 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-500">Korisnik</p>
                <p className="font-medium">{currentSession.user}</p>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-500">Ustanova</p>
                <p className="font-medium">{currentSession.agency}</p>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-500">Uloge</p>
                <div className="flex flex-wrap gap-1 mt-1">
                  {currentSession.roles.map((role) => (
                    <span key={role} className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-primary-100 text-primary-800">
                      {role}
                    </span>
                  ))}
                </div>
              </div>
              <div className="p-4 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-500">IP Adresa</p>
                <p className="font-medium font-mono text-sm">{currentSession.ipAddress}</p>
              </div>
            </div>

            <div className="mt-4 grid grid-cols-3 gap-4">
              <div className="flex items-center gap-2">
                {currentSession.mfaVerified ? (
                  <CheckCircle className="h-5 w-5 text-green-600" />
                ) : (
                  <XCircle className="h-5 w-5 text-gray-400" />
                )}
                <span className="text-sm">MFA verifikovan</span>
              </div>
              <div className="flex items-center gap-2">
                {currentSession.eidVerified ? (
                  <CheckCircle className="h-5 w-5 text-green-600" />
                ) : (
                  <XCircle className="h-5 w-5 text-gray-400" />
                )}
                <span className="text-sm">eID verifikovan</span>
              </div>
              <div className="flex items-center gap-2">
                <Clock className="h-5 w-5 text-gray-400" />
                <span className="text-sm">
                  Ističe: {new Date(currentSession.expiresAt).toLocaleTimeString('sr-RS')}
                </span>
              </div>
            </div>
          </div>

          {/* Session Config */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Konfiguracija sesije</h3>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
              <div className="p-4 border border-gray-200 rounded-lg">
                <p className="text-sm text-gray-500">Access Token TTL</p>
                <p className="text-lg font-semibold">{sessionConfig.accessTokenTTL}</p>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <p className="text-sm text-gray-500">Refresh Token TTL</p>
                <p className="text-lg font-semibold">{sessionConfig.refreshTokenTTL}</p>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <p className="text-sm text-gray-500">Idle Timeout</p>
                <p className="text-lg font-semibold">{sessionConfig.idleTimeout}</p>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <p className="text-sm text-gray-500">Absolute Timeout</p>
                <p className="text-lg font-semibold">{sessionConfig.absoluteTimeout}</p>
              </div>
              <div className="p-4 border border-gray-200 rounded-lg">
                <p className="text-sm text-gray-500">Max sesija</p>
                <p className="text-lg font-semibold">{sessionConfig.maxConcurrentSessions} po korisniku</p>
              </div>
            </div>
          </div>

          {/* Permissions */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Dozvole u sesiji</h3>
            <div className="flex flex-wrap gap-2">
              {currentSession.permissions.map((perm) => (
                <span key={perm} className="inline-flex items-center px-3 py-1 rounded-lg text-sm font-medium bg-gray-100 text-gray-800">
                  <Key className="h-3 w-3 mr-1.5" />
                  {perm}
                </span>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
