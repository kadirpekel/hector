import React from 'react';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';

interface ResourceModalProps {
    isOpen: boolean;
    onClose: () => void;
    title: string;
    children: React.ReactNode;
    onSave: () => void;
    saveDisabled?: boolean;
    saveLabel?: string;
}

export const ResourceModal: React.FC<ResourceModalProps> = ({
    isOpen,
    onClose,
    title,
    children,
    onSave,
    saveDisabled = false,
    saveLabel = 'Save',
}) => {
    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div
                className="bg-hector-darker border border-white/10 rounded-xl shadow-2xl w-full max-w-lg mx-4 max-h-[90vh] flex flex-col"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
                    <h2 className="text-lg font-semibold text-white">{title}</h2>
                    <button
                        onClick={onClose}
                        className="p-1 hover:bg-white/10 rounded transition-colors"
                    >
                        <X size={20} className="text-gray-400" />
                    </button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4 min-h-[400px]">
                    {children}
                </div>

                {/* Footer */}
                <div className="flex justify-end gap-3 px-6 py-4 border-t border-white/10">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={onSave}
                        disabled={saveDisabled}
                        className={cn(
                            "px-4 py-2 text-sm font-medium rounded-lg transition-colors",
                            saveDisabled
                                ? "bg-gray-600 text-gray-400 cursor-not-allowed"
                                : "bg-hector-green hover:bg-hector-green/80 text-white"
                        )}
                    >
                        {saveLabel}
                    </button>
                </div>
            </div>
        </div>
    );
};

// Reusable form field components
interface FormFieldProps {
    label: string;
    required?: boolean;
    hint?: string;
    children: React.ReactNode;
}

export const FormField: React.FC<FormFieldProps> = ({ label, required, hint, children }) => (
    <div>
        <label className="block text-sm font-medium text-gray-300 mb-1.5">
            {label}{required && <span className="text-red-400 ml-1">*</span>}
        </label>
        {children}
        {hint && <p className="mt-1 text-xs text-gray-500">{hint}</p>}
    </div>
);

interface TextInputProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    type?: 'text' | 'password' | 'number';
    className?: string;
    step?: string;
    min?: string;
    max?: string;
    pattern?: string;
}

export const TextInput: React.FC<TextInputProps> = ({
    value,
    onChange,
    placeholder,
    type = 'text',
    className,
    step,
    min,
    max,
    pattern,
}) => (
    <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        step={step}
        min={min}
        max={max}
        pattern={pattern}
        className={cn(
            "w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white",
            "placeholder:text-gray-500 focus:outline-none focus:border-hector-green transition-colors",
            className
        )}
    />
);

interface SelectInputProps {
    value: string;
    onChange: (value: string) => void;
    options: Array<{ value: string; label: string }>;
    placeholder?: string;
}

export const SelectInput: React.FC<SelectInputProps> = ({
    value,
    onChange,
    options,
    placeholder,
}) => (
    <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white focus:outline-none focus:border-hector-green transition-colors"
    >
        {placeholder && <option value="">{placeholder}</option>}
        {options.map((opt) => (
            <option key={opt.value} value={opt.value}>
                {opt.label}
            </option>
        ))}
    </select>
);

interface TextAreaInputProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    rows?: number;
}

export const TextAreaInput: React.FC<TextAreaInputProps> = ({
    value,
    onChange,
    placeholder,
    rows = 3,
}) => (
    <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        rows={rows}
        className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-gray-500 focus:outline-none focus:border-hector-green transition-colors resize-none"
    />
);

// Combo input: text field with dropdown suggestions
interface ComboInputProps {
    value: string;
    onChange: (value: string) => void;
    options: Array<{ value: string; label: string }>;
    placeholder?: string;
}

export const ComboInput: React.FC<ComboInputProps> = ({
    value,
    onChange,
    options,
    placeholder,
}) => {
    const [isOpen, setIsOpen] = React.useState(false);
    const [inputValue, setInputValue] = React.useState(value);
    const containerRef = React.useRef<HTMLDivElement>(null);

    React.useEffect(() => {
        setInputValue(value);
    }, [value]);

    React.useEffect(() => {
        const handleClickOutside = (e: MouseEvent) => {
            if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
                setIsOpen(false);
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const filteredOptions = options.filter(opt =>
        opt.label.toLowerCase().includes(inputValue.toLowerCase()) ||
        opt.value.toLowerCase().includes(inputValue.toLowerCase())
    );

    return (
        <div ref={containerRef} className="relative">
            <input
                type="text"
                value={inputValue}
                onChange={(e) => {
                    setInputValue(e.target.value);
                    onChange(e.target.value);
                    setIsOpen(true);
                }}
                onFocus={() => setIsOpen(true)}
                placeholder={placeholder}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm text-white placeholder:text-gray-500 focus:outline-none focus:border-hector-green transition-colors"
            />
            {isOpen && filteredOptions.length > 0 && (
                <div className="absolute z-50 w-full mt-1 bg-hector-darker border border-white/10 rounded-lg shadow-xl max-h-48 overflow-y-auto">
                    {filteredOptions.map((opt) => (
                        <button
                            key={opt.value}
                            type="button"
                            onClick={() => {
                                setInputValue(opt.value);
                                onChange(opt.value);
                                setIsOpen(false);
                            }}
                            className={cn(
                                "w-full px-3 py-2 text-left text-sm hover:bg-white/10 transition-colors",
                                opt.value === value ? "text-hector-green bg-white/5" : "text-gray-300"
                            )}
                        >
                            <span className="font-medium">{opt.label}</span>
                            {opt.label !== opt.value && (
                                <span className="ml-2 text-xs text-gray-500">{opt.value}</span>
                            )}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
};

interface ToggleInputProps {
    checked: boolean;
    onChange: (checked: boolean) => void;
    label: string;
}

export const ToggleInput: React.FC<ToggleInputProps> = ({ checked, onChange, label }) => (
    <label className="flex items-center gap-3 cursor-pointer">
        <div
            className={cn(
                "w-10 h-5 rounded-full transition-colors relative",
                checked ? "bg-hector-green" : "bg-white/10"
            )}
            onClick={() => onChange(!checked)}
        >
            <div
                className={cn(
                    "absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform",
                    checked ? "translate-x-5" : "translate-x-0.5"
                )}
            />
        </div>
        <span className="text-sm text-gray-300">{label}</span>
    </label>
);
