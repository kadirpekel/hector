/**
 * App Selector Component
 *
 * Dropdown to select and manage apps on the connected Hector server.
 * Apps are scoped per server and fetched via the Admin API.
 */

import { useState, useEffect } from 'react'
import { Plus, Box, Check, ChevronDown, Trash2, Loader2 } from 'lucide-react'
import { useAppsStore } from '../store/appsStore'
import { useServersStore } from '../store/serversStore'
import { useStore } from '../store/useStore'
import { cn } from '../lib/utils'
import { Button } from './ui/button'
import { Input } from './ui/input'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger
} from './ui/dropdown-menu'

interface AppSelectorProps {
    className?: string
    variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link"
}

export function AppSelector({ className, variant = "outline" }: AppSelectorProps) {
    const [isOpen, setIsOpen] = useState(false)
    const [showCreateForm, setShowCreateForm] = useState(false)
    const [newAppName, setNewAppName] = useState('')
    const [creating, setCreating] = useState(false)

    const activeServer = useServersStore((s) => s.getActiveServer())
    const loading = useAppsStore((s) => s.loading)
    const loadApps = useAppsStore((s) => s.loadApps)
    const createApp = useAppsStore((s) => s.createApp)
    const deleteApp = useAppsStore((s) => s.deleteApp)
    const selectApp = useAppsStore((s) => s.selectApp)

    // Get server-scoped data - use selectors to avoid creating new arrays
    const serverId = activeServer?.config.id
    const serverData = useAppsStore((s) => serverId ? s.appsByServer[serverId] : undefined)
    const apps = serverData?.apps || []
    const activeAppId = serverData?.activeAppId || null
    const activeApp = apps.find((a) => a.id === activeAppId)

    // Load apps when server becomes authenticated or changes
    useEffect(() => {
        if (activeServer?.status === 'authenticated' && serverId) {
            loadApps(serverId)
        }
    }, [activeServer?.status, serverId, loadApps])

    const handleCreateApp = async (e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        if (!newAppName.trim() || !serverId) return

        setCreating(true)
        try {
            // Pass empty config - hector backend generates a default config
            await createApp(serverId, newAppName.trim(), '')
            setNewAppName('')
            setShowCreateForm(false)

            // Force reload agents for the new app
            useStore.getState().reloadAgents()

            useStore.getState().setSuccessMessage(`App "${newAppName.trim()}" created`)
        } catch (error) {
            useStore.getState().setError(`Failed to create app: ${(error as Error).message}`)
        } finally {
            setCreating(false)
        }
    }

    const handleSelectApp = async (appId: string) => {
        if (!serverId) return
        try {
            selectApp(serverId, appId)
            setIsOpen(false)
        } catch (error) {
            useStore.getState().setError(`Failed to select app: ${(error as Error).message}`)
        }
    }

    const handleDeleteApp = async (appId: string, e: React.MouseEvent) => {
        e.stopPropagation()
        if (!serverId) return

        const app = apps.find((a) => a.id === appId)
        if (!app) return

        if (!confirm(`Delete app "${app.name}"?\n\nThis cannot be undone.`)) {
            return
        }

        try {
            await deleteApp(serverId, appId)
            useStore.getState().setSuccessMessage(`App "${app.name}" deleted`)
        } catch (error) {
            useStore.getState().setError(`Failed to delete app: ${(error as Error).message}`)
        }
    }

    // Don't show if not connected
    if (!activeServer || activeServer.status !== 'authenticated') {
        return null
    }

    return (
        <DropdownMenu open={isOpen} onOpenChange={setIsOpen}>
            <DropdownMenuTrigger asChild>
                <Button
                    variant={variant}
                    className={cn("min-w-0 w-40 justify-between px-3", className)}
                >
                    <div className="flex items-center gap-2">
                        <Box size={14} className="text-purple-400" />
                        <span className="truncate max-w-[100px]">{activeApp?.name || activeAppId || 'Select App'}</span>
                    </div>
                    <div className="flex items-center gap-2">
                        {loading && <Loader2 size={12} className="animate-spin" />}
                        <ChevronDown size={14} className="opacity-50" />
                    </div>
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-[240px] bg-gray-900 border-gray-800 text-gray-300">
                <DropdownMenuLabel className="flex items-center gap-2">
                    <Box size={14} />
                    Apps
                </DropdownMenuLabel>
                <DropdownMenuSeparator className="bg-gray-800" />
                <div className="max-h-[200px] overflow-y-auto">
                    {apps.length === 0 ? (
                        <div className="p-2 text-sm text-center text-gray-500">
                            {loading ? 'Loading...' : 'No apps found'}
                        </div>
                    ) : (
                        apps.map((app) => (
                            <DropdownMenuItem
                                key={app.id}
                                onSelect={() => handleSelectApp(app.id)}
                                className={cn(
                                    'flex items-center justify-between cursor-pointer focus:bg-gray-800 focus:text-white',
                                    activeAppId === app.id && 'bg-gray-800/50'
                                )}
                            >
                                <div className="flex items-center gap-2 overflow-hidden">
                                    <div className="flex flex-col min-w-0">
                                        <span className="font-medium truncate text-xs">{app.name}</span>
                                        <span className="text-[10px] text-gray-500 truncate">{app.id}</span>
                                    </div>
                                </div>
                                <div className="flex items-center gap-1">
                                    {activeAppId === app.id && <Check size={14} className="text-purple-400" />}
                                    {/* Don't allow deleting the default app */}
                                    {app.id !== 'default' && (
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="h-6 w-6 hover:bg-red-900/50 hover:text-red-400"
                                            onClick={(e) => handleDeleteApp(app.id, e)}
                                        >
                                            <Trash2 size={12} />
                                        </Button>
                                    )}
                                </div>
                            </DropdownMenuItem>
                        ))
                    )}
                </div>
                <DropdownMenuSeparator className="bg-gray-800" />
                {showCreateForm ? (
                    <div className="p-2 space-y-2">
                        <Input
                            placeholder="App Name"
                            value={newAppName}
                            onChange={(e) => setNewAppName(e.target.value)}
                            className="h-7 text-xs bg-black/40 border-gray-700"
                            onClick={(e) => e.stopPropagation()}
                            onKeyDown={(e) => {
                                e.stopPropagation()
                                if (e.key === 'Enter') handleCreateApp(e as unknown as React.MouseEvent)
                            }}
                        />
                        <div className="flex gap-2">
                            <Button
                                size="sm"
                                variant="default"
                                className="h-7 text-xs w-full bg-purple-600 hover:bg-purple-500 text-white"
                                onClick={handleCreateApp}
                                disabled={creating}
                            >
                                {creating ? <Loader2 className="animate-spin" size={12} /> : 'Create'}
                            </Button>
                            <Button
                                size="sm"
                                variant="ghost"
                                className="h-7 text-xs w-full hover:bg-gray-800"
                                onClick={(e) => {
                                    e.stopPropagation()
                                    setShowCreateForm(false)
                                }}
                            >
                                Cancel
                            </Button>
                        </div>
                    </div>
                ) : (
                    <DropdownMenuItem
                        onSelect={(e) => {
                            e.preventDefault()
                            setShowCreateForm(true)
                        }}
                        className="cursor-pointer focus:bg-gray-800 focus:text-white"
                    >
                        <Plus size={14} className="mr-2" />
                        Create App
                    </DropdownMenuItem>
                )}
            </DropdownMenuContent>
        </DropdownMenu>
    )
}
