import React, { useState, useEffect } from 'react';
import { ResourceModal, FormField, TextInput, SelectInput, ToggleInput } from './ResourceModal';
import type { GuardrailConfig } from '../../lib/config-utils';

interface GuardrailModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (id: string, config: GuardrailConfig) => void;
    existingIds: string[];
    editId?: string | null;
    editConfig?: GuardrailConfig | null;
}

export const GuardrailModal: React.FC<GuardrailModalProps> = ({
    isOpen,
    onClose,
    onSave,
    existingIds,
    editId,
    editConfig,
}) => {
    const isEditing = !!editId;

    const [id, setId] = useState('');
    const [enabled, setEnabled] = useState(true);

    // Input validations
    const [inputLength, setInputLength] = useState(false);
    const [inputMinLength, setInputMinLength] = useState('1');
    const [inputMaxLength, setInputMaxLength] = useState('10000');
    const [inputInjection, setInputInjection] = useState(false);
    const [inputSanitizer, setInputSanitizer] = useState(false);
    const [inputPattern, setInputPattern] = useState(false);
    const [inputAllowPatterns, setInputAllowPatterns] = useState('');
    const [inputBlockPatterns, setInputBlockPatterns] = useState('');

    // Output validations
    const [outputPII, setOutputPII] = useState(false);
    const [detectEmail, setDetectEmail] = useState(true);
    const [detectPhone, setDetectPhone] = useState(true);
    const [detectSSN, setDetectSSN] = useState(true);
    const [detectCreditCard, setDetectCreditCard] = useState(true);
    const [redactMode, setRedactMode] = useState('mask');
    const [outputContent, setOutputContent] = useState(false);
    const [blockedKeywords, setBlockedKeywords] = useState('');
    const [blockedPatterns, setBlockedPatterns] = useState('');

    // Tool Authorization
    const [toolAuth, setToolAuth] = useState(false);
    const [allowedTools, setAllowedTools] = useState('');
    const [blockedTools, setBlockedTools] = useState('');

    // Moderation
    const [moderationEnabled, setModerationEnabled] = useState(false);
    const [moderationStrategy, setModerationStrategy] = useState('openai');
    const [openaiModel, setOpenaiModel] = useState('omni-moderation-latest');
    const [openaiThreshold, setOpenaiThreshold] = useState('0.8');
    const [lakeraProjectId, setLakeraProjectId] = useState('');
    const [promptLLM, setPromptLLM] = useState('');
    const [promptTemplate, setPromptTemplate] = useState('');
    const [promptSafeField, setPromptSafeField] = useState('safe');

    // Actions
    const [inputLengthAction, setInputLengthAction] = useState('block');
    const [inputInjectionAction, setInputInjectionAction] = useState('block');
    const [inputPatternAction, setInputPatternAction] = useState('block');
    const [outputPIIAction, setOutputPIIAction] = useState('block');
    const [outputContentAction, setOutputContentAction] = useState('block');
    const [toolAuthAction, setToolAuthAction] = useState('block');
    const [moderationAction, setModerationAction] = useState('block');

    // Lakera additional
    const [lakeraBreakdown, setLakeraBreakdown] = useState(false);
    const [lakeraEndpoint, setLakeraEndpoint] = useState('');


    useEffect(() => {
        if (isOpen) {
            if (isEditing && editConfig) {
                setId(editId);
                setEnabled(editConfig.enabled !== false);

                // Input
                setInputLength(editConfig.input?.length?.enabled || false);
                setInputMinLength(editConfig.input?.length?.min_length?.toString() || '1');
                setInputMaxLength(editConfig.input?.length?.max_length?.toString() || '10000');
                setInputInjection(editConfig.input?.injection?.enabled || false);
                setInputSanitizer(editConfig.input?.sanitizer?.enabled || false);
                setInputPattern(editConfig.input?.pattern?.enabled || false);
                setInputAllowPatterns(editConfig.input?.pattern?.allow_patterns?.join(', ') || '');
                setInputBlockPatterns(editConfig.input?.pattern?.block_patterns?.join(', ') || '');

                // Output
                setOutputPII(editConfig.output?.pii?.enabled || false);
                setDetectEmail(editConfig.output?.pii?.detect_email !== false);
                setDetectPhone(editConfig.output?.pii?.detect_phone !== false);
                setDetectSSN(editConfig.output?.pii?.detect_ssn !== false);
                setDetectCreditCard(editConfig.output?.pii?.detect_credit_card !== false);
                setRedactMode(editConfig.output?.pii?.redact_mode || 'mask');
                setOutputContent(editConfig.output?.content?.enabled || false);
                setBlockedKeywords(editConfig.output?.content?.blocked_keywords?.join(', ') || '');
                setBlockedPatterns(editConfig.output?.content?.blocked_patterns?.join(', ') || '');

                // Tool
                // Handle both old and new structure if possible
                const toolConfig = editConfig.tool as any;
                const toolAuthConfig = toolConfig?.authorization || (toolConfig?.enabled ? toolConfig : undefined);
                setToolAuth(toolAuthConfig?.enabled || false);
                setAllowedTools(toolAuthConfig?.allowed_tools?.join(', ') || '');
                setBlockedTools(toolAuthConfig?.blocked_tools?.join(', ') || '');

                // Moderation
                setModerationEnabled(editConfig.moderation?.enabled || false);
                setModerationStrategy(editConfig.moderation?.strategy || 'openai');
                setOpenaiModel(editConfig.moderation?.openai?.model || 'omni-moderation-latest');
                setOpenaiThreshold(editConfig.moderation?.openai?.threshold?.toString() || '0.8');
                setLakeraProjectId(editConfig.moderation?.lakera?.project_id || '');
                setLakeraBreakdown(editConfig.moderation?.lakera?.breakdown || false);
                setLakeraEndpoint(editConfig.moderation?.lakera?.endpoint || '');
                setPromptLLM(editConfig.moderation?.prompt?.llm || '');
                setPromptTemplate(editConfig.moderation?.prompt?.template || '');
                setPromptSafeField(editConfig.moderation?.prompt?.safe_field || 'safe');

                // Actions
                setInputLengthAction(editConfig.input?.length?.action || 'block');
                setInputInjectionAction(editConfig.input?.injection?.action || 'block');
                setInputPatternAction(editConfig.input?.pattern?.action || 'block');
                setOutputPIIAction(editConfig.output?.pii?.action || 'block');
                setOutputContentAction(editConfig.output?.content?.action || 'block');
                setToolAuthAction(toolAuthConfig?.action || 'block');
                setModerationAction(editConfig.moderation?.action || 'block');

            } else {
                // Defaults
                setId('');
                setEnabled(true);
                setInputLength(false); setInputMinLength('1'); setInputMaxLength('10000');
                setInputInjection(false);
                setInputSanitizer(false);
                setInputPattern(false); setInputAllowPatterns(''); setInputBlockPatterns('');
                setOutputPII(false); setDetectEmail(true); setDetectPhone(true); setDetectSSN(true); setDetectCreditCard(true); setRedactMode('mask');
                setOutputContent(false); setBlockedKeywords(''); setBlockedPatterns('');
                setToolAuth(false); setAllowedTools(''); setBlockedTools('');
                setModerationEnabled(false); setModerationStrategy('openai'); setOpenaiModel('omni-moderation-latest'); setOpenaiThreshold('0.8');
                setLakeraProjectId(''); setPromptLLM(''); setPromptTemplate('');
            }
        }
    }, [isOpen, isEditing, editId, editConfig]);

    const handleSave = () => {
        const config: GuardrailConfig = {
            enabled,
        };

        // Input
        if (inputLength || inputInjection || inputSanitizer || inputPattern) {
            config.input = { chain_mode: 'fail_fast' };
            if (inputLength) {
                config.input.length = {
                    enabled: true,
                    min_length: parseInt(inputMinLength) || 1,
                    max_length: parseInt(inputMaxLength) || 10000,
                    action: inputLengthAction as any,
                };
            }
            if (inputInjection) config.input.injection = { enabled: true, action: inputInjectionAction as any };
            if (inputSanitizer) config.input.sanitizer = { enabled: true, trim_whitespace: true };
            if (inputPattern) {
                config.input.pattern = {
                    enabled: true,
                    allow_patterns: inputAllowPatterns ? inputAllowPatterns.split(',').map(s => s.trim()).filter(Boolean) : undefined,
                    block_patterns: inputBlockPatterns ? inputBlockPatterns.split(',').map(s => s.trim()).filter(Boolean) : undefined,
                    action: inputPatternAction as any,
                };
            }
        }

        // Output
        if (outputPII || outputContent) {
            config.output = { chain_mode: 'fail_fast' };
            if (outputPII) {
                config.output.pii = {
                    enabled: true,
                    detect_email: detectEmail,
                    detect_phone: detectPhone,
                    detect_ssn: detectSSN,
                    detect_credit_card: detectCreditCard,
                    redact_mode: redactMode as 'mask' | 'remove' | 'hash',
                    action: outputPIIAction as any,
                };
            }
            if (outputContent) {
                config.output.content = {
                    enabled: true,
                    blocked_keywords: blockedKeywords ? blockedKeywords.split(',').map(s => s.trim()).filter(Boolean) : undefined,
                    blocked_patterns: blockedPatterns ? blockedPatterns.split(',').map(s => s.trim()).filter(Boolean) : undefined,
                    action: outputContentAction as any,
                };
            }
        }

        // Tool
        if (toolAuth) {
            config.tool = {
                chain_mode: 'fail_fast',
                authorization: {
                    enabled: true,
                    allowed_tools: allowedTools ? allowedTools.split(',').map(s => s.trim()).filter(Boolean) : undefined,
                    blocked_tools: blockedTools ? blockedTools.split(',').map(s => s.trim()).filter(Boolean) : undefined,
                    action: toolAuthAction as any,
                }
            };
        }

        // Moderation
        if (moderationEnabled) {
            config.moderation = {
                enabled: true,
                strategy: moderationStrategy as any,
                action: moderationAction as any,
            };
            if (moderationStrategy === 'openai') {
                config.moderation.openai = {
                    model: openaiModel,
                    threshold: parseFloat(openaiThreshold) || 0.8,
                };
            } else if (moderationStrategy === 'lakera') {
                config.moderation.lakera = {
                    project_id: lakeraProjectId,
                    breakdown: lakeraBreakdown || undefined,
                    endpoint: lakeraEndpoint || undefined,
                };
            } else if (moderationStrategy === 'prompt') {
                config.moderation.prompt = {
                    llm: promptLLM,
                    template: promptTemplate,
                    safe_field: promptSafeField,
                };
            }
        }

        onSave(id, config);
        onClose();
    };

    const isValid = id && !(!isEditing && existingIds.includes(id));

    return (
        <ResourceModal
            isOpen={isOpen}
            onClose={onClose}
            title={isEditing ? `Edit Guardrail: ${editId}` : 'Add Guardrail'}
            onSave={handleSave}
            saveDisabled={!isValid}
        >
            <FormField label="ID" required hint="Unique identifier for this guardrail">
                <TextInput
                    value={id}
                    onChange={setId}
                    placeholder="e.g., default, strict"
                />
            </FormField>

            <ToggleInput
                checked={enabled}
                onChange={setEnabled}
                label="Enabled"
            />

            {/* Input Validations */}
            <div className="pt-4 border-t border-white/10">
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-3">
                    Input Validations
                </div>

                <div className="space-y-3">
                    <div className="flex items-center justify-between">
                        <ToggleInput
                            checked={inputLength}
                            onChange={setInputLength}
                            label="Length check"
                        />
                        {inputLength && (
                            <div className="w-32">
                                <SelectInput
                                    value={inputLengthAction}
                                    onChange={setInputLengthAction}
                                    options={[{ value: 'allow', label: 'Log' }, { value: 'block', label: 'Block' }, { value: 'warn', label: 'Warn' }]}
                                    placeholder="Action"
                                />
                            </div>
                        )}
                    </div>
                    {inputLength && (
                        <div className="grid grid-cols-2 gap-2 pl-4">
                            <FormField label="Min Length">
                                <TextInput
                                    value={inputMinLength}
                                    onChange={setInputMinLength}
                                    type="number"
                                />
                            </FormField>
                            <FormField label="Max Length">
                                <TextInput
                                    value={inputMaxLength}
                                    onChange={setInputMaxLength}
                                    type="number"
                                />
                            </FormField>
                        </div>
                    )}

                    <div className="flex items-center justify-between">
                        <ToggleInput
                            checked={inputInjection}
                            onChange={setInputInjection}
                            label="Injection detection"
                        />
                        {inputInjection && (
                            <div className="w-32">
                                <SelectInput value={inputInjectionAction} onChange={setInputInjectionAction} options={[{ value: 'allow', label: 'Log' }, { value: 'block', label: 'Block' }, { value: 'warn', label: 'Warn' }]} />
                            </div>
                        )}
                    </div>

                    <ToggleInput
                        checked={inputSanitizer}
                        onChange={setInputSanitizer}
                        label="Sanitize input"
                    />

                    <div className="flex items-center justify-between">
                        <ToggleInput
                            checked={inputPattern}
                            onChange={setInputPattern}
                            label="Pattern validation"
                        />
                        {inputPattern && (
                            <div className="w-32">
                                <SelectInput value={inputPatternAction} onChange={setInputPatternAction} options={[{ value: 'allow', label: 'Log' }, { value: 'block', label: 'Block' }, { value: 'warn', label: 'Warn' }]} />
                            </div>
                        )}
                    </div>
                    {inputPattern && (
                        <div className="space-y-2 pl-4">
                            <FormField label="Allow Patterns" hint="Regex (comma-separated)">
                                <TextInput value={inputAllowPatterns} onChange={setInputAllowPatterns} />
                            </FormField>
                            <FormField label="Block Patterns" hint="Regex (comma-separated)">
                                <TextInput value={inputBlockPatterns} onChange={setInputBlockPatterns} />
                            </FormField>
                        </div>
                    )}
                </div>
            </div>

            {/* Output Validations */}
            <div className="pt-4 border-t border-white/10">
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-3">
                    Output Validations
                </div>

                <div className="space-y-3">
                    <div className="flex items-center justify-between">
                        <ToggleInput
                            checked={outputPII}
                            onChange={setOutputPII}
                            label="PII detection"
                        />
                        {outputPII && (
                            <div className="w-32">
                                <SelectInput value={outputPIIAction} onChange={setOutputPIIAction} options={[{ value: 'allow', label: 'Log' }, { value: 'block', label: 'Block' }, { value: 'warn', label: 'Warn' }, { value: 'modify', label: 'Modify' }]} />
                            </div>
                        )}
                    </div>

                    {outputPII && (
                        <div className="space-y-2 pl-4">
                            <div className="grid grid-cols-2 gap-2">
                                <ToggleInput checked={detectEmail} onChange={setDetectEmail} label="Email" />
                                <ToggleInput checked={detectPhone} onChange={setDetectPhone} label="Phone" />
                                <ToggleInput checked={detectSSN} onChange={setDetectSSN} label="SSN" />
                                <ToggleInput checked={detectCreditCard} onChange={setDetectCreditCard} label="Credit Card" />
                            </div>
                            <FormField label="Redact Mode">
                                <SelectInput
                                    value={redactMode}
                                    onChange={setRedactMode}
                                    options={[
                                        { value: 'mask', label: 'Mask (****)' },
                                        { value: 'remove', label: 'Remove' },
                                        { value: 'hash', label: 'Hash' },
                                    ]}
                                />
                            </FormField>
                        </div>
                    )}

                    <div className="flex items-center justify-between">
                        <ToggleInput
                            checked={outputContent}
                            onChange={setOutputContent}
                            label="Content filtering"
                        />
                        {outputContent && (
                            <div className="w-32">
                                <SelectInput value={outputContentAction} onChange={setOutputContentAction} options={[{ value: 'allow', label: 'Log' }, { value: 'block', label: 'Block' }, { value: 'warn', label: 'Warn' }]} />
                            </div>
                        )}
                    </div>

                    {outputContent && (
                        <div className="space-y-2 pl-4">
                            <FormField label="Blocked Keywords" hint="Comma-separated">
                                <TextInput
                                    value={blockedKeywords}
                                    onChange={setBlockedKeywords}
                                    placeholder="password, secret"
                                />
                            </FormField>
                            <FormField label="Blocked Patterns" hint="Regex patterns (comma-separated)">
                                <TextInput
                                    value={blockedPatterns}
                                    onChange={setBlockedPatterns}
                                    placeholder="sk-[a-zA-Z0-9]+"
                                />
                            </FormField>
                        </div>
                    )}
                </div>
            </div>

            {/* Tool Authorization */}
            <div className="pt-4 border-t border-white/10">
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-3">
                    Tool Authorization
                </div>

                <div className="space-y-3">
                    <div className="flex items-center justify-between">
                        <ToggleInput
                            checked={toolAuth}
                            onChange={setToolAuth}
                            label="Enable tool authorization"
                        />
                        {toolAuth && (
                            <div className="w-32">
                                <SelectInput value={toolAuthAction} onChange={setToolAuthAction} options={[{ value: 'allow', label: 'Log' }, { value: 'block', label: 'Block' }, { value: 'warn', label: 'Warn' }]} />
                            </div>
                        )}
                    </div>

                    {toolAuth && (
                        <div className="space-y-2 pl-4">
                            <FormField label="Allowed Tools" hint="Whitelist (comma-separated)">
                                <TextInput
                                    value={allowedTools}
                                    onChange={setAllowedTools}
                                    placeholder="calculator, search"
                                />
                            </FormField>
                            <FormField label="Blocked Tools" hint="Blacklist (comma-separated)">
                                <TextInput
                                    value={blockedTools}
                                    onChange={setBlockedTools}
                                    placeholder="delete_file"
                                />
                            </FormField>
                        </div>
                    )}
                </div>
            </div>

            {/* Moderation */}
            <div className="pt-4 border-t border-white/10">
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-3">
                    LLM Moderation
                </div>

                <div className="space-y-3">
                    <div className="flex items-center justify-between">
                        <ToggleInput
                            checked={moderationEnabled}
                            onChange={setModerationEnabled}
                            label="Enable Moderation"
                        />
                        {moderationEnabled && (
                            <div className="w-32">
                                <SelectInput value={moderationAction} onChange={setModerationAction} options={[{ value: 'block', label: 'Block' }, { value: 'warn', label: 'Warn' }]} />
                            </div>
                        )}
                    </div>

                    {moderationEnabled && (
                        <div className="space-y-3 pl-4">
                            <FormField label="Strategy">
                                <SelectInput
                                    value={moderationStrategy}
                                    onChange={setModerationStrategy}
                                    options={[
                                        { value: 'openai', label: 'OpenAI Moderation' },
                                        { value: 'lakera', label: 'Lakera Guard' },
                                        { value: 'prompt', label: 'Custom Prompt' },
                                    ]}
                                />
                            </FormField>

                            {moderationStrategy === 'openai' && (
                                <>
                                    <FormField label="Model">
                                        <TextInput value={openaiModel} onChange={setOpenaiModel} placeholder="omni-moderation-latest" />
                                    </FormField>
                                    <FormField label="Threshold (0-1)">
                                        <TextInput value={openaiThreshold} onChange={setOpenaiThreshold} type="number" step="0.1" />
                                    </FormField>
                                </>
                            )}

                            {moderationStrategy === 'lakera' && (
                                <>
                                    <FormField label="Project ID">
                                        <TextInput value={lakeraProjectId} onChange={setLakeraProjectId} placeholder="Lakera Project ID" />
                                    </FormField>
                                    <div className="grid grid-cols-2 gap-2">
                                        <FormField label="Endpoint (Optional)">
                                            <TextInput value={lakeraEndpoint} onChange={setLakeraEndpoint} placeholder="Custom Endpoint" />
                                        </FormField>
                                        <div className="pt-6">
                                            <ToggleInput checked={lakeraBreakdown} onChange={setLakeraBreakdown} label="Breakdown" />
                                        </div>
                                    </div>
                                </>
                            )}

                            {moderationStrategy === 'prompt' && (
                                <>
                                    <div className="grid grid-cols-2 gap-2">
                                        <FormField label="LLM">
                                            <TextInput value={promptLLM} onChange={setPromptLLM} placeholder="LLM ID" />
                                        </FormField>
                                        <FormField label="Safe Field">
                                            <TextInput value={promptSafeField} onChange={setPromptSafeField} placeholder="safe" />
                                        </FormField>
                                    </div>
                                    <FormField label="Template">
                                        <div className="relative">
                                            <textarea
                                                className="w-full bg-black/20 border border-white/10 rounded px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-blue-500 min-h-[100px]"
                                                value={promptTemplate}
                                                onChange={(e) => setPromptTemplate(e.target.value)}
                                                placeholder="Check if specific input {input} is safe..."
                                            />
                                        </div>
                                    </FormField>
                                </>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </ResourceModal>
    );
};
