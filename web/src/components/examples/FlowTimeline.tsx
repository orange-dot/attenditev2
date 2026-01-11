import { useState } from 'react'
import { FlowStep, getInstitutionById } from '../../data/exampleCases'
import {
  CheckCircle2,
  Circle,
  ArrowRight,
  Clock,
  Server,
  User,
  Building2,
  Database,
} from 'lucide-react'

interface FlowTimelineProps {
  steps: FlowStep[]
  currentStep: number
}

type ViewMode = 'procedural' | 'technical'

export function FlowTimeline({ steps, currentStep }: FlowTimelineProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('procedural')

  const getInstitutionIcon = (id: string) => {
    if (id === 'citizen') return User
    const inst = getInstitutionById(id)
    if (!inst) return Building2
    if (inst.type === 'datacenter') return Database
    if (inst.type === 'central') return Server
    return Building2
  }

  const getInstitutionName = (id: string) => {
    if (id === 'citizen') return 'Građanin'
    const inst = getInstitutionById(id)
    return inst?.shortName || id
  }

  const getStepStatus = (index: number) => {
    if (index < currentStep) return 'completed'
    if (index === currentStep) return 'active'
    return 'pending'
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 h-full flex flex-col">
      {/* View mode tabs */}
      <div className="flex border-b border-gray-200">
        <button
          onClick={() => setViewMode('procedural')}
          className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
            viewMode === 'procedural'
              ? 'text-primary-600 border-b-2 border-primary-600 bg-primary-50'
              : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
          }`}
        >
          <User className="w-4 h-4 inline-block mr-2" />
          Šta građanin vidi
        </button>
        <button
          onClick={() => setViewMode('technical')}
          className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
            viewMode === 'technical'
              ? 'text-primary-600 border-b-2 border-primary-600 bg-primary-50'
              : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
          }`}
        >
          <Server className="w-4 h-4 inline-block mr-2" />
          Šta sistem radi
        </button>
      </div>

      {/* Timeline */}
      <div className="flex-1 overflow-y-auto p-4">
        <div className="space-y-1">
          {steps.map((step, index) => {
            const status = getStepStatus(index)
            const text = viewMode === 'procedural' ? step.procedural : step.technical
            const FromIcon = getInstitutionIcon(step.from)
            const ToIcon = getInstitutionIcon(step.to)

            // Skip steps with no text in procedural view
            if (viewMode === 'procedural' && !text) return null

            return (
              <div
                key={step.id}
                className={`relative flex gap-3 p-3 rounded-lg transition-all duration-300 ${
                  status === 'active'
                    ? 'bg-primary-50 border border-primary-200 shadow-sm'
                    : status === 'completed'
                      ? 'bg-gray-50'
                      : 'opacity-50'
                }`}
              >
                {/* Status indicator */}
                <div className="flex-shrink-0 mt-0.5">
                  {status === 'completed' ? (
                    <CheckCircle2 className="w-5 h-5 text-green-500" />
                  ) : status === 'active' ? (
                    <div className="relative">
                      <Circle className="w-5 h-5 text-primary-500" />
                      <span className="absolute inset-0 w-5 h-5 bg-primary-500 rounded-full animate-ping opacity-25" />
                    </div>
                  ) : (
                    <Clock className="w-5 h-5 text-gray-300" />
                  )}
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  {/* From -> To header */}
                  <div className="flex items-center gap-2 text-xs text-gray-500 mb-1">
                    <span className="flex items-center gap-1">
                      <FromIcon className="w-3 h-3" />
                      {getInstitutionName(step.from)}
                    </span>
                    <ArrowRight
                      className={`w-3 h-3 ${step.isResponse ? 'text-green-500' : 'text-primary-500'}`}
                    />
                    <span className="flex items-center gap-1">
                      <ToIcon className="w-3 h-3" />
                      {getInstitutionName(step.to)}
                    </span>
                  </div>

                  {/* Description */}
                  <p
                    className={`text-sm ${status === 'active' ? 'text-gray-900 font-medium' : 'text-gray-600'}`}
                  >
                    {text}
                  </p>

                  {/* Data exchanged (only in technical view) */}
                  {viewMode === 'technical' && step.dataExchanged.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                      {step.dataExchanged.map((data, i) => (
                        <span
                          key={i}
                          className={`inline-flex items-center px-2 py-0.5 rounded text-xs ${
                            step.isResponse
                              ? 'bg-green-100 text-green-700'
                              : 'bg-primary-100 text-primary-700'
                          }`}
                        >
                          {data}
                        </span>
                      ))}
                    </div>
                  )}
                </div>

                {/* Step number */}
                <div className="flex-shrink-0">
                  <span
                    className={`inline-flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium ${
                      status === 'active'
                        ? 'bg-primary-600 text-white'
                        : status === 'completed'
                          ? 'bg-green-100 text-green-700'
                          : 'bg-gray-100 text-gray-400'
                    }`}
                  >
                    {index + 1}
                  </span>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Summary */}
      <div className="border-t border-gray-200 p-4 bg-gray-50">
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-500">Progres:</span>
          <span className="font-medium text-gray-900">
            {Math.min(currentStep + 1, steps.length)} / {steps.length} koraka
          </span>
        </div>
        <div className="mt-2 w-full bg-gray-200 rounded-full h-2">
          <div
            className="bg-primary-600 h-2 rounded-full transition-all duration-500"
            style={{ width: `${((currentStep + 1) / steps.length) * 100}%` }}
          />
        </div>
      </div>
    </div>
  )
}
