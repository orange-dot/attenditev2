import { Routes, Route } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import { Dashboard } from './pages/Dashboard'
import { Agencies } from './pages/Agencies'
import { Cases } from './pages/Cases'
import { Documents } from './pages/Documents'
import { AI } from './pages/AI'
import { Privacy } from './pages/Privacy'
import { Security } from './pages/Security'
import { Audit } from './pages/Audit'
import { ChainSecurity } from './pages/ChainSecurity'
import { Federation } from './pages/Federation'
import { Examples } from './pages/Examples'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="agencies" element={<Agencies />} />
        <Route path="cases" element={<Cases />} />
        <Route path="documents" element={<Documents />} />
        <Route path="ai" element={<AI />} />
        <Route path="privacy" element={<Privacy />} />
        <Route path="security" element={<Security />} />
        <Route path="audit" element={<Audit />} />
        <Route path="chain-security" element={<ChainSecurity />} />
        <Route path="federation" element={<Federation />} />
        <Route path="examples" element={<Examples />} />
      </Route>
    </Routes>
  )
}
