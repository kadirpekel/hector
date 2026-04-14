/**
 * CloudProgressModal
 *
 * Shows a step-by-step progress tracker during cloud provisioning.
 * Automatically appears when cloudStore.steps is non-empty,
 * and dismisses on completion or user action.
 */

import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { Cloud, CheckCircle2, Circle, Loader2, XCircle, X } from 'lucide-react';
import { useCloudStore, type CloudStep } from '../store/cloudStore';
import { useCloudAuthStore } from '../store/cloudAuthStore';
import { cn } from '../lib/utils';

export function CloudProgressModal() {
    const steps = useCloudStore((s) => s.steps);
    const status = useCloudStore((s) => s.status);
    const error = useCloudStore((s) => s.error);
    const cloudConnect = useCloudStore((s) => s.connect);
    const cloudLogout = useCloudAuthStore((s) => s.logout);

    const dismissProgress = useCloudStore((s) => s.dismissProgress);

    const handleRetry = () => {
        dismissProgress();
        cloudConnect();
    };

    const handleForgetCredentials = () => {
        dismissProgress();
        cloudLogout();
    };

    // Auto-dismiss after 2s on success
    const [autoDismiss, setAutoDismiss] = useState(false);
    useEffect(() => {
        if (status === 'connected' && steps.length > 0) {
            const timer = setTimeout(() => {
                setAutoDismiss(true);
                dismissProgress();
            }, 2000);
            return () => clearTimeout(timer);
        }
        setAutoDismiss(false);
    }, [status, steps.length, dismissProgress]);

    if (steps.length === 0 || autoDismiss) return null;

    const isWorking = status === 'working';
    const isDone = status === 'connected';
    const isError = status === 'error';

    return createPortal(
        <>
            {/* Backdrop */}
            <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50" />

            {/* Modal */}
            <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
                <div className="bg-gray-900 border border-white/10 rounded-2xl shadow-2xl w-full max-w-sm overflow-hidden">
                    {/* Header */}
                    <div className="flex items-center justify-between p-4 border-b border-white/10">
                        <div className="flex items-center gap-3">
                            <div className={cn(
                                "p-2 rounded-lg",
                                isDone ? "bg-green-500/20" : isError ? "bg-red-500/20" : "bg-blue-500/20"
                            )}>
                                {isDone ? (
                                    <CheckCircle2 className="text-green-500" size={20} />
                                ) : isError ? (
                                    <XCircle className="text-red-500" size={20} />
                                ) : (
                                    <Cloud className="text-blue-400" size={20} />
                                )}
                            </div>
                            <div>
                                <h2 className="text-base font-semibold text-white">
                                    {isDone ? 'Connected!' : isError ? 'Connection Failed' : 'Connecting to Cloud'}
                                </h2>
                                <p className="text-xs text-gray-400">
                                    {isDone
                                        ? 'Your cloud instance is ready'
                                        : isError
                                            ? 'Something went wrong'
                                            : 'Setting up your Hector Cloud instance...'}
                                </p>
                            </div>
                        </div>
                        {!isWorking && (
                            <button
                                onClick={dismissProgress}
                                className="p-1 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                            >
                                <X size={18} />
                            </button>
                        )}
                    </div>

                    {/* Steps */}
                    <div className="p-4 space-y-3">
                        {steps.map((step, i) => (
                            <StepRow key={step.id} step={step} isLast={i === steps.length - 1} />
                        ))}
                    </div>

                    {/* Error detail */}
                    {isError && error && (
                        <div className="px-4 pb-4">
                            <div className="text-xs text-red-400 bg-red-500/10 p-3 rounded-lg break-words">
                                {error}
                            </div>
                        </div>
                    )}

                    {/* Footer */}
                    {isDone && (
                        <div className="px-4 pb-4">
                            <button
                                onClick={dismissProgress}
                                className="w-full py-2 rounded-lg text-sm font-medium transition-colors bg-green-500 hover:bg-green-500/80 text-white"
                            >
                                Start Using Hector Cloud
                            </button>
                        </div>
                    )}
                    {isError && (
                        <div className="px-4 pb-4 space-y-2">
                            <button
                                onClick={handleRetry}
                                className="w-full py-2 rounded-lg text-sm font-medium transition-colors bg-blue-600 hover:bg-blue-500 text-white"
                            >
                                Retry
                            </button>
                            <button
                                onClick={handleForgetCredentials}
                                className="w-full py-2 rounded-lg text-sm font-medium transition-colors bg-gray-700 hover:bg-gray-600 text-white"
                            >
                                Forget Credentials
                            </button>
                        </div>
                    )}
                </div>
            </div>
        </>,
        document.body
    );
}

function StepRow({ step, isLast }: { step: CloudStep; isLast: boolean }) {
    return (
        <div className="flex items-start gap-3">
            {/* Icon + connector line */}
            <div className="flex flex-col items-center">
                <StepIcon status={step.status} />
                {!isLast && (
                    <div className={cn(
                        "w-px h-4 mt-1",
                        step.status === 'done' ? "bg-green-500/40" : "bg-white/10"
                    )} />
                )}
            </div>

            {/* Label + detail */}
            <div className="flex-1 min-w-0 -mt-0.5">
                <div className={cn(
                    "text-sm font-medium",
                    step.status === 'done' ? "text-green-400" :
                    step.status === 'active' ? "text-white" :
                    step.status === 'error' ? "text-red-400" :
                    "text-gray-500"
                )}>
                    {step.label}
                </div>
                {step.detail && (
                    <div className={cn(
                        "text-xs mt-0.5 truncate",
                        step.status === 'error' ? "text-red-400/80" : "text-gray-500"
                    )}>
                        {step.detail}
                    </div>
                )}
            </div>
        </div>
    );
}

function StepIcon({ status }: { status: CloudStep['status'] }) {
    switch (status) {
        case 'done':
            return <CheckCircle2 size={16} className="text-green-500" />;
        case 'active':
            return <Loader2 size={16} className="text-blue-400 animate-spin" />;
        case 'error':
            return <XCircle size={16} className="text-red-500" />;
        default:
            return <Circle size={16} className="text-gray-600" />;
    }
}
