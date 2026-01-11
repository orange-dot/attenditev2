import { useEffect, useState } from 'react'
import { Activity, CheckCircle, XCircle } from 'lucide-react'
import { api } from '../../api/client'

export function Header() {
  const [health, setHealth] = useState<'healthy' | 'unhealthy' | 'checking'>('checking')
  const [aiHealth, setAiHealth] = useState<'healthy' | 'unhealthy' | 'checking'>('checking')

  useEffect(() => {
    const checkHealth = async () => {
      try {
        await api.health()
        setHealth('healthy')
      } catch {
        setHealth('unhealthy')
      }

      try {
        await api.ai.health()
        setAiHealth('healthy')
      } catch {
        setAiHealth('unhealthy')
      }
    }

    checkHealth()
    const interval = setInterval(checkHealth, 30000)
    return () => clearInterval(interval)
  }, [])

  return (
    <header className="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-6">
      <div className="flex items-center gap-2">
        <Activity className="h-5 w-5 text-gray-400" />
        <span className="text-sm text-gray-600">Status:</span>
        <StatusBadge label="API" status={health} />
        <StatusBadge label="AI" status={aiHealth} />
      </div>

      <div className="flex items-center gap-4">
        <span className="text-sm text-gray-500">Demo User</span>
        <div className="h-8 w-8 rounded-full bg-primary-100 flex items-center justify-center">
          <span className="text-sm font-medium text-primary-700">DU</span>
        </div>
      </div>
    </header>
  )
}

function StatusBadge({ label, status }: { label: string; status: 'healthy' | 'unhealthy' | 'checking' }) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
        status === 'healthy'
          ? 'bg-green-100 text-green-700'
          : status === 'unhealthy'
          ? 'bg-red-100 text-red-700'
          : 'bg-gray-100 text-gray-700'
      }`}
    >
      {status === 'healthy' ? (
        <CheckCircle className="h-3 w-3" />
      ) : status === 'unhealthy' ? (
        <XCircle className="h-3 w-3" />
      ) : (
        <Activity className="h-3 w-3 animate-pulse" />
      )}
      {label}
    </span>
  )
}
