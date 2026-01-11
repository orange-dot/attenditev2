import { useEffect, useMemo, useRef } from 'react'
import { MapContainer, TileLayer, Marker, Popup, Polyline, useMap } from 'react-leaflet'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { Institution, FlowStep, getInstitutionById } from '../../data/exampleCases'

// Fix Leaflet default marker icon issue
delete (L.Icon.Default.prototype as unknown as { _getIconUrl?: unknown })._getIconUrl
L.Icon.Default.mergeOptions({
  iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
  iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon.png',
  shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png',
})

interface ExampleMapProps {
  institutions: Institution[]
  steps: FlowStep[]
  currentStep: number
  isPlaying: boolean
}

// Custom marker icons based on institution type
function createCustomIcon(type: Institution['type'], isActive: boolean): L.DivIcon {
  const colors = {
    local: { bg: '#0C4076', border: '#0A3560' },
    central: { bg: '#C6363C', border: '#A52D32' },
    datacenter: { bg: '#F59E0B', border: '#D97706' },
  }

  const color = colors[type]
  const size = isActive ? 40 : 32
  const pulse = isActive ? 'animation: pulse 1s infinite;' : ''

  return L.divIcon({
    className: 'custom-marker',
    html: `
      <div style="
        width: ${size}px;
        height: ${size}px;
        background-color: ${color.bg};
        border: 3px solid ${color.border};
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 0 2px 8px rgba(0,0,0,0.3);
        ${pulse}
        transition: all 0.3s ease;
      ">
        <svg width="${size * 0.5}" height="${size * 0.5}" viewBox="0 0 24 24" fill="white">
          ${
            type === 'local'
              ? '<path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z"/>'
              : type === 'central'
                ? '<path d="M12 2L2 7v10l10 5 10-5V7L12 2zm0 2.18l6 3v5.64l-6 3-6-3V7.18l6-3z"/>'
                : '<path d="M4 6h16v2H4zm0 5h16v2H4zm0 5h16v2H4z"/>'
          }
        </svg>
      </div>
    `,
    iconSize: [size, size],
    iconAnchor: [size / 2, size / 2],
  })
}

// Component to fit map bounds to show all markers
function MapBoundsHandler({ institutions }: { institutions: Institution[] }) {
  const map = useMap()

  useEffect(() => {
    if (institutions.length > 0) {
      const bounds = L.latLngBounds(institutions.map((i) => i.coords))
      map.fitBounds(bounds, { padding: [50, 50] })
    }
  }, [institutions, map])

  return null
}

// Animated polyline component
function AnimatedPolyline({
  from,
  to,
  isActive,
  isCompleted,
  isResponse,
}: {
  from: [number, number]
  to: [number, number]
  isActive: boolean
  isCompleted: boolean
  isResponse?: boolean
}) {
  const color = isResponse ? '#22C55E' : '#0C4076'
  const opacity = isCompleted ? 1 : isActive ? 0.8 : 0.3
  const weight = isActive ? 4 : 3
  const dashArray = isActive ? '10, 10' : undefined

  return (
    <Polyline
      positions={[from, to]}
      pathOptions={{
        color,
        weight,
        opacity,
        dashArray,
      }}
    />
  )
}

export function ExampleMap({ institutions, steps, currentStep, isPlaying }: ExampleMapProps) {
  const mapRef = useRef<L.Map>(null)

  // Determine which institutions are active in current step
  const activeInstitutions = useMemo(() => {
    if (currentStep < 0 || currentStep >= steps.length) return new Set<string>()
    const step = steps[currentStep]
    return new Set([step.from, step.to])
  }, [steps, currentStep])

  // Generate lines for completed and current steps
  const lines = useMemo(() => {
    return steps.slice(0, currentStep + 1).map((step, index) => {
      const fromInst = getInstitutionById(step.from)
      const toInst = getInstitutionById(step.to)

      // Handle citizen (virtual position near the target institution)
      let fromCoords: [number, number]
      let toCoords: [number, number]

      if (step.from === 'citizen') {
        const targetInst = getInstitutionById(step.to)
        fromCoords = targetInst
          ? [targetInst.coords[0] + 0.005, targetInst.coords[1] - 0.005]
          : [45.83, 20.46]
        toCoords = toInst?.coords || [45.83, 20.46]
      } else if (step.to === 'citizen') {
        const sourceInst = getInstitutionById(step.from)
        fromCoords = fromInst?.coords || [45.83, 20.46]
        toCoords = sourceInst
          ? [sourceInst.coords[0] + 0.005, sourceInst.coords[1] - 0.005]
          : [45.83, 20.46]
      } else {
        fromCoords = fromInst?.coords || [45.83, 20.46]
        toCoords = toInst?.coords || [45.83, 20.46]
      }

      return {
        from: fromCoords,
        to: toCoords,
        isActive: index === currentStep,
        isCompleted: index < currentStep,
        isResponse: step.isResponse,
        key: step.id,
      }
    })
  }, [steps, currentStep])

  // Serbia center coordinates
  const serbiaCenter: [number, number] = [44.5, 20.7]

  return (
    <div className="relative h-full w-full rounded-lg overflow-hidden border border-gray-200">
      <style>
        {`
          @keyframes pulse {
            0% { transform: scale(1); box-shadow: 0 2px 8px rgba(0,0,0,0.3); }
            50% { transform: scale(1.1); box-shadow: 0 4px 16px rgba(0,0,0,0.4); }
            100% { transform: scale(1); box-shadow: 0 2px 8px rgba(0,0,0,0.3); }
          }
          .leaflet-container {
            font-family: inherit;
          }
        `}
      </style>

      <MapContainer
        center={serbiaCenter}
        zoom={8}
        style={{ height: '100%', width: '100%' }}
        ref={mapRef}
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />

        <MapBoundsHandler institutions={institutions} />

        {/* Draw connection lines */}
        {lines.map((line) => (
          <AnimatedPolyline
            key={line.key}
            from={line.from}
            to={line.to}
            isActive={line.isActive}
            isCompleted={line.isCompleted}
            isResponse={line.isResponse}
          />
        ))}

        {/* Institution markers */}
        {institutions.map((institution) => (
          <Marker
            key={institution.id}
            position={institution.coords}
            icon={createCustomIcon(institution.type, activeInstitutions.has(institution.id))}
          >
            <Popup>
              <div className="text-sm">
                <h3 className="font-bold text-gray-900">{institution.name}</h3>
                <p className="text-gray-600 text-xs mt-1">{institution.description}</p>
                <div className="mt-2 flex items-center gap-1">
                  <span
                    className={`inline-block w-2 h-2 rounded-full ${
                      institution.type === 'local'
                        ? 'bg-primary-600'
                        : institution.type === 'central'
                          ? 'bg-accent-600'
                          : 'bg-amber-500'
                    }`}
                  />
                  <span className="text-xs text-gray-500">
                    {institution.type === 'local'
                      ? 'Lokalna institucija'
                      : institution.type === 'central'
                        ? 'Centralna institucija'
                        : 'Data centar'}
                  </span>
                </div>
              </div>
            </Popup>
          </Marker>
        ))}
      </MapContainer>

      {/* Legend */}
      <div className="absolute bottom-4 left-4 bg-white rounded-lg shadow-lg p-3 z-[1000]">
        <h4 className="text-xs font-semibold text-gray-700 mb-2">Legenda</h4>
        <div className="space-y-1.5">
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded-full bg-primary-600" />
            <span className="text-xs text-gray-600">Lokalna institucija</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded-full bg-accent-600" />
            <span className="text-xs text-gray-600">Centralna institucija</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded-full bg-amber-500" />
            <span className="text-xs text-gray-600">Data centar</span>
          </div>
          <div className="border-t border-gray-200 my-1.5" />
          <div className="flex items-center gap-2">
            <span className="w-4 h-0.5 bg-primary-600" />
            <span className="text-xs text-gray-600">Zahtev</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-4 h-0.5 bg-green-500" />
            <span className="text-xs text-gray-600">Odgovor</span>
          </div>
        </div>
      </div>

      {/* Playing indicator */}
      {isPlaying && (
        <div className="absolute top-4 right-4 bg-primary-600 text-white px-3 py-1.5 rounded-full text-xs font-medium z-[1000] flex items-center gap-2">
          <span className="w-2 h-2 bg-white rounded-full animate-pulse" />
          Animacija u toku
        </div>
      )}
    </div>
  )
}
