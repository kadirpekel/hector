import { useRef, memo } from 'react';
import Editor from '@monaco-editor/react';
import "../../lib/monaco"; // Initialize Monaco configuration

interface ConfigEditorProps {
    initialValue: string;
    theme: string;
    onChange: (value: string | undefined) => void;
    onMount?: (editor: any, monaco: any) => void;
}

export const ConfigEditor = memo(({ initialValue, theme, onChange, onMount }: ConfigEditorProps) => {
    // Use a ref to track if this is the first mount to handle initial value
    const isFirstMount = useRef(true);

    const handleEditorMount = (editor: any, monaco: any) => {
        if (onMount) onMount(editor, monaco);
        // Explicitly set initial value on mount to ensure synchronization
        if (isFirstMount.current) {
            editor.setValue(initialValue);
            isFirstMount.current = false;
        }
    };

    return (
        <Editor
            height="100%"
            language="yaml"
            // IMPORTANT: We do NOT pass 'value' to avoid controlled component behavior
            // causing re-renders/cursor jumps. We rely on internal state + onChange.
            defaultValue={initialValue}
            onChange={onChange}
            theme={theme}
            onMount={handleEditorMount}
            options={{
                minimap: { enabled: false },
                fontSize: 14,
                lineNumbers: 'on',
                wordWrap: 'on',
                tabSize: 2,
                insertSpaces: true,
                detectIndentation: false, // Force 2 spaces
                scrollBeyondLastLine: false,
                automaticLayout: true,
                contextmenu: true,
                quickSuggestions: true,
                suggest: {
                    showWords: true,
                    showSnippets: true,
                },
                suggestOnTriggerCharacters: true,
                acceptSuggestionOnEnter: 'smart',
                tabCompletion: 'on',
                wordBasedSuggestions: 'currentDocument',
                hover: { enabled: true },
                parameterHints: { enabled: true },
                scrollbar: {
                    vertical: 'auto', // Or visible/hidden/auto
                    horizontal: 'auto',
                    verticalScrollbarSize: 8, // Smaller scrollbar
                    horizontalScrollbarSize: 8,
                    useShadows: false,
                },
            }}
        />
    );
}, (prevProps, nextProps) => {
    // Custom comparison to Prevent Re-renders
    // Only re-render if theme changes. 
    // We explicitly IGNORE initialValue changes because that's only for first mount.
    // We ignore onChange function identity changes assuming it's stable or we don't care.
    return prevProps.theme === nextProps.theme;
});
