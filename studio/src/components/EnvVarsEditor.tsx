import React, { useState } from 'react';
import { Plus, Trash2, Eye, EyeOff, Globe, FileText, List, Copy, Check } from 'lucide-react';
import { cn } from '../lib/utils';

interface EnvVar {
    key: string;
    value: string;
    isGlobal?: boolean; // Whether this var is inherited from global
}

interface EnvVarsEditorProps {
    envVars: Record<string, string>;
    globalEnvVars?: Record<string, string>; // For showing inherited globals
    onChange: (envVars: Record<string, string>) => void;
    showInherited?: boolean; // Whether to show inherited global vars
}

export const EnvVarsEditor: React.FC<EnvVarsEditorProps> = ({
    envVars,
    globalEnvVars = {},
    onChange,
    showInherited = false,
}) => {
    const [revealedKeys, setRevealedKeys] = useState<Set<string>>(new Set());
    const [newKey, setNewKey] = useState('');
    const [newValue, setNewValue] = useState('');
    const [isRawMode, setIsRawMode] = useState(false);
    const [rawText, setRawText] = useState('');
    const [rawError, setRawError] = useState<string | null>(null);
    const [copied, setCopied] = useState(false);

    // Combine inherited globals with server vars for display
    const displayVars: EnvVar[] = [];

    if (showInherited) {
        // Add globals that aren't overridden
        for (const [key, value] of Object.entries(globalEnvVars)) {
            if (!(key in envVars)) {
                displayVars.push({ key, value, isGlobal: true });
            }
        }
    }

    // Add server vars
    for (const [key, value] of Object.entries(envVars)) {
        displayVars.push({ key, value, isGlobal: false });
    }

    // Sort by key
    displayVars.sort((a, b) => a.key.localeCompare(b.key));

    const toggleReveal = (key: string) => {
        const newRevealed = new Set(revealedKeys);
        if (newRevealed.has(key)) {
            newRevealed.delete(key);
        } else {
            newRevealed.add(key);
        }
        setRevealedKeys(newRevealed);
    };

    const handleAdd = () => {
        const trimmedKey = newKey.trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
        if (!trimmedKey || trimmedKey in envVars) return;

        onChange({ ...envVars, [trimmedKey]: newValue });
        setNewKey('');
        setNewValue('');
    };

    const handleUpdate = (key: string, value: string) => {
        onChange({ ...envVars, [key]: value });
    };

    const handleDelete = (key: string) => {
        const { [key]: _, ...rest } = envVars;
        onChange(rest);
    };

    const isSecretKey = (key: string): boolean => {
        const secretPatterns = ['KEY', 'SECRET', 'PASSWORD', 'TOKEN', 'CREDENTIAL'];
        return secretPatterns.some(p => key.toUpperCase().includes(p));
    };

    const maskValue = (value: string): string => {
        if (value.length <= 6) return '••••••';
        return value.slice(0, 3) + '•••' + value.slice(-3);
    };

    // --- Raw Mode Logic ---

    const envToString = (vars: Record<string, string>) => {
        return Object.entries(vars)
            .map(([k, v]) => `${k}=${v}`)
            .join('\n');
    };

    const parseEnvString = (str: string): Record<string, string> => {
        const vars: Record<string, string> = {};
        str.split('\n').forEach(line => {
            const trimmed = line.trim();
            if (!trimmed || trimmed.startsWith('#')) return; // Skip empty lines and comments

            const eqIdx = trimmed.indexOf('=');
            if (eqIdx > 0) {
                const key = trimmed.slice(0, eqIdx).trim();
                let val = trimmed.slice(eqIdx + 1).trim();
                // Basic quote removal if wrapped in same quotes
                if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
                    val = val.slice(1, -1);
                }
                vars[key] = val;
            }
        });
        return vars;
    };

    const switchToRaw = () => {
        setRawText(envToString(envVars));
        setRawError(null);
        setIsRawMode(true);
    };

    const switchToList = () => {
        try {
            const parsed = parseEnvString(rawText);
            onChange(parsed);
            setIsRawMode(false);
            setRawError(null);
        } catch (err) {
            setRawError('Failed to parse environment variables');
        }
    };

    const handleRawChange = (text: string) => {
        setRawText(text);
        // Live update if possible, or wait for blur/switch? 
        // Let's live update to keep parent state in sync if we want 'Settings' modal to catch it on Save
        // But invalid state might be annoying. Let's just update local state and sync on valid parse?
        // Actually, for the parent 'Save' to work, we need to call onChange.
        // But if raw text is invalid (e.g. typing a key without value yet), it might drop data if we parse aggressively.
        // Better strategy: Only parse and sync when switching back OR when user stops typing?
        // Or just let the rawText stay local until 'switchToList' OR we can expose a way to get raw text.
        // However, the parent Modal asks for 'envVars' object on Save. 
        // So we MUST sync to parent.

        // Let's try to parse aggressively but forgive partials?
        // Or simply: When in raw mode, we update parent ONLY if it parses well?
        // Actually, common pattern: When in raw mode, updating parent state might be tricky if parent expects object.
        // If we want to support "Save" from modal while in Raw Mode, we should probably update parent on every change that parses correctly.
        const parsed = parseEnvString(text);
        onChange(parsed);
    };

    const copyToClipboard = () => {
        const text = isRawMode ? rawText : envToString(envVars);
        navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    return (
        <div className="h-full flex flex-col min-h-0 space-y-3">
            <div className="flex items-center justify-between pb-2 mb-2 border-b border-white/10 shrink-0">
                <div className="text-xs text-gray-400">
                    {isRawMode ? 'Edit variables as text' : 'Manage variables list'}
                </div>
                <div className="flex items-center gap-1">
                    <button
                        onClick={copyToClipboard}
                        className="p-1.5 hover:bg-white/10 rounded transition-colors text-gray-400 hover:text-white"
                        title="Copy all variables"
                    >
                        {copied ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
                    </button>
                    <div className="w-px h-4 bg-white/10 mx-1" />
                    <button
                        onClick={() => isRawMode ? switchToList() : switchToRaw()}
                        className="p-1.5 hover:bg-white/10 rounded transition-colors text-gray-400 hover:text-white flex items-center gap-2"
                        title={isRawMode ? "Switch to List View" : "Switch to Raw Text View"}
                    >
                        {isRawMode ? (
                            <List size={14} />
                        ) : (
                            <FileText size={14} />
                        )}
                        <span className="text-xs font-medium">{isRawMode ? 'List' : 'Raw'}</span>
                    </button>
                </div>
            </div>

            {isRawMode ? (
                <div className="flex-1 flex flex-col min-h-0 space-y-2">
                    <textarea
                        value={rawText}
                        onChange={(e) => handleRawChange(e.target.value)}
                        className="flex-1 w-full bg-black/40 border border-gray-700/50 rounded-lg p-3 font-mono text-xs text-green-400 focus:outline-none focus:border-hector-green resize-none leading-relaxed"
                        placeholder="KEY=VALUE"
                        spellCheck={false}
                    />
                    {rawError && (
                        <div className="text-xs text-red-400 px-1">
                            {rawError}
                        </div>
                    )}
                    <p className="text-xs text-gray-500">
                        Enter environment variables in <code>KEY=VALUE</code> format. One per line.
                    </p>
                </div>
            ) : (
                <>
                    {/* Existing variables */}
                    <div className="flex-1 overflow-y-auto pr-1 space-y-2 min-h-0">
                        {displayVars.length === 0 && (
                            <div className="text-center py-6 text-gray-500 text-sm">
                                No environment variables defined.
                            </div>
                        )}

                        {displayVars.map(({ key, value, isGlobal }) => (
                            <div
                                key={key}
                                className={cn(
                                    "flex items-center gap-2 rounded-lg border p-2",
                                    isGlobal
                                        ? "bg-blue-500/5 border-blue-500/20"
                                        : "bg-white/5 border-white/10"
                                )}
                            >
                                {/* Key */}
                                <div className="flex items-center gap-2 min-w-[140px]">
                                    {isGlobal && (
                                        <span title="Inherited from global">
                                            <Globe size={14} className="text-blue-400 shrink-0" />
                                        </span>
                                    )}
                                    <span className="font-mono text-sm text-gray-300 truncate" title={key}>
                                        {key}
                                    </span>
                                </div>

                                {/* Value */}
                                <div className="flex-1 flex items-center gap-2">
                                    {isGlobal ? (
                                        <span className="flex-1 font-mono text-sm text-gray-500 truncate">
                                            {isSecretKey(key) ? maskValue(value) : value}
                                        </span>
                                    ) : (
                                        <input
                                            type={isSecretKey(key) && !revealedKeys.has(key) ? 'password' : 'text'}
                                            value={value}
                                            onChange={(e) => handleUpdate(key, e.target.value)}
                                            className="flex-1 px-2 py-1 bg-transparent border-none text-sm text-white font-mono focus:outline-none focus:ring-1 focus:ring-hector-green/50 rounded"
                                        />
                                    )}
                                </div>

                                {/* Actions */}
                                <div className="flex items-center gap-1">
                                    {isSecretKey(key) && !isGlobal && (
                                        <button
                                            onClick={() => toggleReveal(key)}
                                            className="p-1 hover:bg-white/10 rounded transition-colors"
                                            title={revealedKeys.has(key) ? 'Hide value' : 'Show value'}
                                        >
                                            {revealedKeys.has(key) ? (
                                                <EyeOff size={14} className="text-gray-400" />
                                            ) : (
                                                <Eye size={14} className="text-gray-400" />
                                            )}
                                        </button>
                                    )}

                                    {!isGlobal && (
                                        <button
                                            onClick={() => handleDelete(key)}
                                            className="p-1 hover:bg-red-500/20 rounded transition-colors"
                                            title="Delete"
                                        >
                                            <Trash2 size={14} className="text-red-400" />
                                        </button>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>

                    {/* Add new variable */}
                    <div className="flex items-center gap-2 pt-2 border-t border-white/10 shrink-0">
                        <input
                            type="text"
                            value={newKey}
                            onChange={(e) => setNewKey(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, '_'))}
                            placeholder="VARIABLE_NAME"
                            className="w-40 px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white font-mono placeholder:text-gray-500 focus:outline-none focus:border-hector-green"
                        />
                        <input
                            type="text"
                            value={newValue}
                            onChange={(e) => setNewValue(e.target.value)}
                            placeholder="value"
                            className="flex-1 px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white font-mono placeholder:text-gray-500 focus:outline-none focus:border-hector-green"
                            onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
                        />
                        <button
                            onClick={handleAdd}
                            disabled={!newKey.trim()}
                            className={cn(
                                "p-2 rounded-lg transition-colors",
                                newKey.trim()
                                    ? "bg-hector-green hover:bg-hector-green/80 text-white"
                                    : "bg-gray-700 text-gray-500 cursor-not-allowed"
                            )}
                            title="Add variable"
                        >
                            <Plus size={18} />
                        </button>
                    </div>
                </>
            )}

            {/* Help text */}
            {!isRawMode && showInherited && Object.keys(globalEnvVars).length > 0 && (
                <p className="text-xs text-gray-500 flex items-center gap-1">
                    <Globe size={12} className="text-blue-400" />
                    Variables with globe icon are inherited from global settings
                </p>
            )}
        </div>
    );
};

