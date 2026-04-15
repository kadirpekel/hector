import { useState, useEffect } from 'react';
import { Settings, Check, Loader2, Trash2, Cloud, RefreshCw, Plus, X } from 'lucide-react';
import { useServersStore } from '../store/serversStore';
import { useCloudStore } from '../store/cloudStore';
import { getInstance, updateInstance } from '../services/cloudApi';
import { EnvVarsEditor } from './EnvVarsEditor';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import { Switch } from './ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from './ui/dialog';
import type {
    CloudInstanceConfig,
    CloudServerConfig,
    CloudRateLimitConfig,
    CloudRateLimitRule,
    CloudQueueConfig,
    CloudObservabilityConfig,
    CloudLoggingConfig,
} from '../types';

interface ServerSettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
    serverId: string;
}

export function ServerSettingsModal({ isOpen, onClose, serverId }: ServerSettingsModalProps) {
    const server = useServersStore((s) => s.servers[serverId]);
    const cloudStore = useCloudStore();

    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [destroying, setDestroying] = useState(false);
    const [refreshing, setRefreshing] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Config state
    const [envVars, setEnvVars] = useState<Record<string, string>>({});
    const [authConfig, setAuthConfig] = useState<CloudInstanceConfig['auth']>({});
    const [rateLimit, setRateLimit] = useState<CloudRateLimitConfig>({ enabled: false });
    const [queue, setQueue] = useState<CloudQueueConfig>({});
    const [observability, setObservability] = useState<CloudObservabilityConfig>({});
    const [logging, setLogging] = useState<CloudLoggingConfig>({});

    const instanceId = server?.config.cloud?.instanceId;

    // Load config from cloud on open
    useEffect(() => {
        if (!isOpen || !instanceId) return;
        let cancelled = false;

        const loadConfig = async () => {
            setLoading(true);
            setError(null);
            try {
                const instance = await getInstance(instanceId);
                if (cancelled) return;
                const cfg = instance.config || {};
                setEnvVars(cfg.env || {});
                setAuthConfig(cfg.auth || {});
                const srv = (cfg.server || {}) as CloudServerConfig;
                setRateLimit(srv.rate_limit || { enabled: false });
                setQueue(srv.queue || {});
                setObservability(srv.observability || {});
                setLogging(srv.logging || {});
            } catch (err) {
                if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load config');
            } finally {
                if (!cancelled) setLoading(false);
            }
        };

        loadConfig();
        return () => { cancelled = true; };
    }, [isOpen, instanceId]);

    if (!server) return null;

    const isCloudServer = server.config.type === 'managed-cloud' && !!instanceId;

    const handleSave = async () => {
        if (!instanceId) return;
        setSaving(true);
        setError(null);
        try {
            const config: Partial<CloudInstanceConfig> = {
                env: envVars,
                server: {
                    rate_limit: rateLimit.enabled ? rateLimit : { enabled: false },
                    queue,
                    observability,
                    logging,
                },
            };
            await updateInstance(instanceId, config);
            onClose();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save');
        } finally {
            setSaving(false);
        }
    };

    // Rate limit rule helpers
    const addRateLimitRule = () => {
        setRateLimit(prev => ({
            ...prev,
            limits: [...(prev.limits || []), { type: 'count', window: 'minute', limit: 100 }],
        }));
    };

    const updateRateLimitRule = (idx: number, updates: Partial<CloudRateLimitRule>) => {
        setRateLimit(prev => ({
            ...prev,
            limits: (prev.limits || []).map((r, i) => i === idx ? { ...r, ...updates } : r),
        }));
    };

    const removeRateLimitRule = (idx: number) => {
        setRateLimit(prev => ({
            ...prev,
            limits: (prev.limits || []).filter((_, i) => i !== idx),
        }));
    };

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-[640px] h-[560px] bg-gray-900 border-gray-800 text-gray-300 flex flex-col">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Settings size={20} />
                        Instance Settings
                    </DialogTitle>
                    <DialogDescription>
                        Configure <strong>{server.config.name}</strong>.
                        {isCloudServer && ' Changes will restart the instance.'}
                    </DialogDescription>
                </DialogHeader>

                {loading ? (
                    <div className="flex items-center justify-center py-12">
                        <Loader2 className="animate-spin mr-2" size={20} />
                        <span className="text-sm text-gray-400">Loading configuration...</span>
                    </div>
                ) : !isCloudServer ? (
                    <div className="py-8 text-center text-gray-500 text-sm">
                        Configuration is only available for cloud instances.
                    </div>
                ) : (
                    <Tabs defaultValue="env" className="flex-1 min-h-0 flex flex-col">
                        <TabsList className="bg-gray-800/50 shrink-0">
                            <TabsTrigger value="env" className="text-xs">Environment</TabsTrigger>
                            <TabsTrigger value="auth" className="text-xs">Auth</TabsTrigger>
                            <TabsTrigger value="ratelimit" className="text-xs">Rate Limit</TabsTrigger>
                            <TabsTrigger value="queue" className="text-xs">Queue</TabsTrigger>
                            <TabsTrigger value="observability" className="text-xs">Observability</TabsTrigger>
                            <TabsTrigger value="logging" className="text-xs">Logging</TabsTrigger>
                            <TabsTrigger value="danger" className="text-xs text-red-400">Danger</TabsTrigger>
                        </TabsList>

                        <div className="flex-1 overflow-y-auto min-h-0 mt-2 pr-1">
                            {/* Environment Variables */}
                            <TabsContent value="env" className="m-0 h-full">
                                <EnvVarsEditor envVars={envVars} onChange={setEnvVars} />
                            </TabsContent>

                            {/* Auth (JWKS) — read-only, set at provision */}
                            <TabsContent value="auth" className="m-0 space-y-4 p-1">
                                <p className="text-xs text-gray-500">
                                    Authentication is configured at provision time and cannot be changed after creation.
                                </p>
                                <div className="space-y-3">
                                    <div className="space-y-1">
                                        <Label className="text-xs">JWKS URL</Label>
                                        <Input
                                            value={authConfig?.jwks_url || ''}
                                            className="bg-black/40 border-gray-700 text-xs font-mono"
                                            disabled
                                        />
                                    </div>
                                    <div className="space-y-1">
                                        <Label className="text-xs">Issuer</Label>
                                        <Input
                                            value={authConfig?.issuer || ''}
                                            className="bg-black/40 border-gray-700 text-xs font-mono"
                                            disabled
                                        />
                                    </div>
                                    <div className="space-y-1">
                                        <Label className="text-xs">Audience</Label>
                                        <Input
                                            value={authConfig?.audience || ''}
                                            className="bg-black/40 border-gray-700 text-xs font-mono"
                                            disabled
                                        />
                                    </div>
                                    <div className="space-y-1">
                                        <Label className="text-xs">Client ID</Label>
                                        <Input
                                            value={authConfig?.client_id || ''}
                                            className="bg-black/40 border-gray-700 text-xs font-mono"
                                            disabled
                                        />
                                    </div>
                                </div>
                            </TabsContent>

                            {/* Rate Limiting */}
                            <TabsContent value="ratelimit" className="m-0 space-y-4 p-1">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <Label className="text-sm">Enable Rate Limiting</Label>
                                        <p className="text-xs text-gray-500">Limit requests per session or user.</p>
                                    </div>
                                    <Switch
                                        checked={rateLimit.enabled}
                                        onCheckedChange={(enabled) => setRateLimit(prev => ({ ...prev, enabled }))}
                                    />
                                </div>

                                {rateLimit.enabled && (
                                    <div className="space-y-4 pt-2 border-t border-gray-800">
                                        <div className="space-y-1">
                                            <Label className="text-xs">Scope</Label>
                                            <select
                                                value={rateLimit.scope || 'session'}
                                                onChange={(e) => setRateLimit(prev => ({ ...prev, scope: e.target.value as 'session' | 'user' }))}
                                                className="w-full px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-xs text-white"
                                            >
                                                <option value="session">Per Session</option>
                                                <option value="user">Per User</option>
                                            </select>
                                        </div>

                                        <div className="space-y-2">
                                            <div className="flex items-center justify-between">
                                                <Label className="text-xs">Rules</Label>
                                                <Button variant="ghost" size="sm" className="h-6 text-xs" onClick={addRateLimitRule}>
                                                    <Plus size={12} className="mr-1" /> Add Rule
                                                </Button>
                                            </div>
                                            {(rateLimit.limits || []).map((rule, idx) => (
                                                <div key={idx} className="flex items-center gap-2 p-2 bg-white/5 rounded-lg border border-white/10">
                                                    <select
                                                        value={rule.type}
                                                        onChange={(e) => updateRateLimitRule(idx, { type: e.target.value as 'count' | 'token' })}
                                                        className="px-2 py-1 bg-black/40 border border-gray-700 rounded text-xs text-white"
                                                    >
                                                        <option value="count">Count</option>
                                                        <option value="token">Token</option>
                                                    </select>
                                                    <Input
                                                        type="number"
                                                        value={rule.limit}
                                                        onChange={(e) => updateRateLimitRule(idx, { limit: parseInt(e.target.value) || 0 })}
                                                        className="w-20 h-7 bg-black/40 border-gray-700 text-xs"
                                                        placeholder="Limit"
                                                    />
                                                    <span className="text-xs text-gray-500">per</span>
                                                    <select
                                                        value={rule.window}
                                                        onChange={(e) => updateRateLimitRule(idx, { window: e.target.value as CloudRateLimitRule['window'] })}
                                                        className="px-2 py-1 bg-black/40 border border-gray-700 rounded text-xs text-white"
                                                    >
                                                        <option value="minute">Minute</option>
                                                        <option value="hour">Hour</option>
                                                        <option value="day">Day</option>
                                                        <option value="week">Week</option>
                                                        <option value="month">Month</option>
                                                    </select>
                                                    <Button variant="ghost" size="icon" className="h-6 w-6 shrink-0" onClick={() => removeRateLimitRule(idx)}>
                                                        <X size={12} className="text-red-400" />
                                                    </Button>
                                                </div>
                                            ))}
                                            {(!rateLimit.limits || rateLimit.limits.length === 0) && (
                                                <p className="text-xs text-gray-500 text-center py-2">No rules defined. Add a rule to start limiting.</p>
                                            )}
                                        </div>
                                    </div>
                                )}
                            </TabsContent>

                            {/* Queue */}
                            <TabsContent value="queue" className="m-0 space-y-4 p-1">
                                <p className="text-xs text-gray-500">Configure the background task queue.</p>
                                <div className="grid grid-cols-2 gap-3">
                                    <div className="space-y-1">
                                        <Label className="text-xs">Workers</Label>
                                        <Input
                                            type="number"
                                            value={queue.workers || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, workers: parseInt(e.target.value) || undefined }))}
                                            placeholder="4"
                                            className="bg-black/40 border-gray-700 text-xs"
                                        />
                                    </div>
                                    <div className="space-y-1">
                                        <Label className="text-xs">Max Retries</Label>
                                        <Input
                                            type="number"
                                            value={queue.max_retries || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, max_retries: parseInt(e.target.value) || undefined }))}
                                            placeholder="3"
                                            className="bg-black/40 border-gray-700 text-xs"
                                        />
                                    </div>
                                    <div className="space-y-1">
                                        <Label className="text-xs">Initial Delay</Label>
                                        <Input
                                            value={queue.initial_delay || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, initial_delay: e.target.value || undefined }))}
                                            placeholder="1s"
                                            className="bg-black/40 border-gray-700 text-xs"
                                        />
                                    </div>
                                    <div className="space-y-1">
                                        <Label className="text-xs">Max Delay</Label>
                                        <Input
                                            value={queue.max_delay || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, max_delay: e.target.value || undefined }))}
                                            placeholder="5m"
                                            className="bg-black/40 border-gray-700 text-xs"
                                        />
                                    </div>
                                    <div className="space-y-1 col-span-2">
                                        <Label className="text-xs">Stale Threshold</Label>
                                        <Input
                                            value={queue.stale_threshold || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, stale_threshold: e.target.value || undefined }))}
                                            placeholder="5m"
                                            className="bg-black/40 border-gray-700 text-xs"
                                        />
                                    </div>
                                </div>
                            </TabsContent>

                            {/* Observability */}
                            <TabsContent value="observability" className="m-0 space-y-4 p-1">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <Label className="text-sm">Prometheus Metrics</Label>
                                        <p className="text-xs text-gray-500">Expose /metrics endpoint.</p>
                                    </div>
                                    <Switch
                                        checked={observability.metrics_enabled || false}
                                        onCheckedChange={(checked) => setObservability(prev => ({ ...prev, metrics_enabled: checked }))}
                                    />
                                </div>
                                <div className="space-y-1">
                                    <Label className="text-xs">OTLP Tracing Endpoint</Label>
                                    <Input
                                        value={observability.tracing_endpoint || ''}
                                        onChange={(e) => setObservability(prev => ({ ...prev, tracing_endpoint: e.target.value || undefined }))}
                                        placeholder="http://localhost:4318"
                                        className="bg-black/40 border-gray-700 text-xs font-mono"
                                    />
                                </div>
                            </TabsContent>

                            {/* Logging */}
                            <TabsContent value="logging" className="m-0 space-y-4 p-1">
                                <div className="space-y-1">
                                    <Label className="text-xs">Log Level</Label>
                                    <select
                                        value={logging.level || 'info'}
                                        onChange={(e) => setLogging(prev => ({ ...prev, level: e.target.value as CloudLoggingConfig['level'] }))}
                                        className="w-full px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-xs text-white"
                                    >
                                        <option value="debug">Debug</option>
                                        <option value="info">Info</option>
                                        <option value="warn">Warn</option>
                                        <option value="error">Error</option>
                                    </select>
                                </div>
                                <div className="space-y-1">
                                    <Label className="text-xs">Log Format</Label>
                                    <select
                                        value={logging.format || 'text'}
                                        onChange={(e) => setLogging(prev => ({ ...prev, format: e.target.value as CloudLoggingConfig['format'] }))}
                                        className="w-full px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-xs text-white"
                                    >
                                        <option value="text">Text</option>
                                        <option value="json">JSON</option>
                                    </select>
                                </div>
                            </TabsContent>

                            {/* Danger Zone */}
                            <TabsContent value="danger" className="m-0 p-1">
                                <div className="space-y-4">
                                    {/* Cloud Instance Info */}
                                    {server.config.cloud && (
                                        <div className="space-y-3">
                                            <div className="flex items-center gap-2 text-sm text-gray-400">
                                                <Cloud size={14} />
                                                <span className="font-medium text-gray-300">Cloud Instance</span>
                                            </div>
                                            <div className="grid grid-cols-2 gap-2 text-xs text-gray-400">
                                                {server.config.cloud.instanceId && (
                                                    <>
                                                        <span>Instance ID</span>
                                                        <span className="font-mono text-gray-300 truncate">{server.config.cloud.instanceId}</span>
                                                    </>
                                                )}
                                                {server.config.cloud.region && (
                                                    <>
                                                        <span>Region</span>
                                                        <span className="text-gray-300">{server.config.cloud.region}</span>
                                                    </>
                                                )}
                                                <span>Status</span>
                                                <span className="text-gray-300">{cloudStore.status}</span>
                                            </div>

                                            <div className="flex gap-2 pt-1">
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    className="text-xs"
                                                    disabled={refreshing}
                                                    onClick={async () => {
                                                        setRefreshing(true);
                                                        try { await cloudStore.refreshStatus(); }
                                                        finally { setRefreshing(false); }
                                                    }}
                                                >
                                                    {refreshing ? <Loader2 size={12} className="mr-1 animate-spin" /> : <RefreshCw size={12} className="mr-1" />}
                                                    Refresh
                                                </Button>
                                            </div>
                                        </div>
                                    )}

                                    <div className="border border-red-900/30 rounded-lg p-4 bg-red-900/5 space-y-3">
                                        <h4 className="text-sm font-medium text-red-400">Destroy Instance</h4>
                                        <p className="text-xs text-gray-400">
                                            Permanently destroy your cloud instance and all its data. This action cannot be undone.
                                        </p>
                                        <Button
                                            variant="destructive"
                                            size="sm"
                                            className="text-xs"
                                            disabled={destroying}
                                            onClick={async () => {
                                                if (!confirm('This will permanently destroy your cloud instance and all its data. Continue?')) return;
                                                setDestroying(true);
                                                try {
                                                    await cloudStore.destroy();
                                                    onClose();
                                                } finally {
                                                    setDestroying(false);
                                                }
                                            }}
                                        >
                                            {destroying ? <Loader2 size={12} className="mr-1 animate-spin" /> : <Trash2 size={12} className="mr-1" />}
                                            Destroy Instance
                                        </Button>
                                    </div>
                                </div>
                            </TabsContent>
                        </div>
                    </Tabs>
                )}

                {error && (
                    <div className="text-xs text-red-400 bg-red-900/10 border border-red-900/20 rounded p-2">
                        {error}
                    </div>
                )}

                <DialogFooter>
                    <Button variant="ghost" onClick={onClose} disabled={saving}>
                        Cancel
                    </Button>
                    {isCloudServer && !loading && (
                        <Button
                            onClick={handleSave}
                            disabled={saving}
                            className="bg-hector-green hover:bg-hector-green/80 text-white"
                        >
                            {saving ? <Loader2 className="animate-spin mr-2" size={16} /> : <Check className="mr-2" size={16} />}
                            Save & Restart
                        </Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
