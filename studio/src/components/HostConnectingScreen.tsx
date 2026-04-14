import hectorIcon from '../assets/hector.png'
import type { ServerState } from '../types'

interface HostConnectingScreenProps {
  server: ServerState | undefined
  onRetry: () => void
}

export function HostConnectingScreen({ server, onRetry }: HostConnectingScreenProps) {
  const hasError = server?.status === 'unreachable' || server?.status === 'error'

  return (
    <div className="flex-1 flex items-center justify-center text-gray-500 bg-gray-900/20">
      <div className="text-center max-w-md px-6">
        <div className="w-16 h-16 flex items-center justify-center mx-auto mb-4">
          <img src={hectorIcon} alt="Hector" className="w-full h-full object-contain" />
        </div>
        <h2 className="text-xl font-medium text-gray-200 mb-2">Connecting to Hector...</h2>
        {hasError ? (
          <>
            <p className="text-sm text-red-400 mb-4">
              {server?.lastError || 'Could not connect to the server.'}
            </p>
            <button
              onClick={onRetry}
              className="px-5 py-3 rounded-lg font-medium text-sm transition-all bg-hector-green hover:bg-hector-green/80 text-white mx-auto"
            >
              Retry Connection
            </button>
          </>
        ) : (
          <p className="text-sm text-gray-400">Please wait...</p>
        )}
      </div>
    </div>
  )
}
