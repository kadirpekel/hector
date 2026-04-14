import React from "react";
import { X, Settings, Bot, Zap, Shield, FileText, Users, Clock, Play, Webhook, Bell, Globe, Code, Cpu, Briefcase, Book } from "lucide-react";
import type { Node } from "@xyflow/react";

interface PropertiesPanelProps {
  nodeId: string;
  node?: Node;
  onClose: () => void;
  onUpdate?: (updates: Record<string, any>) => void;
  readonly?: boolean;
  // Resource options for dropdowns
  llmOptions?: string[];
  toolOptions?: string[];
  guardrailOptions?: string[];
  documentStoreOptions?: string[];
  agentOptions?: string[];
}

// Helper to validate strict schema rules
const validateStrictSchema = (schema: any, path: string = ''): string | null => {
  if (!schema || typeof schema !== 'object') return null;

  const type = schema.type;
  if (type === 'object' || schema.properties) {
    if (schema.additionalProperties !== false) {
      return `${path ? path : 'root'}: additionalProperties must be false in strict mode`;
    }

    const required = new Set(Array.isArray(schema.required) ? schema.required : []);
    const properties = schema.properties || {};

    for (const key of Object.keys(properties)) {
      if (!required.has(key)) {
        return `${path ? path : 'root'}: property '${key}' must be listed in required array`;
      }

      const error = validateStrictSchema(properties[key], path ? `${path}.${key}` : key);
      if (error) return error;
    }
  }
  return null;
};

export const PropertiesPanel: React.FC<PropertiesPanelProps> = ({
  nodeId,
  node,
  onClose,
  onUpdate,
  readonly = false,
  llmOptions = [],
  toolOptions = [],
  guardrailOptions = [],
  documentStoreOptions = [],
  agentOptions = [],
}) => {
  if (!node) return null;

  const data = node.data as any;
  const agentType = data?.agentType || 'llm';
  const isWorkflow = ['sequential', 'parallel', 'loop', 'conditional'].includes(agentType);
  const isConditional = agentType === 'conditional';
  const isRemote = agentType === 'remote';
  const isRunner = agentType === 'runner';

  const [localData, setLocalData] = React.useState({
    label: data?.label || '',
    description: data?.description || '',
    instruction: data?.instruction || '',
    instruction_file: data?.instruction_file || '',
    llm: data?.llm || '',
    tools: data?.tools || [],
    guardrails: data?.guardrails || '',
    documentStores: data?.documentStores || [],
    subAgents: data?.subAgents || [],
    agentTools: data?.agentTools || [],

    context: data?.context || { strategy: 'none', window_size: 0, budget: 0 },
    url: data?.url || '',
    // Trigger config (schedule or webhook)
    trigger: data?.trigger || {
      type: '',
      enabled: true,
      // Schedule fields
      cron: '',
      timezone: 'UTC',
      input: '',
      // Webhook fields
      path: '',
      methods: ['POST'],
      secret: '',
      signature_header: '',
      webhook_input: { template: '', session_id: '' },
      response: { mode: 'sync' }
    },
    // Notifications list
    notifications: data?.notifications || [],

    // Missing Schema Fields
    reasoning: data?.reasoning || {
      max_iterations: 100,
      enable_exit_tool: false,
      enable_escalate_tool: false,
      termination_conditions: [],
      completion_instruction: ''
    },
    // Conditional Agent
    condition_agent: data?.condition_agent || '',
    condition_field: data?.condition_field || 'safe',
    on_true_agent: data?.on_true_agent || '',
    on_false_response: data?.on_false_response || 'I cannot process this request.',
    // Safety
    disable_safety_protocols: data?.disable_safety_protocols || false,

    prompt: data?.prompt || { system_prompt: '', role: '', guidance: '' },
    structured_output: data?.structured_output ? {
      ...data.structured_output,
      schemaStr: data.structured_output.schema ? JSON.stringify(data.structured_output.schema, null, 2) : ''
    } : { schemaStr: '', strict: true, name: 'response' },
    include_context: data?.include_context || false,
    include_context_limit: data?.include_context_limit || 5,
    include_context_max_length: data?.include_context_max_length || 500,
    streaming: data?.streaming !== false, // default true
    input_modes: data?.input_modes || ['text/plain'],
    output_modes: data?.output_modes || ['text/plain'],

    skills: data?.skills || [],
    agent_card_url: data?.agent_card_url || '',
    timeout: data?.timeout || 0,
    headers: data?.headers || {},
    headersStr: data?.headers ? JSON.stringify(data.headers, null, 2) : '',
  });

  const [customTool, setCustomTool] = React.useState('');

  const handleAddCustomTool = () => {
    if (!customTool.trim()) return;

    // Check if valid format (alphanumeric + underscore/dash usually)
    const toolName = customTool.trim();

    // Don't add if already exists
    if (!localData.tools.includes(toolName)) {
      handleChange('tools', [...localData.tools, toolName]);
    }

    setCustomTool('');
  };

  React.useEffect(() => {
    setLocalData({
      label: data?.label || '',
      description: data?.description || '',
      instruction: data?.instruction || '',
      instruction_file: data?.instruction_file || '',
      llm: data?.llm || '',
      tools: data?.tools || [],
      guardrails: data?.guardrails || '',
      documentStores: data?.documentStores || [],
      subAgents: data?.subAgents || [],
      agentTools: data?.agentTools || [],

      context: data?.context || { strategy: 'none', window_size: 0, budget: 0 },
      url: data?.url || '',
      trigger: data?.trigger || { type: '', cron: '', timezone: 'UTC', input: '', enabled: true },
      notifications: data?.notifications || [],

      reasoning: data?.reasoning || {
        max_iterations: 100,
        enable_exit_tool: false,
        enable_escalate_tool: false,
        termination_conditions: [],
        completion_instruction: ''
      },
      // Conditional Agent
      condition_agent: data?.condition_agent || '',
      condition_field: data?.condition_field || 'safe',
      on_true_agent: data?.on_true_agent || '',
      on_false_response: data?.on_false_response || 'I cannot process this request.',
      disable_safety_protocols: data?.disable_safety_protocols || false,

      prompt: data?.prompt || { system_prompt: '', role: '', guidance: '' },
      structured_output: data?.structured_output ? {
        ...data.structured_output,
        schemaStr: data.structured_output.schema ? JSON.stringify(data.structured_output.schema, null, 2) : ''
      } : { schemaStr: '', strict: true, name: 'response' },
      include_context: data?.include_context || false,
      include_context_limit: data?.include_context_limit || 5,
      include_context_max_length: data?.include_context_max_length || 500,
      streaming: data?.streaming !== false,
      input_modes: data?.input_modes || ['text/plain'],
      output_modes: data?.output_modes || ['text/plain'],

      skills: data?.skills || [],
      agent_card_url: data?.agent_card_url || '',
      timeout: data?.timeout || 0,
      headers: data?.headers || {},
      headersStr: data?.headers ? JSON.stringify(data.headers, null, 2) : '',
    });
  }, [nodeId]);

  const handleChange = (field: string, value: any) => {
    setLocalData((prev) => ({ ...prev, [field]: value }));
    if (!readonly && onUpdate) {
      onUpdate({ [field]: value });
    }
  };

  const handleMultiSelect = (field: string, value: string, checked: boolean) => {
    const currentValues = localData[field as keyof typeof localData] as string[] || [];
    const newValues = checked
      ? [...currentValues, value]
      : currentValues.filter((v) => v !== value);
    handleChange(field, newValues);
  };

  return (
    <div className="w-80 bg-black/40 border-l border-white/10 flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-white/10">
        <div className="flex items-center gap-2">
          <Settings size={18} className="text-gray-400" />
          <h3 className="font-semibold text-sm">Agent Properties</h3>
        </div>
        <button
          onClick={onClose}
          className="p-1 hover:bg-white/10 rounded transition-colors"
          title="Close"
        >
          <X size={18} />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Agent ID (readonly) */}
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">
            Agent ID
          </label>
          <div className="px-3 py-2 bg-white/5 border border-white/10 rounded text-sm font-mono text-gray-400">
            {nodeId}
          </div>
        </div>

        {/* Agent Type selector */}
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">
            Agent Type
          </label>
          <select
            value={agentType}
            onChange={(e) => handleChange('agentType', e.target.value)}
            disabled={readonly}
            className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
          >
            <option value="llm">LLM Agent</option>
            <option value="runner">Runner (Tool Pipeline)</option>
            <option value="sequential">Sequential Workflow</option>
            <option value="parallel">Parallel Workflow</option>
            <option value="loop">Loop Workflow</option>
            <option value="conditional">Conditional Router</option>
            <option value="remote">Remote Agent</option>
          </select>
        </div>

        {/* Basic Info Section */}
        <div className="space-y-3">
          <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
            <Bot size={12} />
            <span>Basic Info</span>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5">Display Name</label>
            <input
              type="text"
              value={localData.label}
              onChange={(e) => handleChange('label', e.target.value)}
              disabled={readonly}
              className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
              placeholder="e.g., Research Assistant"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1.5">Description</label>
            <textarea
              value={localData.description}
              onChange={(e) => handleChange('description', e.target.value)}
              disabled={readonly}
              className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green resize-none disabled:opacity-50"
              rows={2}
              placeholder="What does this agent do?"
            />
          </div>
        </div>



        {isConditional && (
          <div className="space-y-3 pt-2 border-t border-white/10">
            <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
              <Bot size={12} />
              <span>Conditional Logic</span>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1.5">Condition Agent</label>
              <select
                value={localData.condition_agent}
                onChange={(e) => handleChange('condition_agent', e.target.value)}
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
              >
                <option value="">Select evaluator agent...</option>
                {agentOptions.map((agent) => (
                  <option key={agent} value={agent}>{agent}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1.5">Condition Field</label>
              <input
                type="text"
                value={localData.condition_field}
                onChange={(e) => handleChange('condition_field', e.target.value)}
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                placeholder="JSON field (e.g. safe)"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1.5">On True Agent</label>
              <select
                value={localData.on_true_agent}
                onChange={(e) => handleChange('on_true_agent', e.target.value)}
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
              >
                <option value="">Select agent to run...</option>
                {agentOptions.map((agent) => (
                  <option key={agent} value={agent}>{agent}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1.5">On False Response</label>
              <textarea
                value={localData.on_false_response}
                onChange={(e) => handleChange('on_false_response', e.target.value)}
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green resize-none disabled:opacity-50"
                rows={2}
                placeholder="Message to return if false..."
              />
            </div>
          </div>
        )}

        {/* LLM & Tools Section (for non-workflow agents) */}
        {!isWorkflow && !isRemote && !isRunner && (<>
          <div className="space-y-3 pt-2 border-t border-white/10">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
                <Cpu size={12} />
                <span>Core Intelligence</span>
              </div>
              <div className="flex items-center gap-2">
                <label className="text-xs text-gray-500 font-medium">Streaming</label>
                <button
                  onClick={() => handleChange('streaming', !localData.streaming)}
                  className={`relative inline-flex h-4 w-7 items-center rounded-full transition-colors ${localData.streaming ? 'bg-hector-green' : 'bg-white/10'}`}
                >
                  <span className={`inline-block h-3 w-3 transform rounded-full bg-white transition-transform ${localData.streaming ? 'translate-x-[14px]' : 'translate-x-0.5'}`} />
                </button>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1.5">LLM</label>
              <select
                value={localData.llm}
                onChange={(e) => handleChange('llm', e.target.value)}
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
              >
                <option value="">Select LLM...</option>
                {llmOptions.map((llm) => (
                  <option key={llm} value={llm}>{llm}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1.5">Instruction</label>
              <textarea
                value={localData.instruction}
                onChange={(e) => handleChange('instruction', e.target.value)}
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm font-mono focus:outline-none focus:border-hector-green resize-none disabled:opacity-50"
                rows={4}
                placeholder="System prompt for this agent..."
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1.5">Instruction File <span className="text-xs text-gray-500 font-normal">(Optional path to markdown file)</span></label>
              <div className="relative">
                <FileText size={14} className="absolute left-3 top-2.5 text-gray-500" />
                <input
                  type="text"
                  value={localData.instruction_file || ''}
                  onChange={(e) => handleChange('instruction_file', e.target.value)}
                  disabled={readonly}
                  className="w-full pl-9 pr-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50 font-mono"
                  placeholder="e.g. prompts/coder.md"
                />
              </div>
            </div>

            {/* Prompt Details (Collapsed by default logic or always visible? Always visible for now) */}
            <div className="space-y-2 pt-2 border-t border-white/5">
              <label className="block text-xs font-medium text-gray-500 uppercase">Prompt Details</label>
              <input
                type="text"
                value={localData.prompt.role || ''}
                onChange={(e) => handleChange('prompt', { ...localData.prompt, role: e.target.value })}
                className="w-full px-3 py-1.5 bg-white/5 border border-white/10 rounded text-xs focus:outline-none focus:border-hector-green"
                placeholder="Role (e.g. 'Senior Engineer')"
              />
              <input
                type="text"
                value={localData.prompt.guidance || ''}
                onChange={(e) => handleChange('prompt', { ...localData.prompt, guidance: e.target.value })}
                className="w-full px-3 py-1.5 bg-white/5 border border-white/10 rounded text-xs focus:outline-none focus:border-hector-green"
                placeholder="Guidance (Additional instructions)"
              />
              <div className="relative">
                <textarea
                  value={localData.prompt.system_prompt || ''}
                  onChange={(e) => handleChange('prompt', { ...localData.prompt, system_prompt: e.target.value })}
                  className="w-full px-3 py-1.5 bg-white/5 border border-white/10 rounded text-xs font-mono focus:outline-none focus:border-hector-green resize-none"
                  rows={2}
                  placeholder="System Prompt (Available override)"
                />
              </div>
            </div>

            {/* Reasoning Configuration */}
            <div className="space-y-2 pt-2 border-t border-white/5">
              <div className="flex items-center justify-between">
                <label className="text-xs font-medium text-gray-500 uppercase">Reasoning</label>
                <div className="flex items-center gap-2 text-xs">
                  <span className="text-gray-500">Max Iterations</span>
                  <input
                    type="number"
                    value={localData.reasoning.max_iterations}
                    onChange={(e) => handleChange('reasoning', { ...localData.reasoning, max_iterations: parseInt(e.target.value) })}
                    className="w-12 bg-white/5 border border-white/10 rounded px-1 py-0.5 text-center"
                  />
                </div>
              </div>

              <div className="flex gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={localData.reasoning.enable_exit_tool}
                    onChange={(e) => handleChange('reasoning', { ...localData.reasoning, enable_exit_tool: e.target.checked })}
                    className="rounded border-white/20 bg-white/5 text-hector-green"
                  />
                  <span className="text-xs text-gray-400">Exit Tool</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={localData.reasoning.enable_escalate_tool}
                    onChange={(e) => handleChange('reasoning', { ...localData.reasoning, enable_escalate_tool: e.target.checked })}
                    className="rounded border-white/20 bg-white/5 text-hector-green"
                  />
                  <span className="text-xs text-gray-400">Escalate Tool</span>
                </label>
              </div>

              <div className="text-xs text-gray-500 mt-1">
                Termination:
                {['no_tool_calls', 'escalate', 'transfer'].map(cond => (
                  <label key={cond} className="inline-flex items-center gap-1 ml-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={(localData.reasoning.termination_conditions || []).includes(cond)}
                      onChange={(e) => {
                        const current = localData.reasoning.termination_conditions || [];
                        const next = e.target.checked ? [...current, cond] : current.filter((c: string) => c !== cond);
                        handleChange('reasoning', { ...localData.reasoning, termination_conditions: next });
                      }}
                      className="hidden"
                    />
                    <span className={`px-1.5 py-0.5 rounded border ${(localData.reasoning.termination_conditions || []).includes(cond)
                      ? 'border-hector-green text-hector-green bg-hector-green/10'
                      : 'border-white/10 text-gray-500'
                      }`}>{cond}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Tools Selection */}
            {(toolOptions.length > 0 || localData.tools.length > 0) && (
              <div className="pt-2 border-t border-white/5">
                <label className="block text-sm font-medium mb-1.5">Tools</label>
                <div className="space-y-1 max-h-32 overflow-y-auto bg-white/5 border border-white/10 rounded p-2">
                  {toolOptions.map((tool) => (
                    <label key={tool} className="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded">
                      <input
                        type="checkbox"
                        checked={localData.tools.includes(tool)}
                        onChange={(e) => handleMultiSelect('tools', tool, e.target.checked)}
                        disabled={readonly}
                        className="rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
                      />
                      <span>{tool}</span>
                    </label>
                  ))}
                  {localData.tools.filter((t: string) => !toolOptions.includes(t)).map((tool: string) => (
                    <label key={tool} className="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded group">
                      <input
                        type="checkbox"
                        checked={true}
                        onChange={() => handleMultiSelect('tools', tool, false)}
                        className="rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
                      />
                      <span className="flex-1 truncate text-gray-300" title={tool}>{tool}</span>
                    </label>
                  ))}
                </div>
                {!readonly && (
                  <div className="mt-2 flex gap-2">
                    <input
                      type="text"
                      value={customTool}
                      onChange={(e) => setCustomTool(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && handleAddCustomTool()}
                      className="flex-1 px-2 py-1 bg-white/5 border border-white/10 rounded text-xs focus:outline-none focus:border-hector-green placeholder-gray-600"
                      placeholder="Add custom tool..."
                    />
                    <button onClick={handleAddCustomTool} disabled={!customTool.trim()} className="px-2 py-1 bg-white/10 hover:bg-white/20 text-white text-xs rounded">Add</button>
                  </div>
                )}
              </div>
            )}

            {/* Task Tracking Shortcut */}
            <div className="pt-2 border-t border-white/5">
              <label className="flex items-center gap-2 cursor-pointer p-2 hover:bg-white/5 rounded transition-colors group">
                <input
                  type="checkbox"
                  checked={localData.tools.includes('todo_write')}
                  onChange={(e) => handleMultiSelect('tools', 'todo_write', e.target.checked)}
                  disabled={readonly}
                  className="rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
                />
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="block text-sm font-medium text-gray-200 group-hover:text-white transition-colors">
                      Enable Task Tracking
                    </span>
                    <span className="bg-white/10 text-[10px] px-1.5 py-0.5 rounded text-gray-400 font-mono">
                      todo_write
                    </span>
                  </div>
                  <span className="block text-xs text-gray-500 mt-0.5">
                    Adds a persistent task manager tool. Recommended for complex, multi-step agents.
                  </span>
                </div>
              </label>
            </div>

            {/* Structured Output */}
            <div className="pt-2 border-t border-white/5">
              <div className="flex items-center justify-between mb-1.5">
                <label className="block text-xs font-medium text-gray-500 uppercase">Structured Output</label>
                <div className="flex items-center gap-2">
                  <span className="text-[10px] text-gray-500 uppercase tracking-wider">Strict</span>
                  <input
                    type="checkbox"
                    checked={localData.structured_output?.strict}
                    onChange={(e) => handleChange('structured_output', { ...localData.structured_output, strict: e.target.checked })}
                    className="rounded border-white/20 bg-white/5 text-hector-green"
                  />
                </div>
              </div>
              {(() => {
                const validationError = localData.structured_output?.strict && localData.structured_output?.schema
                  ? validateStrictSchema(localData.structured_output.schema)
                  : null;

                return (
                  <div className="space-y-1">
                    <textarea
                      value={localData.structured_output?.schemaStr || ''}
                      onChange={(e) => {
                        const str = e.target.value;
                        let schema = null;
                        try { schema = JSON.parse(str); } catch { }
                        const newStruc = { ...localData.structured_output, schemaStr: str, schema: schema };
                        setLocalData(prev => ({ ...prev, structured_output: newStruc }));
                        if (onUpdate && !readonly) onUpdate({ structured_output: newStruc });
                      }}
                      className={`w-full px-3 py-2 bg-white/5 border rounded text-xs font-mono focus:outline-none resize-none text-gray-300 ${validationError ? 'border-red-500/50 focus:border-red-500' : 'border-white/10 focus:border-hector-green'
                        }`}
                      rows={3}
                      placeholder='JSON Schema { "type": "object", ... }'
                    />
                    {validationError && (
                      <div className="text-[10px] text-red-400 flex items-start gap-1 p-1 bg-red-500/10 rounded border border-red-500/20">
                        <span className="font-bold">Error:</span>
                        <span className="break-all">{validationError}</span>
                      </div>
                    )}
                  </div>
                );
              })()}
            </div>

          </div>

          {/* Capabilities Section */}
          <div className="pt-2 border-t border-white/10 space-y-3">
            <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
              <Briefcase size={12} />
              <span>Capabilities</span>
            </div>

            {/* I/O Modes */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-xs font-medium text-gray-500">I/O Interface</label>
              </div>
              <div className="space-y-3 bg-white/5 p-2 rounded border border-white/10">
                <div>
                  <label className="block text-[10px] text-gray-400 mb-1">Input Modes</label>
                  <div className="flex flex-wrap gap-1 bg-black/20 border border-white/10 rounded px-2 py-1 min-h-[28px]">
                    {(localData.input_modes || []).map((mode: string, i: number) => (
                      <span key={i} className="inline-flex items-center gap-1 bg-white/10 text-[10px] px-1.5 py-0.5 rounded text-gray-300 border border-white/5">
                        {mode}
                        <button
                          onClick={() => handleChange('input_modes', localData.input_modes.filter((_: string, idx: number) => idx !== i))}
                          className="hover:text-white transition-colors"
                        >
                          <X size={10} />
                        </button>
                      </span>
                    ))}
                    <input
                      type="text"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ',') {
                          e.preventDefault();
                          const val = e.currentTarget.value.trim();
                          if (val && !localData.input_modes?.includes(val)) {
                            handleChange('input_modes', [...(localData.input_modes || []), val]);
                            e.currentTarget.value = '';
                          }
                        } else if (e.key === 'Backspace' && !e.currentTarget.value && localData.input_modes?.length > 0) {
                          handleChange('input_modes', localData.input_modes.slice(0, -1));
                        }
                      }}
                      placeholder={localData.input_modes?.length ? "" : "text/plain"}
                      className="flex-1 bg-transparent text-[10px] focus:outline-none min-w-[60px] py-0.5"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-[10px] text-gray-400 mb-1">Output Modes</label>
                  <div className="flex flex-wrap gap-1 bg-black/20 border border-white/10 rounded px-2 py-1 min-h-[28px]">
                    {(localData.output_modes || []).map((mode: string, i: number) => (
                      <span key={i} className="inline-flex items-center gap-1 bg-white/10 text-[10px] px-1.5 py-0.5 rounded text-gray-300 border border-white/5">
                        {mode}
                        <button
                          onClick={() => handleChange('output_modes', localData.output_modes.filter((_: string, idx: number) => idx !== i))}
                          className="hover:text-white transition-colors"
                        >
                          <X size={10} />
                        </button>
                      </span>
                    ))}
                    <input
                      type="text"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ',') {
                          e.preventDefault();
                          const val = e.currentTarget.value.trim();
                          if (val && !localData.output_modes?.includes(val)) {
                            handleChange('output_modes', [...(localData.output_modes || []), val]);
                            e.currentTarget.value = '';
                          }
                        } else if (e.key === 'Backspace' && !e.currentTarget.value && localData.output_modes?.length > 0) {
                          handleChange('output_modes', localData.output_modes.slice(0, -1));
                        }
                      }}
                      placeholder={localData.output_modes?.length ? "" : "text/plain"}
                      className="flex-1 bg-transparent text-[10px] focus:outline-none min-w-[60px] py-0.5"
                    />
                  </div>
                </div>
              </div>
            </div>

            {/* Skills List */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-xs font-medium text-gray-500">Skills</label>
                <button
                  onClick={() => {
                    const newSkill = { id: '', name: '', description: '' };
                    handleChange('skills', [...(localData.skills || []), newSkill]);
                  }}
                  className="text-[10px] bg-white/10 hover:bg-white/20 px-1.5 py-0.5 rounded text-white transition-colors"
                >
                  + Add
                </button>
              </div>

              <div className="space-y-2">
                {(localData.skills || []).map((skill: any, idx: number) => (
                  <div key={idx} className="bg-white/5 p-2 rounded border border-white/10 space-y-2">
                    <div className="flex items-center gap-2">
                      <input
                        value={skill.id}
                        onChange={(e) => {
                          const newSkills = [...localData.skills];
                          newSkills[idx] = { ...skill, id: e.target.value };
                          handleChange('skills', newSkills);
                        }}
                        placeholder="Skill ID"
                        className="flex-1 bg-transparent text-xs border-b border-white/10 focus:border-hector-green focus:outline-none pb-0.5"
                      />
                      <button onClick={() => {
                        const newSkills = localData.skills.filter((_: any, i: number) => i !== idx);
                        handleChange('skills', newSkills);
                      }} className="text-gray-500 hover:text-red-400"><X size={12} /></button>
                    </div>
                    <input
                      value={skill.name}
                      onChange={(e) => {
                        const newSkills = [...localData.skills];
                        newSkills[idx] = { ...skill, name: e.target.value };
                        handleChange('skills', newSkills);
                      }}
                      placeholder="Display Name"
                      className="w-full bg-transparent text-xs text-gray-300 focus:outline-none"
                    />
                    <textarea
                      value={skill.description}
                      onChange={(e) => {
                        const newSkills = [...localData.skills];
                        newSkills[idx] = { ...skill, description: e.target.value };
                        handleChange('skills', newSkills);
                      }}
                      placeholder="Description"
                      rows={2}
                      className="w-full bg-black/20 text-xs text-gray-400 p-1.5 rounded border border-white/5 focus:border-white/20 focus:outline-none resize-none"
                    />
                    <input
                      value={skill.tags?.join(', ') || ''}
                      onChange={(e) => {
                        const newSkills = [...localData.skills];
                        newSkills[idx] = { ...skill, tags: e.target.value.split(',').map((s: string) => s.trim()).filter(Boolean) };
                        handleChange('skills', newSkills);
                      }}
                      placeholder="Tags (comma-separated)"
                      className="w-full bg-transparent text-xs text-gray-500 border-b border-white/5 focus:border-white/20 focus:outline-none pb-0.5"
                    />
                    <textarea
                      value={skill.examples?.join('\n') || ''}
                      onChange={(e) => {
                        const newSkills = [...localData.skills];
                        newSkills[idx] = { ...skill, examples: e.target.value.split('\n').filter(Boolean) };
                        handleChange('skills', newSkills);
                      }}
                      placeholder="Examples (one per line)"
                      rows={2}
                      className="w-full bg-black/20 text-xs text-gray-400 p-1.5 rounded border border-white/5 focus:border-white/20 focus:outline-none resize-none"
                    />

                  </div>
                ))}
                {(localData.skills || []).length === 0 && (
                  <div className="text-center text-[10px] text-gray-600 italic">No skills defined</div>
                )}
              </div>
            </div>

            {/* Agent Tools */}
            {!isWorkflow && agentOptions.length > 0 && (
              <div>
                <label className="block text-sm font-medium mb-1.5">Agent Tools</label>
                <div className="space-y-1 max-h-24 overflow-y-auto bg-white/5 border border-white/10 rounded p-2">
                  {agentOptions.map((agent) => (
                    <label key={agent} className="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded">
                      <input
                        type="checkbox"
                        checked={localData.agentTools.includes(agent)}
                        onChange={(e) => handleMultiSelect('agentTools', agent, e.target.checked)}
                        disabled={readonly}
                        className="rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
                      />
                      <span>{agent}</span>
                    </label>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="pt-2 border-t border-white/10">
            <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
              <Clock size={12} />
              <span>Context & Memory</span>
            </div>

            <div className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Strategy</label>
                <select
                  value={localData.context.strategy || 'none'}
                  onChange={(e) => handleChange('context', { ...localData.context, strategy: e.target.value })}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                >
                  <option value="none">None (Full History)</option>
                  <option value="buffer_window">Buffer Window</option>
                  <option value="token_window">Token Window</option>
                  <option value="summary_buffer">Summary Buffer</option>
                </select>
              </div>

              {(localData.context.strategy === 'buffer_window' || localData.context.strategy === 'token_window') && (
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">
                    {localData.context.strategy === 'buffer_window' ? 'Window Size (messages)' : 'Token Budget'}
                  </label>
                  <input
                    type="number"
                    value={localData.context.strategy === 'buffer_window'
                      ? (localData.context.window_size || '')
                      : (localData.context.budget || '')}
                    onChange={(e) => {
                      const val = parseInt(e.target.value) || 0;
                      if (localData.context.strategy === 'buffer_window') {
                        handleChange('context', { ...localData.context, window_size: val });
                      } else {
                        handleChange('context', { ...localData.context, budget: val });
                      }
                    }}
                    disabled={readonly}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                    placeholder={localData.context.strategy === 'buffer_window' ? 'e.g. 10' : 'e.g. 4096'}
                  />
                </div>
              )}

              {localData.context.strategy === 'summary_buffer' && (
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">Token Budget</label>
                  <input
                    type="number"
                    value={localData.context.budget || ''}
                    onChange={(e) => handleChange('context', { ...localData.context, budget: parseInt(e.target.value) || 0 })}
                    disabled={readonly}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                    placeholder="e.g. 4096"
                  />
                </div>
              )}

              {/* Advanced settings for summary_buffer only */}
              {localData.context.strategy === 'summary_buffer' && (
                <div className="space-y-2 pt-2 border-t border-white/5">
                  <label className="block text-xs font-medium text-gray-500 uppercase">Summary Buffer Settings</label>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="block text-[10px] text-gray-400">Threshold</label>
                      <input
                        type="number" step="0.1" min="0" max="1"
                        value={localData.context?.threshold || ''}
                        onChange={(e) => handleChange('context', { ...localData.context, threshold: parseFloat(e.target.value) })}
                        placeholder="0.85"
                        className="w-full bg-white/5 border border-white/10 rounded px-1.5 py-1 text-xs focus:outline-none focus:border-hector-green"
                      />
                    </div>
                    <div>
                      <label className="block text-[10px] text-gray-400">Target</label>
                      <input
                        type="number" step="0.1" min="0" max="1"
                        value={localData.context?.target || ''}
                        onChange={(e) => handleChange('context', { ...localData.context, target: parseFloat(e.target.value) })}
                        placeholder="0.6"
                        className="w-full bg-white/5 border border-white/10 rounded px-1.5 py-1 text-xs focus:outline-none focus:border-hector-green"
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-[10px] text-gray-400 mb-1">Summarizer LLM (optional)</label>
                    <select
                      value={localData.context?.summarizer_llm || ''}
                      onChange={(e) => handleChange('context', { ...localData.context, summarizer_llm: e.target.value || undefined })}
                      className="w-full bg-white/5 border border-white/10 rounded px-1.5 py-1 text-xs focus:outline-none focus:border-hector-green"
                    >
                      <option value="">Use agent's LLM</option>
                      {llmOptions.map(llm => (
                        <option key={llm} value={llm}>{llm}</option>
                      ))}
                    </select>
                  </div>
                </div>
              )}

              {/* Preserve Recent for token_window only (summary_buffer uses internal constant) */}
              {localData.context.strategy === 'token_window' && (
                <div className="pt-2">
                  <label className="block text-[10px] text-gray-400 mb-1">Preserve Recent Messages</label>
                  <input
                    type="number" min="0"
                    value={localData.context?.preserve_recent || ''}
                    onChange={(e) => handleChange('context', { ...localData.context, preserve_recent: parseInt(e.target.value) })}
                    placeholder="5"
                    className="w-full bg-white/5 border border-white/10 rounded px-1.5 py-1 text-xs focus:outline-none focus:border-hector-green"
                  />
                </div>
              )}
            </div>
          </div>

          {/* Knowledge (RAG) Section */}
          <div className="pt-2 border-t border-white/10">
            <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
              <Book size={12} />
              <span>Knowledge Base</span>
            </div>

            {documentStoreOptions.length > 0 ? (
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="block text-sm font-medium flex items-center gap-2">
                    Select Stores
                  </label>
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] text-gray-500 uppercase tracking-wider">Inject Context</span>
                    <input
                      type="checkbox"
                      checked={localData.include_context}
                      onChange={(e) => handleChange('include_context', e.target.checked)}
                      className="rounded border-white/20 bg-white/5 text-hector-green"
                    />
                  </div>
                </div>

                {localData.include_context && (
                  <div className="grid grid-cols-2 gap-2 bg-white/5 p-2 rounded border border-white/10 mb-2">
                    <div>
                      <label className="block text-[10px] text-gray-400">Limit</label>
                      <input
                        type="number"
                        value={localData.include_context_limit}
                        onChange={(e) => handleChange('include_context_limit', parseInt(e.target.value))}
                        className="w-full bg-black/20 border border-white/10 rounded px-1.5 py-0.5 text-xs focus:outline-none focus:border-hector-green"
                      />
                    </div>
                    <div>
                      <label className="block text-[10px] text-gray-400">Max Chars</label>
                      <input
                        type="number"
                        value={localData.include_context_max_length}
                        onChange={(e) => handleChange('include_context_max_length', parseInt(e.target.value))}
                        className="w-full bg-black/20 border border-white/10 rounded px-1.5 py-0.5 text-xs focus:outline-none focus:border-hector-green"
                      />
                    </div>
                  </div>
                )}

                <div className="space-y-1 max-h-24 overflow-y-auto bg-white/5 border border-white/10 rounded p-2">
                  {documentStoreOptions.map((store) => (
                    <label key={store} className="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded">
                      <input
                        type="checkbox"
                        checked={localData.documentStores.includes(store)}
                        onChange={(e) => handleMultiSelect('documentStores', store, e.target.checked)}
                        disabled={readonly}
                        className="rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
                      />
                      <span>{store}</span>
                    </label>
                  ))}
                </div>
              </div>
            ) : (
              <div className="text-xs text-gray-500 italic p-2 border border-white/5 rounded border-dashed text-center">
                No document stores available.
              </div>
            )}
          </div>

          {/* Safety Configuration */}
          <div className="pt-2 border-t border-white/10">
            <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
              <Shield size={12} />
              <span>Safety & Governance</span>
            </div>

            {guardrailOptions.length > 0 && (
              <div className="mb-3 pb-3 border-b border-white/5">
                <label className="block text-sm font-medium mb-1.5">Guardrails</label>
                <select
                  value={localData.guardrails}
                  onChange={(e) => handleChange('guardrails', e.target.value)}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                >
                  <option value="">None</option>
                  {guardrailOptions.map((g) => (
                    <option key={g} value={g}>{g}</option>
                  ))}
                </select>
              </div>
            )}

            <label className="flex items-start gap-2 cursor-pointer p-2 hover:bg-white/5 rounded transition-colors group">
              <input
                type="checkbox"
                checked={localData.disable_safety_protocols}
                onChange={(e) => handleChange('disable_safety_protocols', e.target.checked)}
                disabled={readonly}
                className="mt-0.5 rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
              />
              <div className="flex-1">
                <span className="block text-sm font-medium text-gray-200 group-hover:text-white transition-colors">
                  Disable Safety Protocols
                </span>
                <span className="block text-xs text-gray-500 mt-0.5">
                  Disable automatic injection of "Tool Reliability" and "Chain of Thought" system prompts.
                  Enable this only if providing your own robust prompt engineering.
                </span>
              </div>
            </label>
          </div>
        </>)}

        {/* Workflow Sub-Agents Section */}
        {
          isWorkflow && agentOptions.length > 0 && (
            <div className="space-y-3 pt-2 border-t border-white/10">
              <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
                <Users size={12} />
                <span>Sub-Agents</span>
              </div>

              <div className="space-y-1 max-h-40 overflow-y-auto bg-white/5 border border-white/10 rounded p-2">
                {agentOptions.map((agent) => (
                  <label key={agent} className="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded">
                    <input
                      type="checkbox"
                      checked={localData.subAgents.includes(agent)}
                      onChange={(e) => handleMultiSelect('subAgents', agent, e.target.checked)}
                      disabled={readonly}
                      className="rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
                    />
                    <span>{agent}</span>
                  </label>
                ))}
              </div>
            </div>
          )
        }

        {/* Remote Agent Configuration */}
        {
          isRemote && (
            <div className="space-y-3 pt-2 border-t border-white/10">
              <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
                <Globe size={12} />
                <span>Remote Agent</span>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Base URL</label>
                <input
                  type="text"
                  value={localData.url}
                  onChange={(e) => handleChange('url', e.target.value)}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                  placeholder="http://localhost:9000"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Agent Card URL</label>
                <input
                  type="text"
                  value={localData.agent_card_url || ''}
                  onChange={(e) => handleChange('agent_card_url', e.target.value)}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                  placeholder="https://..."
                />
              </div>

              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">Timeout (ms)</label>
                  <input
                    type="number"
                    value={localData.timeout || ''}
                    onChange={(e) => handleChange('timeout', parseInt(e.target.value))}
                    disabled={readonly}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1 flex items-center gap-1">
                  <Code size={10} />
                  Headers (JSON)
                </label>
                <textarea
                  value={localData.headersStr || JSON.stringify(localData.headers || {}, null, 2)}
                  onChange={(e) => {
                    const str = e.target.value;
                    try {
                      const headers = JSON.parse(str);
                      handleChange('headers', headers);
                    } catch { }
                    setLocalData(prev => ({ ...prev, headersStr: str }));
                  }}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-xs font-mono focus:outline-none focus:border-hector-green resize-none text-gray-300"
                  rows={2}
                  placeholder="{ 'Authorization': 'Bearer ...' }"
                />
              </div>
            </div>
          )
        }

        {/* Runner Agent Tools */}
        {
          isRunner && toolOptions.length > 0 && (
            <div className="space-y-3 pt-2 border-t border-white/10">
              <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
                <Play size={12} />
                <span>Tool Pipeline</span>
              </div>
              <p className="text-xs text-gray-500">
                Tools execute in order without LLM reasoning. Output of each tool feeds into the next.
              </p>
              <div className="space-y-1 max-h-32 overflow-y-auto bg-white/5 border border-white/10 rounded p-2">
                {/* Show configured tool options */}
                {toolOptions.map((tool) => (
                  <label key={tool} className="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded">
                    <input
                      type="checkbox"
                      checked={localData.tools.includes(tool)}
                      onChange={(e) => handleMultiSelect('tools', tool, e.target.checked)}
                      disabled={readonly}
                      className="rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
                    />
                    <span>{tool}</span>
                  </label>
                ))}

                {/* Show custom added tools that aren't in options */}
                {localData.tools
                  .filter((t: string) => !toolOptions.includes(t))
                  .map((tool: string) => (
                    <label key={tool} className="flex items-center gap-2 text-sm cursor-pointer hover:bg-white/5 p-1 rounded group">
                      <input
                        type="checkbox"
                        checked={true}
                        onChange={() => handleMultiSelect('tools', tool, false)}
                        disabled={readonly}
                        className="rounded border-white/20 bg-white/5 text-hector-green focus:ring-hector-green"
                      />
                      <span className="flex-1 truncate text-gray-300" title={tool}>{tool}</span>
                      <button
                        onClick={(e) => {
                          e.preventDefault();
                          handleMultiSelect('tools', tool, false);
                        }}
                        className="opacity-0 group-hover:opacity-100 p-0.5 hover:bg-white/10 rounded text-gray-500 hover:text-red-400 transition-all"
                      >
                        <X size={12} />
                      </button>
                    </label>
                  ))}
              </div>

              {/* Custom tool input */}
              {!readonly && (
                <div className="mt-2 flex gap-2">
                  <input
                    type="text"
                    value={customTool}
                    onChange={(e) => setCustomTool(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        handleAddCustomTool();
                      }
                    }}
                    className="flex-1 px-2 py-1 bg-white/5 border border-white/10 rounded text-xs focus:outline-none focus:border-hector-green placeholder-gray-600"
                    placeholder="Add custom tool..."
                  />
                  <button
                    onClick={handleAddCustomTool}
                    disabled={!customTool.trim()}
                    className="px-2 py-1 bg-white/10 hover:bg-white/20 text-white text-xs rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Add
                  </button>
                </div>
              )}
            </div>
          )
        }

        {/* Trigger Configuration */}
        <div className="space-y-3 pt-2 border-t border-white/10">
          <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
            <Zap size={12} />
            <span>Trigger</span>
          </div>

          <div>
            <label className="block text-xs font-medium text-gray-500 mb-1">Trigger Type</label>
            <select
              value={localData.trigger?.type || ''}
              onChange={(e) => handleChange('trigger', {
                ...localData.trigger,
                type: e.target.value
              })}
              disabled={readonly}
              className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
            >
              <option value="">None</option>
              <option value="schedule">Schedule (Cron)</option>
              <option value="webhook">Webhook (HTTP)</option>
            </select>
          </div>

          {/* Schedule Trigger Fields */}
          {localData.trigger?.type === 'schedule' && (
            <div className="space-y-3 pl-4 border-l border-white/10">
              <div className="flex items-center gap-2 text-xs text-gray-400">
                <Clock size={12} />
                <span>Schedule Configuration</span>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Cron Expression</label>
                <input
                  type="text"
                  value={localData.trigger?.cron || ''}
                  onChange={(e) => handleChange('trigger', { ...localData.trigger, cron: e.target.value })}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm font-mono focus:outline-none focus:border-hector-green disabled:opacity-50"
                  placeholder="0 9 * * *"
                />
                <p className="text-xs text-gray-500 mt-1">e.g., "0 9 * * *" = daily at 9am</p>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Timezone</label>
                <input
                  type="text"
                  value={localData.trigger?.timezone || 'UTC'}
                  onChange={(e) => handleChange('trigger', { ...localData.trigger, timezone: e.target.value })}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                  placeholder="UTC"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Input Message</label>
                <textarea
                  value={localData.trigger?.input || ''}
                  onChange={(e) => handleChange('trigger', { ...localData.trigger, input: e.target.value })}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green resize-none disabled:opacity-50"
                  rows={2}
                  placeholder="Input for scheduled runs..."
                />
              </div>
            </div>
          )}

          {/* Webhook Trigger Fields */}
          {localData.trigger?.type === 'webhook' && (
            <div className="space-y-3 pl-4 border-l border-white/10">
              <div className="flex items-center gap-2 text-xs text-gray-400">
                <Webhook size={12} />
                <span>Webhook Configuration</span>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Path</label>
                <input
                  type="text"
                  value={localData.trigger?.path || ''}
                  onChange={(e) => handleChange('trigger', { ...localData.trigger, path: e.target.value })}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm font-mono focus:outline-none focus:border-hector-green disabled:opacity-50"
                  placeholder="/webhooks/my-agent"
                />
                <p className="text-xs text-gray-500 mt-1">Leave empty for default: /webhooks/{'{agent-name}'}</p>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Response Mode</label>
                <select
                  value={localData.trigger?.response?.mode || 'sync'}
                  onChange={(e) => handleChange('trigger', {
                    ...localData.trigger,
                    response: { ...localData.trigger?.response, mode: e.target.value }
                  })}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                >
                  <option value="sync">Sync (wait for completion)</option>
                  <option value="async">Async (return task ID)</option>
                  <option value="callback">Callback (POST result to URL)</option>
                </select>
              </div>

              {localData.trigger?.response?.mode === 'callback' && (
                <div className="pl-2 border-l-2 border-hector-green/30 ml-1">
                  <label className="block text-xs font-medium text-gray-500 mb-1">Callback URL</label>
                  <input
                    type="text"
                    value={localData.trigger?.response?.callback_url || ''}
                    onChange={(e) => handleChange('trigger', {
                      ...localData.trigger,
                      response: { ...localData.trigger?.response, callback_url: e.target.value }
                    })}
                    disabled={readonly}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                    placeholder="https://api.example.com/webhook"
                  />
                  <p className="text-xs text-gray-500 mt-1">Fallback URL if not provided in payload</p>
                </div>
              )}

              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">HMAC Secret</label>
                <input
                  type="text"
                  value={localData.trigger?.secret || ''}
                  onChange={(e) => handleChange('trigger', { ...localData.trigger, secret: e.target.value })}
                  disabled={readonly}
                  className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                  placeholder="Optional signature verification secret"
                />
              </div>

              {localData.trigger?.secret && (
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">Signature Header</label>
                  <input
                    type="text"
                    value={localData.trigger?.signature_header || ''}
                    onChange={(e) => handleChange('trigger', { ...localData.trigger, signature_header: e.target.value })}
                    disabled={readonly}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50"
                    placeholder="X-Webhook-Signature"
                  />
                </div>
              )}

              {/* Webhook Input Configuration */}
              <div className="space-y-3 pt-2 border-t border-white/10">
                <div className="flex items-center gap-2 text-xs text-gray-400">
                  <Code size={12} />
                  <span>Payload Transformation</span>
                </div>

                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">Input Template</label>
                  <textarea
                    value={localData.trigger?.webhook_input?.template || ''}
                    onChange={(e) => handleChange('trigger', {
                      ...localData.trigger,
                      webhook_input: { ...localData.trigger?.webhook_input, template: e.target.value }
                    })}
                    disabled={readonly}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm font-mono focus:outline-none focus:border-hector-green resize-none disabled:opacity-50"
                    rows={4}
                    placeholder={'{{ if eq .Body.action "opened" }}\nProcess: {{ .Body.title }}\n{{ end }}'}
                  />
                  <p className="text-xs text-gray-500 mt-1">Go template to transform webhook payload. Empty output = skip event.</p>
                </div>

                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">Session ID Template</label>
                  <input
                    type="text"
                    value={localData.trigger?.webhook_input?.session_id || ''}
                    onChange={(e) => handleChange('trigger', {
                      ...localData.trigger,
                      webhook_input: { ...localData.trigger?.webhook_input, session_id: e.target.value }
                    })}
                    disabled={readonly}
                    className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm font-mono focus:outline-none focus:border-hector-green disabled:opacity-50"
                    placeholder="pr-{{ .Body.repository.full_name }}-{{ .Body.number }}"
                  />
                  <p className="text-xs text-gray-500 mt-1">Derive session ID for conversation continuity</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Notifications Configuration */}
        <div className="space-y-3 pt-2 border-t border-white/10">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-medium text-gray-500 uppercase tracking-wider">
              <Bell size={12} />
              <span>Notifications</span>
            </div>
            <button
              onClick={() => {
                const newNotif = {
                  id: `notif-${Date.now().toString().slice(-4)}`,
                  type: 'webhook',
                  enabled: true,
                  events: ['task.failed'],
                  url: '',
                  headers: {}
                };
                // @ts-ignore
                handleChange('notifications', [...(localData.notifications || []), newNotif]);
              }}
              className="p-1 hover:bg-white/10 rounded text-gray-400 hover:text-white transition-colors"
              title="Add Notification"
            >
              <Zap size={14} />
            </button>
          </div>

          <div className="space-y-4">
            {/* @ts-ignore */}
            {(localData.notifications || []).map((notif: any, index: number) => (
              <div key={index} className="p-3 bg-white/5 border border-white/10 rounded-lg space-y-3">
                <div className="flex items-center justify-between">
                  <input
                    type="text"
                    value={notif.id}
                    onChange={(e) => {
                      const newNotifs = [...(localData.notifications as any[])];
                      newNotifs[index] = { ...notif, id: e.target.value };
                      handleChange('notifications', newNotifs);
                    }}
                    className="bg-transparent text-xs font-mono text-hector-green focus:outline-none w-24"
                    placeholder="ID"
                  />
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => {
                        const newNotifs = [...(localData.notifications as any[])];
                        newNotifs[index] = { ...notif, enabled: !notif.enabled };
                        handleChange('notifications', newNotifs);
                      }}
                      className={`text-[10px] px-1.5 py-0.5 rounded border ${notif.enabled
                        ? 'border-hector-green text-hector-green bg-hector-green/10'
                        : 'border-gray-600 text-gray-500'
                        }`}
                    >
                      {notif.enabled ? 'ON' : 'OFF'}
                    </button>
                    <button
                      onClick={() => {
                        const newNotifs = (localData.notifications as any[]).filter((_, i) => i !== index);
                        handleChange('notifications', newNotifs);
                      }}
                      className="text-gray-500 hover:text-red-400"
                    >
                      <X size={14} />
                    </button>
                  </div>
                </div>

                <div>
                  <input
                    type="text"
                    value={notif.url || ''}
                    onChange={(e) => {
                      const newNotifs = [...(localData.notifications as any[])];
                      newNotifs[index] = { ...notif, url: e.target.value };
                      handleChange('notifications', newNotifs);
                    }}
                    className="w-full px-2 py-1.5 bg-black/20 border border-white/10 rounded text-xs font-mono focus:outline-none focus:border-hector-green"
                    placeholder="Webhook URL (Slack, etc.)"
                  />
                </div>

                <div>
                  <label className="block text-[10px] text-gray-500 mb-1">Headers (one per line: Key: Value)</label>
                  <textarea
                    defaultValue={
                      notif.headers
                        ? Object.entries(notif.headers as Record<string, string>)
                          .map(([k, v]) => `${k}: ${v}`)
                          .join('\n')
                        : ''
                    }
                    onBlur={(e) => {
                      const lines = e.target.value.split('\n');
                      const headers: Record<string, string> = {};
                      lines.forEach(line => {
                        const colonIdx = line.indexOf(':');
                        if (colonIdx > 0) {
                          const key = line.substring(0, colonIdx).trim();
                          const value = line.substring(colonIdx + 1).trim();
                          if (key) headers[key] = value;
                        }
                      });
                      const newNotifs = [...(localData.notifications as any[])];
                      newNotifs[index] = { ...notif, headers: Object.keys(headers).length > 0 ? headers : undefined };
                      handleChange('notifications', newNotifs);
                    }}
                    rows={2}
                    className="w-full px-2 py-1.5 bg-black/20 border border-white/10 rounded text-xs font-mono focus:outline-none focus:border-hector-green resize-none"
                    placeholder="Authorization: Bearer ${TOKEN}&#10;X-Custom-Header: value"
                  />
                </div>

                <div className="flex flex-wrap gap-2">
                  {['started', 'completed', 'failed'].map(suffix => {
                    const event = `task.${suffix}`;
                    const isChecked = (notif.events || []).includes(event);
                    return (
                      <button
                        key={event}
                        onClick={() => {
                          const events = new Set(notif.events || []);
                          if (isChecked) events.delete(event);
                          else events.add(event);

                          const newNotifs = [...(localData.notifications as any[])];
                          newNotifs[index] = { ...notif, events: Array.from(events) };
                          handleChange('notifications', newNotifs);
                        }}
                        className={`text-[10px] px-1.5 py-0.5 rounded border transition-colors ${isChecked
                          ? 'border-blue-500/50 text-blue-400 bg-blue-500/10'
                          : 'border-white/10 text-gray-500 hover:border-white/30'
                          }`}
                      >
                        {suffix}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}

            {(localData.notifications as any[] || []).length === 0 && (
              <div className="text-center p-4 border border-white/5 rounded dashed border-dashed text-xs text-gray-600">
                No notifications configured
              </div>
            )}
          </div>
        </div>


      </div >
    </div >
  );
};
