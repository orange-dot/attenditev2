import { useState, useEffect, useCallback, useRef } from 'react'
import { Play, Pause, RotateCcw, Camera, Download, CheckCircle2, Database } from 'lucide-react'
import html2canvas from 'html2canvas'
import { useCases, getInstitutionsForUseCase } from '../data/exampleCases'
import { ExampleMap } from '../components/examples/ExampleMap'
import { FlowTimeline } from '../components/examples/FlowTimeline'
import { UseCaseSelector } from '../components/examples/UseCaseSelector'
import { api } from '../api/client'

export function Examples() {
  const [selectedUseCaseId, setSelectedUseCaseId] = useState(useCases[0].id)
  const [currentStep, setCurrentStep] = useState(-1)
  const [isPlaying, setIsPlaying] = useState(false)
  const [isCapturing, setIsCapturing] = useState(false)
  const [sessionId, setSessionId] = useState<string | null>(null)
  const [auditCount, setAuditCount] = useState(0)
  const [lastAuditMessage, setLastAuditMessage] = useState<string | null>(null)
  const captureRef = useRef<HTMLDivElement>(null)
  const isExecutingRef = useRef(false)

  const selectedUseCase = useCases.find((uc) => uc.id === selectedUseCaseId) || useCases[0]
  const institutions = getInstitutionsForUseCase(selectedUseCase)

  // Reset animation when use case changes
  useEffect(() => {
    setCurrentStep(-1)
    setIsPlaying(false)
    setSessionId(null)
    setAuditCount(0)
    setLastAuditMessage(null)
  }, [selectedUseCaseId])

  // Execute simulation step
  const executeStep = useCallback(async (stepIndex: number, simSessionId: string) => {
    if (stepIndex >= selectedUseCase.steps.length) {
      // Simulation completed
      try {
        await api.simulation.complete({
          session_id: simSessionId,
          use_case_id: selectedUseCase.id,
          use_case_title: selectedUseCase.title,
          total_steps: selectedUseCase.steps.length,
          success: true,
        })
        setLastAuditMessage('Simulacija završena - svi koraci zabeleženi')
        setAuditCount((prev) => prev + 1)
      } catch (error) {
        console.error('Failed to complete simulation:', error)
      }
      setIsPlaying(false)
      return
    }

    const step = selectedUseCase.steps[stepIndex]

    try {
      const response = await api.simulation.step({
        use_case_id: selectedUseCase.id,
        use_case_title: selectedUseCase.title,
        session_id: simSessionId,
        step: {
          step_id: step.id,
          from_institution: step.from,
          to_institution: step.to,
          action: step.isResponse ? 'DATA_RESPONSE' : 'DATA_REQUEST',
          description: step.technical,
          data_exchanged: step.dataExchanged,
          is_response: step.isResponse || false,
        },
      })

      if (response.success) {
        setAuditCount((prev) => prev + 1)
        setLastAuditMessage(response.message)
      }
    } catch (error) {
      console.error('Failed to execute step:', error)
    }

    setCurrentStep(stepIndex)
  }, [selectedUseCase])

  // Animation loop with API calls
  useEffect(() => {
    if (!isPlaying || !sessionId || isExecutingRef.current) return

    const runNextStep = async () => {
      isExecutingRef.current = true
      const nextStep = currentStep + 1

      if (nextStep >= selectedUseCase.steps.length) {
        // Complete simulation
        await executeStep(nextStep, sessionId)
        isExecutingRef.current = false
        return
      }

      await executeStep(nextStep, sessionId)
      isExecutingRef.current = false
    }

    const timer = setTimeout(runNextStep, currentStep < 0 ? 100 : 1500)
    return () => clearTimeout(timer)
  }, [isPlaying, sessionId, currentStep, selectedUseCase.steps.length, executeStep])

  const handlePlayPause = useCallback(async () => {
    if (isPlaying) {
      setIsPlaying(false)
      return
    }

    if (currentStep >= selectedUseCase.steps.length - 1 || sessionId === null) {
      // Start new simulation
      setCurrentStep(-1)
      setAuditCount(0)
      setLastAuditMessage(null)

      try {
        const response = await api.simulation.start({
          use_case_id: selectedUseCase.id,
          use_case_title: selectedUseCase.title,
          total_steps: selectedUseCase.steps.length,
          citizen_jmbg: '0101990710123', // Demo JMBG
        })

        if (response.success) {
          setSessionId(response.session_id)
          setAuditCount(1)
          setLastAuditMessage(response.message)
          setIsPlaying(true)
        }
      } catch (error) {
        console.error('Failed to start simulation:', error)
        // Fallback to offline mode
        setSessionId('offline-' + Date.now())
        setIsPlaying(true)
      }
    } else {
      setIsPlaying(true)
    }
  }, [isPlaying, currentStep, selectedUseCase, sessionId])

  const handleReset = useCallback(() => {
    setCurrentStep(-1)
    setIsPlaying(false)
    setSessionId(null)
    setAuditCount(0)
    setLastAuditMessage(null)
  }, [])

  const handleCapture = useCallback(async () => {
    if (!captureRef.current) return

    setIsCapturing(true)
    try {
      const canvas = await html2canvas(captureRef.current, {
        backgroundColor: '#ffffff',
        scale: 2,
        useCORS: true,
        allowTaint: true,
      })

      const link = document.createElement('a')
      link.download = `attendit-primer-${selectedUseCase.id}-${Date.now()}.png`
      link.href = canvas.toDataURL('image/png')
      link.click()
    } catch (error) {
      console.error('Error capturing screenshot:', error)
    } finally {
      setIsCapturing(false)
    }
  }, [selectedUseCase.id])

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Primeri korišćenja</h1>
          <p className="text-gray-500 mt-1">
            Vizualizacija toka razmene podataka između institucija
          </p>
        </div>
        <button
          onClick={handleCapture}
          disabled={isCapturing}
          className="btn btn-secondary flex items-center gap-2"
        >
          {isCapturing ? (
            <>
              <Download className="w-4 h-4 animate-bounce" />
              Čuvam...
            </>
          ) : (
            <>
              <Camera className="w-4 h-4" />
              Sačuvaj sliku
            </>
          )}
        </button>
      </div>

      {/* Use case selector */}
      <UseCaseSelector
        useCases={useCases}
        selectedId={selectedUseCaseId}
        onSelect={setSelectedUseCaseId}
      />

      {/* Use case description */}
      <div className="bg-primary-50 border border-primary-200 rounded-lg p-4">
        <h3 className="font-medium text-primary-900">{selectedUseCase.title}</h3>
        <p className="text-sm text-primary-700 mt-1">{selectedUseCase.description}</p>
      </div>

      {/* Playback controls */}
      <div className="bg-white rounded-lg border border-gray-200 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <button
              onClick={handlePlayPause}
              className={`p-3 rounded-full transition-colors ${
                isPlaying
                  ? 'bg-accent-100 text-accent-600 hover:bg-accent-200'
                  : 'bg-primary-100 text-primary-600 hover:bg-primary-200'
              }`}
            >
              {isPlaying ? <Pause className="w-5 h-5" /> : <Play className="w-5 h-5" />}
            </button>
            <button
              onClick={handleReset}
              className="p-3 rounded-full bg-gray-100 text-gray-600 hover:bg-gray-200 transition-colors"
            >
              <RotateCcw className="w-5 h-5" />
            </button>
            <span className="ml-3 text-sm text-gray-600">
              {currentStep < 0
                ? 'Kliknite Play za pokretanje simulacije'
                : currentStep >= selectedUseCase.steps.length - 1
                  ? 'Simulacija završena'
                  : `Korak ${currentStep + 1} od ${selectedUseCase.steps.length}`}
            </span>
          </div>

          <div className="flex items-center gap-4 text-sm">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-primary-600" />
              <span className="text-gray-600">Zahtev</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-green-500" />
              <span className="text-gray-600">Odgovor</span>
            </div>
          </div>
        </div>
      </div>

      {/* Audit status banner */}
      {sessionId && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-green-100 rounded-lg">
                <Database className="w-5 h-5 text-green-600" />
              </div>
              <div>
                <h4 className="font-medium text-green-900 flex items-center gap-2">
                  Audit zapisi se generišu
                  <CheckCircle2 className="w-4 h-4" />
                </h4>
                <p className="text-sm text-green-700">
                  {lastAuditMessage || 'Čekam pokretanje...'}
                </p>
              </div>
            </div>
            <div className="text-right">
              <div className="text-2xl font-bold text-green-700">{auditCount}</div>
              <div className="text-xs text-green-600">zapisa</div>
            </div>
          </div>
          {sessionId && !sessionId.startsWith('offline') && (
            <div className="mt-2 pt-2 border-t border-green-200">
              <p className="text-xs text-green-600">
                Session ID: <span className="font-mono">{sessionId.slice(0, 12)}...</span>
                {' | '}
                <a href="/audit" className="underline hover:text-green-800">
                  Pogledaj audit log →
                </a>
              </p>
            </div>
          )}
        </div>
      )}

      {/* Main content - Map and Timeline */}
      <div ref={captureRef} className="bg-white rounded-lg border border-gray-200 p-4">
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-4" style={{ minHeight: '500px' }}>
          {/* Map - 3/5 width */}
          <div className="lg:col-span-3 h-[500px]">
            <ExampleMap
              institutions={institutions}
              steps={selectedUseCase.steps}
              currentStep={currentStep}
              isPlaying={isPlaying}
            />
          </div>

          {/* Timeline - 2/5 width */}
          <div className="lg:col-span-2 h-[500px]">
            <FlowTimeline steps={selectedUseCase.steps} currentStep={currentStep} />
          </div>
        </div>
      </div>

      {/* Key points */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h4 className="font-medium text-gray-900 mb-2">Bez dokumenata</h4>
          <p className="text-sm text-gray-600">
            Građanin ne mora da nosi papire. Sistem automatski proverava sve potrebne podatke.
          </p>
        </div>
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h4 className="font-medium text-gray-900 mb-2">Minimalni podaci</h4>
          <p className="text-sm text-gray-600">
            Razmenjuju se samo potrebne informacije (DA/NE, kategorija), ne celi dokumenti.
          </p>
        </div>
        <div className="bg-white rounded-lg border border-gray-200 p-4">
          <h4 className="font-medium text-gray-900 mb-2">Lokalna kontrola</h4>
          <p className="text-sm text-gray-600">
            Svaka institucija zadržava kontrolu nad svojim podacima. Nema centralnog skladištenja.
          </p>
        </div>
      </div>
    </div>
  )
}
