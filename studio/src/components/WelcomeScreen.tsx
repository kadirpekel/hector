import hectorIcon from '../assets/hector.png'

export function WelcomeScreen() {
  return (
    <div className="flex-1 flex items-center justify-center text-gray-500 bg-gray-900/20">
      <div className="text-center max-w-md px-6">
        <div className="w-16 h-16 flex items-center justify-center mx-auto mb-4">
          <img src={hectorIcon} alt="Hector" className="w-full h-full object-contain" />
        </div>
        <h2 className="text-xl font-medium text-gray-200 mb-2">Welcome to Hector Studio</h2>
        <p className="text-sm text-gray-400">
          Waiting for connection to Hector server...
        </p>
      </div>
    </div>
  )
}
