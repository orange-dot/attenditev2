import { UseCase } from '../../data/exampleCases'
import { Users, Heart, Clock, Zap } from 'lucide-react'

interface UseCaseSelectorProps {
  useCases: UseCase[]
  selectedId: string
  onSelect: (id: string) => void
}

export function UseCaseSelector({ useCases, selectedId, onSelect }: UseCaseSelectorProps) {
  const getIcon = (id: string) => {
    if (id === 'gerontoloski-smestaj') return Users
    return Heart
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <h2 className="text-sm font-semibold text-gray-700 mb-3">Izaberite primer</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {useCases.map((useCase) => {
          const Icon = getIcon(useCase.id)
          const isSelected = selectedId === useCase.id

          return (
            <button
              key={useCase.id}
              onClick={() => onSelect(useCase.id)}
              className={`text-left p-4 rounded-lg border-2 transition-all ${
                isSelected
                  ? 'border-primary-500 bg-primary-50 shadow-sm'
                  : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
              }`}
            >
              <div className="flex items-start gap-3">
                <div
                  className={`p-2 rounded-lg ${isSelected ? 'bg-primary-100' : 'bg-gray-100'}`}
                >
                  <Icon
                    className={`w-5 h-5 ${isSelected ? 'text-primary-600' : 'text-gray-500'}`}
                  />
                </div>
                <div className="flex-1 min-w-0">
                  <h3
                    className={`font-medium ${isSelected ? 'text-primary-900' : 'text-gray-900'}`}
                  >
                    {useCase.title}
                  </h3>
                  <p className="text-xs text-gray-500 mt-0.5">{useCase.subtitle}</p>

                  {/* Time comparison */}
                  <div className="mt-3 flex items-center gap-4 text-xs">
                    <div className="flex items-center gap-1 text-gray-400">
                      <Clock className="w-3 h-3" />
                      <span className="line-through">{useCase.traditionalTime}</span>
                    </div>
                    <div
                      className={`flex items-center gap-1 ${isSelected ? 'text-primary-600' : 'text-green-600'}`}
                    >
                      <Zap className="w-3 h-3" />
                      <span className="font-medium">{useCase.newTime}</span>
                    </div>
                  </div>
                </div>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
