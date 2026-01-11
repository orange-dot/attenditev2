import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Building2,
  FolderOpen,
  FileText,
  Shield,
  ShieldCheck,
  Network,
  Brain,
  ClipboardList,
  EyeOff,
  Map,
  Link,
} from 'lucide-react'

const navigation = [
  { name: 'Kontrolna tabla', href: '/', icon: LayoutDashboard },
  { name: 'Primeri', href: '/examples', icon: Map },
  { name: 'Ustanove', href: '/agencies', icon: Building2 },
  { name: 'Predmeti', href: '/cases', icon: FolderOpen },
  { name: 'Dokumenta', href: '/documents', icon: FileText },
  { name: 'AI Analiza', href: '/ai', icon: Brain },
  { name: 'Privatnost', href: '/privacy', icon: EyeOff },
  { name: 'Bezbednost', href: '/security', icon: ShieldCheck },
  { name: 'Revizija', href: '/audit', icon: ClipboardList },
  { name: 'Integritet lanca', href: '/chain-security', icon: Link },
  { name: 'Federacija', href: '/federation', icon: Network },
]

export function Sidebar() {
  return (
    <div className="flex h-full w-64 flex-col bg-primary-900">
      {/* Logo */}
      <div className="flex h-16 items-center gap-3 px-6 border-b border-primary-700">
        <div className="flex items-center justify-center h-10 w-10 rounded-lg bg-gradient-to-b from-accent-500 via-primary-500 to-white/90">
          <Shield className="h-6 w-6 text-white" />
        </div>
        <div>
          <div className="text-sm font-semibold text-white">Vlada Srbije</div>
          <div className="text-xs text-gray-400">Platforma za interoperabilnost</div>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-4 py-4 space-y-1">
        {navigation.map((item) => (
          <NavLink
            key={item.name}
            to={item.href}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-accent-500 text-white'
                  : 'text-gray-300 hover:bg-primary-700 hover:text-white'
              }`
            }
          >
            <item.icon className="h-5 w-5" />
            {item.name}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t border-primary-700 p-4">
        <div className="rounded-lg bg-primary-800 p-3">
          <div className="text-xs font-medium text-gray-300">Demo verzija</div>
          <div className="mt-1 text-xs text-gray-400">
            Sistem socijalne zaštite
          </div>
        </div>
      </div>
    </div>
  )
}
