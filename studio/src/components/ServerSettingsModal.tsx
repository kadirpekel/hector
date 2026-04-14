import { useState, useEffect } from 'react';
import { Settings, Check, Loader2, Trash2, Cloud, RefreshCw } from 'lucide-react';
import { useServersStore } from '../store/serversStore';
import { useCloudStore } from '../store/cloudStore';
import { CLOUD_ENABLED } from '../lib/cloudEnabled';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Label } from './ui/label';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from './ui/dialog';

interface ServerSettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
    serverId: string;
}

export function ServerSettingsModal({ isOpen, onClose, serverId }: ServerSettingsModalProps) {
    const server = useServersStore((s) => s.servers[serverId]);
    const updateServer = useServersStore((s) => s.updateServer);
    const cloudStore = useCloudStore();

    const [name, setName] = useState('');
    const [url, setUrl] = useState('');
    const [adminKey, setAdminKey] = useState('');
    const [saving, setSaving] = useState(false);
    const [destroying, setDestroying] = useState(false);
    const [refreshing, setRefreshing] = useState(false);

    const isCloudServer = CLOUD_ENABLED && server?.config.type === 'managed-cloud';

    useEffect(() => {
        if (isOpen && server) {
            setName(server.config.name);
            setUrl(server.config.url);
            setAdminKey(server.config.adminKey || '');
        }
    }, [isOpen, server]);

    if (!server) return null;

    const handleSave = async (e?: React.FormEvent) => {
        if (e) e.preventDefault();
        setSaving(true);
        try {
            updateServer({
                ...server,
                config: {
                    ...server.config,
                    name: name.trim(),
                    url: url.trim().replace(/\/+$/, ''),
                    adminKey: adminKey.trim() || undefined,
                },
            });
            onClose();
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open={isOpen} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-[480px] bg-gray-900 border-gray-800 text-gray-300">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Settings size={20} />
                        Server Settings
                    </DialogTitle>
                    <DialogDescription>
                        Configure settings for <strong>{server.config.name}</strong>.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="name">Server Name</Label>
                        <Input
                            id="name"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            className="bg-black/40 border-gray-700"
                        />
                    </div>
                    <div className="space-y-2">
                        <Label htmlFor="url">URL</Label>
                        <Input
                            id="url"
                            value={url}
                            onChange={(e) => setUrl(e.target.value)}
                            className="bg-black/40 border-gray-700"
                            disabled={isCloudServer}
                        />
                    </div>
                    <div className="space-y-2">
                        <Label htmlFor="adminKey">Admin Key</Label>
                        <Input
                            id="adminKey"
                            type="password"
                            placeholder="Enter admin key"
                            value={adminKey}
                            onChange={(e) => setAdminKey(e.target.value)}
                            className="bg-black/40 border-gray-700"
                            disabled={isCloudServer}
                        />
                    </div>

                    {/* Cloud Instance Info */}
                    {isCloudServer && server.config.cloud && (
                        <div className="space-y-3 pt-2 border-t border-gray-800">
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
                                        try {
                                            await cloudStore.refreshStatus();
                                        } finally {
                                            setRefreshing(false);
                                        }
                                    }}
                                >
                                    {refreshing ? <Loader2 size={12} className="mr-1 animate-spin" /> : <RefreshCw size={12} className="mr-1" />}
                                    Refresh
                                </Button>
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
                    )}
                </div>

                <DialogFooter>
                    <Button variant="ghost" onClick={onClose} disabled={saving}>
                        Cancel
                    </Button>
                    <Button
                        onClick={() => handleSave()}
                        disabled={saving || !name.trim() || !url.trim()}
                        className="bg-hector-green hover:bg-hector-green/80 text-white"
                    >
                        {saving ? <Loader2 className="animate-spin mr-2" size={16} /> : <Check className="mr-2" size={16} />}
                        Save
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
