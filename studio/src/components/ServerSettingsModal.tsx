import { useState } from 'react';
import { Settings, Check, Loader2, Plus, X, Shield } from 'lucide-react';
import { useServersStore } from '../store/serversStore';
import { useStore } from '../store/useStore';
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
import { apiFetch } from '../lib/api-utils';
import type {
    RateLimitConfig,
    RateLimitRule,
    QueueConfig,
    ObservabilityConfig,
    LoggingConfig,
} from '../types';

interface ServerSettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export function ServerSettingsModal({ isOpen, onClose }: ServerSettingsModalProps) {
    const server = useServersStore((s) => s.getActiveServer());

    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    // Config state
    const [envVars, setEnvVars] = useState<Record<string, string>>({});
    const [authConfig, setAuthConfig] = useState({
        jwksUrl: '',
        issuer: '',
        audience: '',
        clientId: '',
    });
    const [rateLimit, setRateLimit] = useState<RateLimitConfig>({ enabled: false });
    const [queue, setQueue] = useState<QueueConfig>({});
    const [observability, setObservability] = useState<ObservabilityConfig>({});
    const [logging, setLogging] = useState<LoggingConfig>({});

    if (!server) return null;

    // Convert structured config to flat env vars for the reload endpoint
    const buildEnvOverrides = (): Record<string, string> => {
        const env: Record<string, string> = { ...envVars };

        // Auth
        if (authConfig.jwksUrl) {
            env['HECTOR_AUTH_JWKS_URL'] = authConfig.jwksUrl;
            if (authConfig.issuer) env['HECTOR_AUTH_ISSUER'] = authConfig.issuer;
            if (authConfig.audience) env['HECTOR_AUTH_AUDIENCE'] = authConfig.audience;
            if (authConfig.clientId) env['HECTOR_AUTH_CLIENT_ID'] = authConfig.clientId;
        }

        // Rate limiting
        if (rateLimit.enabled) {
            env['HECTOR_RATE_LIMIT_ENABLED'] = 'true';
            if (rateLimit.scope) env['HECTOR_RATE_LIMIT_SCOPE'] = rateLimit.scope;
            if (rateLimit.limits && rateLimit.limits.length > 0) {
                env['HECTOR_RATE_LIMIT_LIMITS'] = JSON.stringify(rateLimit.limits);
            }
        }

        // Queue
        if (queue.workers) env['HECTOR_QUEUE_WORKERS'] = String(queue.workers);
        if (queue.max_retries) env['HECTOR_QUEUE_MAX_RETRIES'] = String(queue.max_retries);
        if (queue.initial_delay) env['HECTOR_QUEUE_INITIAL_DELAY'] = queue.initial_delay;
        if (queue.max_delay) env['HECTOR_QUEUE_MAX_DELAY'] = queue.max_delay;
        if (queue.stale_threshold) env['HECTOR_QUEUE_STALE_THRESHOLD'] = queue.stale_threshold;

        // Observability
        if (observability.tracing_endpoint) env['HECTOR_TRACING_ENDPOINT'] = observability.tracing_endpoint;

        // Logging
        if (logging.level) env['HECTOR_LOG_LEVEL'] = logging.level;
        if (logging.format) env['HECTOR_LOG_FORMAT'] = logging.format;

        return env;
    };

    const handleSave = async () => {
        setSaving(true);
        setError(null);
        setSuccess(null);
        try {
            const env = buildEnvOverrides();
            const filteredEnv: Record<string, string> = {};
            for (const [key, value] of Object.entries(env)) {
                if (value.length > 0) filteredEnv[key] = value;
            }

            const body = Object.keys(filteredEnv).length > 0
                ? JSON.stringify({ env: filteredEnv })
                : undefined;

            const res = await apiFetch('/admin/reload', {
                method: 'POST',
                ...(body ? { headers: { 'Content-Type': 'application/json' }, body } : {}),
            });

            if (!res.ok) {
                const data = await res.json().catch(() => ({ error: res.statusText }));
                throw new Error(data.error || `HTTP ${res.status}`);
            }

            setSuccess('Settings applied and server reloaded.');
            setTimeout(() => useStore.getState().reloadAgents(), 500);
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

    const updateRateLimitRule = (idx: number, updates: Partial<RateLimitRule>) => {
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
            <DialogContent className="sm:max-w-[800px] h-[600px] bg-gray-900 border-gray-800 text-gray-300 flex flex-col p-0 gap-0 overflow-hidden">
                <DialogHeader className="p-6 pb-2 border-b border-gray-800 shrink-0">
                    <DialogTitle className="flex items-center gap-2">
                        <Settings size={20} />
                        Server Settings
                    </DialogTitle>
                    <DialogDescription>
                        Configure settings for <strong>{server.config.name}</strong>.
                    </DialogDescription>
                </DialogHeader>

                <Tabs defaultValue="general" className="flex-1 flex flex-col min-h-0">
                    <div className="px-6 py-2 border-b border-gray-800 bg-black/20 shrink-0">
                        <TabsList className="bg-gray-800/50">
                            <TabsTrigger value="general">General</TabsTrigger>
                            <TabsTrigger value="env">Environment</TabsTrigger>
                            <TabsTrigger value="ratelimit">Rate Limit</TabsTrigger>
                            <TabsTrigger value="queue">Queue</TabsTrigger>
                            <TabsTrigger value="observability">Observability</TabsTrigger>
                            <TabsTrigger value="logging">Logging</TabsTrigger>
                        </TabsList>
                    </div>

                    <div className="flex-1 overflow-y-auto min-h-0">
                        {/* GENERAL TAB */}
                        <TabsContent value="general" className="m-0 p-6 space-y-6">
                            <div className="space-y-4">
                                {/* Authentication */}
                                <div className="border border-gray-800 rounded-lg p-4 space-y-4 bg-black/20">
                                    <h3 className="text-sm font-medium text-gray-400 uppercase tracking-wider flex items-center gap-2">
                                        <Shield size={14} />
                                        Authentication (JWKS)
                                    </h3>
                                    <div className="space-y-4">
                                        <div className="space-y-2">
                                            <Label htmlFor="jwks-url">JWKS URL</Label>
                                            <Input
                                                id="jwks-url"
                                                placeholder="https://your-domain.com/.well-known/jwks.json"
                                                value={authConfig.jwksUrl}
                                                onChange={(e) => setAuthConfig({ ...authConfig, jwksUrl: e.target.value })}
                                                className="bg-black/40 border-gray-700"
                                            />
                                            <p className="text-xs text-gray-500">
                                                The URL to your JSON Web Key Set for verifying JWTs.
                                            </p>
                                        </div>

                                        <div className="grid grid-cols-2 gap-4">
                                            <div className="space-y-2">
                                                <Label htmlFor="issuer">Issuer</Label>
                                                <Input
                                                    id="issuer"
                                                    placeholder="https://your-domain.com/"
                                                    value={authConfig.issuer}
                                                    onChange={(e) => setAuthConfig({ ...authConfig, issuer: e.target.value })}
                                                    className="bg-black/40 border-gray-700"
                                                />
                                            </div>
                                            <div className="space-y-2">
                                                <Label htmlFor="audience">Audience</Label>
                                                <Input
                                                    id="audience"
                                                    placeholder="hector-app"
                                                    value={authConfig.audience}
                                                    onChange={(e) => setAuthConfig({ ...authConfig, audience: e.target.value })}
                                                    className="bg-black/40 border-gray-700"
                                                />
                                            </div>
                                        </div>
                                        <div className="space-y-2">
                                            <Label htmlFor="client-id">Client ID (Optional)</Label>
                                            <Input
                                                id="client-id"
                                                placeholder="client_xyz"
                                                value={authConfig.clientId}
                                                onChange={(e) => setAuthConfig({ ...authConfig, clientId: e.target.value })}
                                                className="bg-black/40 border-gray-700"
                                            />
                                        </div>
                                    </div>
                                </div>

                                {/* Logging */}
                                <div className="border border-gray-800 rounded-lg p-4 space-y-4 bg-black/20">
                                    <h3 className="text-sm font-medium text-gray-400 uppercase tracking-wider">Logging</h3>
                                    <div className="grid grid-cols-2 gap-4">
                                        <div className="space-y-2">
                                            <Label htmlFor="log-level">Log Level</Label>
                                            <select
                                                id="log-level"
                                                value={logging.level || 'info'}
                                                onChange={(e) => setLogging(prev => ({ ...prev, level: e.target.value as LoggingConfig['level'] }))}
                                                className="w-full px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-sm text-white"
                                            >
                                                <option value="debug">Debug</option>
                                                <option value="info">Info</option>
                                                <option value="warn">Warn</option>
                                                <option value="error">Error</option>
                                            </select>
                                        </div>
                                        <div className="space-y-2">
                                            <Label htmlFor="log-format">Log Format</Label>
                                            <select
                                                id="log-format"
                                                value={logging.format || 'text'}
                                                onChange={(e) => setLogging(prev => ({ ...prev, format: e.target.value as LoggingConfig['format'] }))}
                                                className="w-full px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-sm text-white"
                                            >
                                                <option value="text">Text</option>
                                                <option value="json">JSON</option>
                                            </select>
                                        </div>
                                    </div>
                                </div>

                                {/* Observability */}
                                <div className="border border-gray-800 rounded-lg p-4 space-y-4 bg-black/20">
                                    <h3 className="text-sm font-medium text-gray-400 uppercase tracking-wider">Observability</h3>
                                    <div className="space-y-2">
                                        <Label htmlFor="tracing-endpoint">OTLP Tracing Endpoint</Label>
                                        <Input
                                            id="tracing-endpoint"
                                            value={observability.tracing_endpoint || ''}
                                            onChange={(e) => setObservability(prev => ({ ...prev, tracing_endpoint: e.target.value || undefined }))}
                                            placeholder="http://localhost:4318"
                                            className="bg-black/40 border-gray-700 font-mono"
                                        />
                                        <p className="text-xs text-gray-500">
                                            OpenTelemetry collector endpoint for distributed tracing.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </TabsContent>

                        {/* ENVIRONMENT TAB */}
                        <TabsContent value="env" className="m-0 flex flex-col h-full bg-black/20">
                            <div className="p-4 bg-gray-900/50 border-b border-gray-800">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <h3 className="text-sm font-medium text-gray-200">Environment Variables</h3>
                                        <p className="text-xs text-gray-500 mt-1">
                                            Variables are applied on save and trigger a configuration reload.
                                        </p>
                                    </div>
                                </div>
                            </div>
                            <div className="flex-1 p-4 overflow-hidden">
                                <div className="h-full flex flex-col">
                                    <EnvVarsEditor envVars={envVars} onChange={setEnvVars} />
                                </div>
                            </div>
                        </TabsContent>

                        {/* RATE LIMIT TAB */}
                        <TabsContent value="ratelimit" className="m-0 p-6 space-y-6">
                            <div className="flex items-center justify-between space-x-2 border border-gray-800 rounded-lg p-3 bg-black/20">
                                <div className="space-y-0.5">
                                    <Label htmlFor="rate-limit-toggle" className="text-base font-medium text-gray-200">
                                        Enable Rate Limiting
                                    </Label>
                                    <div className="text-xs text-gray-400">
                                        Limit requests per session or user.
                                    </div>
                                </div>
                                <Switch
                                    id="rate-limit-toggle"
                                    checked={rateLimit.enabled}
                                    onCheckedChange={(enabled) => setRateLimit(prev => ({ ...prev, enabled }))}
                                />
                            </div>

                            {rateLimit.enabled && (
                                <div className="space-y-4">
                                    <div className="space-y-2">
                                        <Label>Scope</Label>
                                        <select
                                            value={rateLimit.scope || 'session'}
                                            onChange={(e) => setRateLimit(prev => ({ ...prev, scope: e.target.value as 'session' | 'user' }))}
                                            className="w-full px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-sm text-white"
                                        >
                                            <option value="session">Per Session</option>
                                            <option value="user">Per User</option>
                                        </select>
                                    </div>

                                    <div className="space-y-3">
                                        <div className="flex items-center justify-between">
                                            <Label>Rules</Label>
                                            <Button variant="ghost" size="sm" className="h-7" onClick={addRateLimitRule}>
                                                <Plus size={14} className="mr-1" /> Add Rule
                                            </Button>
                                        </div>
                                        {(rateLimit.limits || []).map((rule, idx) => (
                                            <div key={idx} className="flex items-center gap-2 p-3 bg-black/20 rounded-lg border border-gray-800">
                                                <select
                                                    value={rule.type}
                                                    onChange={(e) => updateRateLimitRule(idx, { type: e.target.value as 'count' | 'token' })}
                                                    className="px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-sm text-white"
                                                >
                                                    <option value="count">Count</option>
                                                    <option value="token">Token</option>
                                                </select>
                                                <Input
                                                    type="number"
                                                    value={rule.limit}
                                                    onChange={(e) => updateRateLimitRule(idx, { limit: parseInt(e.target.value) || 0 })}
                                                    className="w-24 bg-black/40 border-gray-700"
                                                    placeholder="Limit"
                                                />
                                                <span className="text-sm text-gray-500">per</span>
                                                <select
                                                    value={rule.window}
                                                    onChange={(e) => updateRateLimitRule(idx, { window: e.target.value as RateLimitRule['window'] })}
                                                    className="px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-sm text-white"
                                                >
                                                    <option value="minute">Minute</option>
                                                    <option value="hour">Hour</option>
                                                    <option value="day">Day</option>
                                                    <option value="week">Week</option>
                                                    <option value="month">Month</option>
                                                </select>
                                                <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => removeRateLimitRule(idx)}>
                                                    <X size={14} className="text-red-400" />
                                                </Button>
                                            </div>
                                        ))}
                                        {(!rateLimit.limits || rateLimit.limits.length === 0) && (
                                            <div className="text-sm text-gray-500 text-center py-4 border border-gray-800 border-dashed rounded-lg">
                                                No rules defined. Add a rule to start limiting.
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}
                        </TabsContent>

                        {/* QUEUE TAB */}
                        <TabsContent value="queue" className="m-0 p-6 space-y-6">
                            <div className="border border-gray-800 rounded-lg p-4 space-y-4 bg-black/20">
                                <h3 className="text-sm font-medium text-gray-400 uppercase tracking-wider">Task Queue</h3>
                                <p className="text-xs text-gray-500">Configure the background task queue for persistent tasks.</p>
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <Label htmlFor="queue-workers">Workers</Label>
                                        <Input
                                            id="queue-workers"
                                            type="number"
                                            value={queue.workers || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, workers: parseInt(e.target.value) || undefined }))}
                                            placeholder="4"
                                            className="bg-black/40 border-gray-700"
                                        />
                                    </div>
                                    <div className="space-y-2">
                                        <Label htmlFor="queue-retries">Max Retries</Label>
                                        <Input
                                            id="queue-retries"
                                            type="number"
                                            value={queue.max_retries || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, max_retries: parseInt(e.target.value) || undefined }))}
                                            placeholder="3"
                                            className="bg-black/40 border-gray-700"
                                        />
                                    </div>
                                    <div className="space-y-2">
                                        <Label htmlFor="queue-init-delay">Initial Delay</Label>
                                        <Input
                                            id="queue-init-delay"
                                            value={queue.initial_delay || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, initial_delay: e.target.value || undefined }))}
                                            placeholder="1s"
                                            className="bg-black/40 border-gray-700"
                                        />
                                    </div>
                                    <div className="space-y-2">
                                        <Label htmlFor="queue-max-delay">Max Delay</Label>
                                        <Input
                                            id="queue-max-delay"
                                            value={queue.max_delay || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, max_delay: e.target.value || undefined }))}
                                            placeholder="5m"
                                            className="bg-black/40 border-gray-700"
                                        />
                                    </div>
                                    <div className="space-y-2 col-span-2">
                                        <Label htmlFor="queue-stale">Stale Threshold</Label>
                                        <Input
                                            id="queue-stale"
                                            value={queue.stale_threshold || ''}
                                            onChange={(e) => setQueue(prev => ({ ...prev, stale_threshold: e.target.value || undefined }))}
                                            placeholder="5m"
                                            className="bg-black/40 border-gray-700"
                                        />
                                    </div>
                                </div>
                            </div>
                        </TabsContent>

                        {/* OBSERVABILITY TAB */}
                        <TabsContent value="observability" className="m-0 p-6 space-y-6">
                            <div className="border border-gray-800 rounded-lg p-4 space-y-4 bg-black/20">
                                <h3 className="text-sm font-medium text-gray-400 uppercase tracking-wider">Tracing</h3>
                                <div className="space-y-2">
                                    <Label htmlFor="obs-tracing">OTLP Tracing Endpoint</Label>
                                    <Input
                                        id="obs-tracing"
                                        value={observability.tracing_endpoint || ''}
                                        onChange={(e) => setObservability(prev => ({ ...prev, tracing_endpoint: e.target.value || undefined }))}
                                        placeholder="http://localhost:4318"
                                        className="bg-black/40 border-gray-700 font-mono"
                                    />
                                    <p className="text-xs text-gray-500">
                                        OpenTelemetry collector endpoint for distributed tracing.
                                    </p>
                                </div>
                            </div>
                        </TabsContent>

                        {/* LOGGING TAB */}
                        <TabsContent value="logging" className="m-0 p-6 space-y-6">
                            <div className="border border-gray-800 rounded-lg p-4 space-y-4 bg-black/20">
                                <h3 className="text-sm font-medium text-gray-400 uppercase tracking-wider">Logging</h3>
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <Label htmlFor="log-level-tab">Log Level</Label>
                                        <select
                                            id="log-level-tab"
                                            value={logging.level || 'info'}
                                            onChange={(e) => setLogging(prev => ({ ...prev, level: e.target.value as LoggingConfig['level'] }))}
                                            className="w-full px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-sm text-white"
                                        >
                                            <option value="debug">Debug</option>
                                            <option value="info">Info</option>
                                            <option value="warn">Warn</option>
                                            <option value="error">Error</option>
                                        </select>
                                    </div>
                                    <div className="space-y-2">
                                        <Label htmlFor="log-format-tab">Log Format</Label>
                                        <select
                                            id="log-format-tab"
                                            value={logging.format || 'text'}
                                            onChange={(e) => setLogging(prev => ({ ...prev, format: e.target.value as LoggingConfig['format'] }))}
                                            className="w-full px-3 py-2 bg-black/40 border border-gray-700 rounded-md text-sm text-white"
                                        >
                                            <option value="text">Text</option>
                                            <option value="json">JSON</option>
                                        </select>
                                    </div>
                                </div>
                            </div>
                        </TabsContent>
                    </div>

                    {error && (
                        <div className="mx-6 mb-2 text-xs text-red-400 bg-red-900/10 border border-red-900/20 rounded p-2">
                            {error}
                        </div>
                    )}
                    {success && (
                        <div className="mx-6 mb-2 text-xs text-green-400 bg-green-900/10 border border-green-900/20 rounded p-2">
                            {success}
                        </div>
                    )}

                    <DialogFooter className="p-6 border-t border-gray-800 shrink-0 bg-gray-900 z-10">
                        <Button variant="ghost" onClick={onClose} disabled={saving}>
                            Cancel
                        </Button>
                        <Button
                            onClick={handleSave}
                            disabled={saving}
                            className="bg-hector-green hover:bg-hector-green/80 text-white"
                        >
                            {saving ? <Loader2 className="animate-spin mr-2" size={16} /> : <Check className="mr-2" size={16} />}
                            Save Changes
                        </Button>
                    </DialogFooter>
                </Tabs>
            </DialogContent>
        </Dialog>
    );
}
