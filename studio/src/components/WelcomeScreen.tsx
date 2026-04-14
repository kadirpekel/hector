import { Globe, Cloud } from 'lucide-react'
import hectorIcon from '../assets/hector.png'
import { CLOUD_ENABLED } from '../lib/cloudEnabled'

interface WelcomeScreenProps {
  isCloudAuthenticated: boolean | null
  cloudStatus: string
  onAddServer: () => void
  onConnectCloud: () => void
}

export function WelcomeScreen({ isCloudAuthenticated, cloudStatus, onAddServer, onConnectCloud }: WelcomeScreenProps) {
  // Show cloud button when: not yet authenticated, OR authenticated but not connected/working
  const showCloudButton = CLOUD_ENABLED && cloudStatus !== 'connected' && cloudStatus !== 'working'
  const cloudButtonLabel = isCloudAuthenticated ? 'Connect to Cloud' : 'Set Up Cloud'

  return (
    <div className="flex-1 flex items-center justify-center text-gray-500 bg-gray-900/20">
      <div className="text-center max-w-md px-6">
        <div className="w-16 h-16 flex items-center justify-center mx-auto mb-4">
          <img src={hectorIcon} alt="Hector" className="w-full h-full object-contain" />
        </div>
        <h2 className="text-xl font-medium text-gray-200 mb-2">Welcome to Hector Studio</h2>
        <p className="text-sm text-gray-400 mb-6">
          Connect to a Hector server to get started.
        </p>
        <button
          onClick={onAddServer}
          className="flex items-center justify-center gap-2 px-5 py-3 rounded-lg font-medium text-sm transition-all bg-hector-green hover:bg-hector-green/80 text-white mx-auto"
        >
          <Globe size={16} />
          <span>Add Server</span>
        </button>
        {showCloudButton && (
          <button
            onClick={onConnectCloud}
            className="flex items-center justify-center gap-2 px-5 py-3 rounded-lg font-medium text-sm transition-all bg-white/5 hover:bg-white/10 text-gray-300 border border-white/10 mx-auto mt-3"
          >
            <Cloud size={16} />
            <span>{cloudButtonLabel}</span>
          </button>
        )}
      </div>
    </div>
  )
}
