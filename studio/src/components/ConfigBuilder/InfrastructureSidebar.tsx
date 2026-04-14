import React, { useState, useMemo, useCallback } from 'react';
import { Cpu, Box, Layers, Wrench, Plus, Pencil, Trash2, Shield, FileText } from 'lucide-react';
import type { Node } from '@xyflow/react';

import { parseConfig, updateConfigSection, removeFromConfigSection } from '../../lib/config-utils';
import type { LLMConfig, ToolConfig, GuardrailConfig, EmbedderConfig, VectorStoreConfig, DocumentStoreConfig } from '../../lib/config-utils';
import { LLMModal } from './LLMModal';
import { ToolModal } from './ToolModal';
import { GuardrailModal } from './GuardrailModal';
import { EmbedderModal, VectorStoreModal, DocumentStoreModal } from './OtherModals';
import { useStore } from '../../store/useStore';



interface InfrastructureSidebarProps {
  yamlContent: string;
  nodes: Node[];
  collapsed?: boolean;
  onToggle?: () => void;
}

interface SectionItemProps {
  name: string;
  config: any;
  color: string;
  usageCount?: number;
  onEdit: () => void;
  onDelete: () => void;
}

const SectionItem: React.FC<SectionItemProps> = ({ name, config, color, usageCount, onEdit, onDelete }) => (
  <div className="p-2 rounded border border-white/5 bg-white/5 hover:bg-white/10 transition-colors group">
    <div className="flex items-center justify-between mb-1">
      <span className={`font-heading font-medium text-xs ${color}`}>{name}</span>
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        {usageCount !== undefined && usageCount > 0 && (
          <span className="text-[10px] bg-white/10 px-1.5 py-0.5 rounded text-gray-400 mr-1">
            {usageCount} refs
          </span>
        )}
        <button
          onClick={(e) => { e.stopPropagation(); onEdit(); }}
          className="p-1 hover:bg-white/10 rounded"
          title="Edit"
        >
          <Pencil size={12} className="text-gray-400 hover:text-white" />
        </button>
        <button
          onClick={(e) => { e.stopPropagation(); onDelete(); }}
          className="p-1 hover:bg-red-500/20 rounded"
          title="Delete"
        >
          <Trash2 size={12} className="text-gray-400 hover:text-red-400" />
        </button>
      </div>
    </div>
    <div className="flex items-center gap-1.5 text-[10px] text-gray-400">
      {config.provider && (
        <span className="px-1 py-px rounded bg-white/5 border border-white/5">{config.provider}</span>
      )}
      {config.model && (
        <span className="truncate max-w-[120px]" title={config.model}>{config.model}</span>
      )}
      {config.type && (
        <span className="px-1 py-px rounded bg-white/5 border border-white/5 capitalize">{config.type}</span>
      )}
      {config.driver && (
        <span className="uppercase tracking-wider">{config.driver}</span>
      )}
      {/* Guardrail specific tags */}
      {config.input && (
        <span className="px-1 py-px rounded bg-white/5 border border-white/5 text-gray-500">Input</span>
      )}
      {config.output && (
        <span className="px-1 py-px rounded bg-white/5 border border-white/5 text-gray-500">Output</span>
      )}
      {config.tool?.authorization && (
        <span className="px-1 py-px rounded bg-white/5 border border-white/5 text-gray-500">Auth</span>
      )}
      {config.moderation?.enabled && (
        <span className="px-1 py-px rounded bg-white/5 border border-white/5 text-gray-500">Mod</span>
      )}
    </div>
  </div>
);

interface SectionHeaderProps {
  icon: React.ReactNode;
  iconBgClass: string;
  title: string;
  count: number;
  isExpanded: boolean;
  dotColor: string;
  onToggle: () => void;
  onAdd: () => void;
}

const SectionHeader: React.FC<SectionHeaderProps> = ({
  icon, iconBgClass, title, count, isExpanded, dotColor, onToggle, onAdd
}) => (
  <div className="w-full px-3 py-2 flex items-center justify-between hover:bg-white/5 transition-colors group">
    <button onClick={onToggle} className="flex items-center gap-3 flex-1">
      <div className={`w-8 h-8 rounded ${iconBgClass} flex items-center justify-center group-hover:opacity-80 transition-opacity`}>
        {icon}
      </div>
      <div className="text-left">
        <div className="text-sm font-medium text-gray-200">{title}</div>
        <div className="text-[10px] text-gray-500">{count} configured</div>
      </div>
    </button>
    <div className="flex items-center gap-2">
      <button
        onClick={(e) => { e.stopPropagation(); onAdd(); }}
        className="p-1 hover:bg-white/10 rounded opacity-0 group-hover:opacity-100 transition-opacity"
        title={`Add ${title.slice(0, -1)}`}
      >
        <Plus size={14} className="text-gray-400 hover:text-white" />
      </button>
      {isExpanded && (
        <div className={`w-1.5 h-1.5 rounded-full ${dotColor}`} />
      )}
    </div>
  </div>
);

export const InfrastructureSidebar: React.FC<InfrastructureSidebarProps> = React.memo(({
  yamlContent,
  nodes,
  collapsed = false,
  onToggle
}) => {
  const setYamlContent = useStore((s) => s.setStudioYamlContent);

  const [expandedSection, setExpandedSection] = useState<string | null>('llms');

  // Modal states
  const [llmModalOpen, setLlmModalOpen] = useState(false);
  const [toolModalOpen, setToolModalOpen] = useState(false);
  const [guardrailModalOpen, setGuardrailModalOpen] = useState(false);
  const [embedderModalOpen, setEmbedderModalOpen] = useState(false);
  const [vectorStoreModalOpen, setVectorStoreModalOpen] = useState(false);
  const [documentStoreModalOpen, setDocumentStoreModalOpen] = useState(false);

  const [editingLLM, setEditingLLM] = useState<{ id: string; config: LLMConfig } | null>(null);
  const [editingTool, setEditingTool] = useState<{ id: string; config: ToolConfig } | null>(null);
  const [editingGuardrail, setEditingGuardrail] = useState<{ id: string; config: GuardrailConfig } | null>(null);
  const [editingEmbedder, setEditingEmbedder] = useState<{ id: string; config: EmbedderConfig } | null>(null);
  const [editingVectorStore, setEditingVectorStore] = useState<{ id: string; config: VectorStoreConfig } | null>(null);
  const [editingDocumentStore, setEditingDocumentStore] = useState<{ id: string; config: DocumentStoreConfig } | null>(null);

  const infrastructure = useMemo(() => {
    try {
      const config = parseConfig(yamlContent);
      const llms = config.llms || {};
      const embedders = config.embedders || {};
      const vectorStores = config.vector_stores || {};
      const tools = config.tools || {};
      const guardrails = config.guardrails || {};
      const documentStores = config.document_stores || {};

      // Calculate LLM usage from nodes
      const llmUsage: Record<string, number> = {};
      nodes.forEach((node) => {
        if (node.data && typeof node.data === 'object' && 'llm' in node.data) {
          const llmName = (node.data as any).llm;
          if (llmName) {
            llmUsage[llmName] = (llmUsage[llmName] || 0) + 1;
          }
        }
      });

      return {
        llms,
        embedders,
        vectorStores,
        tools,
        guardrails,
        documentStores,
        llmUsage,
      };
    } catch (e) {
      return {
        llms: {},
        embedders: {},
        vectorStores: {},
        tools: {},
        guardrails: {},
        documentStores: {},
        llmUsage: {},
      };
    }
  }, [yamlContent, nodes]);

  // Handlers for LLMs
  const handleAddLLM = useCallback(() => {
    setEditingLLM(null);
    setLlmModalOpen(true);
  }, []);

  const handleEditLLM = useCallback((id: string, config: LLMConfig) => {
    setEditingLLM({ id, config });
    setLlmModalOpen(true);
  }, []);

  const handleDeleteLLM = useCallback((id: string) => {
    const newYaml = removeFromConfigSection(yamlContent, 'llms', id);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  const handleSaveLLM = useCallback((id: string, config: LLMConfig) => {
    const newYaml = updateConfigSection(yamlContent, 'llms', id, config);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  // Handlers for Tools
  const handleAddTool = useCallback(() => {
    setEditingTool(null);
    setToolModalOpen(true);
  }, []);

  const handleEditTool = useCallback((id: string, config: ToolConfig) => {
    setEditingTool({ id, config });
    setToolModalOpen(true);
  }, []);

  const handleDeleteTool = useCallback((id: string) => {
    const newYaml = removeFromConfigSection(yamlContent, 'tools', id);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  const handleSaveTool = useCallback((id: string, config: ToolConfig) => {
    const newYaml = updateConfigSection(yamlContent, 'tools', id, config);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  // Handlers for Guardrails
  const handleAddGuardrail = useCallback(() => {
    setEditingGuardrail(null);
    setGuardrailModalOpen(true);
  }, []);

  const handleEditGuardrail = useCallback((id: string, config: GuardrailConfig) => {
    setEditingGuardrail({ id, config });
    setGuardrailModalOpen(true);
  }, []);

  const handleDeleteGuardrail = useCallback((id: string) => {
    const newYaml = removeFromConfigSection(yamlContent, 'guardrails', id);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  const handleSaveGuardrail = useCallback((id: string, config: GuardrailConfig) => {
    const newYaml = updateConfigSection(yamlContent, 'guardrails', id, config);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  // Handlers for Embedders
  const handleAddEmbedder = useCallback(() => {
    setEditingEmbedder(null);
    setEmbedderModalOpen(true);
  }, []);

  const handleEditEmbedder = useCallback((id: string, config: EmbedderConfig) => {
    setEditingEmbedder({ id, config });
    setEmbedderModalOpen(true);
  }, []);

  const handleDeleteEmbedder = useCallback((id: string) => {
    const newYaml = removeFromConfigSection(yamlContent, 'embedders', id);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  const handleSaveEmbedder = useCallback((id: string, config: EmbedderConfig) => {
    const newYaml = updateConfigSection(yamlContent, 'embedders', id, config);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  // Handlers for Vector Stores
  const handleAddVectorStore = useCallback(() => {
    setEditingVectorStore(null);
    setVectorStoreModalOpen(true);
  }, []);

  const handleEditVectorStore = useCallback((id: string, config: VectorStoreConfig) => {
    setEditingVectorStore({ id, config });
    setVectorStoreModalOpen(true);
  }, []);

  const handleDeleteVectorStore = useCallback((id: string) => {
    const newYaml = removeFromConfigSection(yamlContent, 'vector_stores', id);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  const handleSaveVectorStore = useCallback((id: string, config: VectorStoreConfig) => {
    const newYaml = updateConfigSection(yamlContent, 'vector_stores', id, config);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  // Handlers for Document Stores
  const handleAddDocumentStore = useCallback(() => {
    setEditingDocumentStore(null);
    setDocumentStoreModalOpen(true);
  }, []);

  const handleEditDocumentStore = useCallback((id: string, config: DocumentStoreConfig) => {
    setEditingDocumentStore({ id, config });
    setDocumentStoreModalOpen(true);
  }, []);

  const handleDeleteDocumentStore = useCallback((id: string) => {
    const newYaml = removeFromConfigSection(yamlContent, 'document_stores', id);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  const handleSaveDocumentStore = useCallback((id: string, config: DocumentStoreConfig) => {
    const newYaml = updateConfigSection(yamlContent, 'document_stores', id, config);
    setYamlContent(newYaml);
  }, [yamlContent, setYamlContent]);

  const SECTIONS = useMemo(() => [
    {
      key: 'llms' as const,
      title: 'LLMs',
      icon: Cpu,
      color: 'text-blue-400',
      iconBg: 'bg-blue-500/10 group-hover:bg-blue-500/20',
      dotColor: 'bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.8)]',
      itemColor: 'text-blue-300',
      add: handleAddLLM,
      edit: handleEditLLM,
      delete: handleDeleteLLM,
    },
    {
      key: 'tools' as const,
      title: 'Tools',
      icon: Wrench,
      color: 'text-yellow-400',
      iconBg: 'bg-yellow-500/10 group-hover:bg-yellow-500/20',
      dotColor: 'bg-yellow-500 shadow-[0_0_8px_rgba(234,179,8,0.8)]',
      itemColor: 'text-yellow-300',
      add: handleAddTool,
      edit: handleEditTool,
      delete: handleDeleteTool,
    },
    {
      key: 'guardrails' as const,
      title: 'Guardrails',
      icon: Shield,
      color: 'text-red-400',
      iconBg: 'bg-red-500/10 group-hover:bg-red-500/20',
      dotColor: 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.8)]',
      itemColor: 'text-red-300',
      add: handleAddGuardrail,
      edit: handleEditGuardrail,
      delete: handleDeleteGuardrail,
    },
    {
      key: 'embedders' as const,
      title: 'Embedders',
      icon: Layers,
      color: 'text-purple-400',
      iconBg: 'bg-purple-500/10 group-hover:bg-purple-500/20',
      dotColor: 'bg-purple-500 shadow-[0_0_8px_rgba(168,85,247,0.8)]',
      itemColor: 'text-purple-300',
      add: handleAddEmbedder,
      edit: handleEditEmbedder,
      delete: handleDeleteEmbedder,
    },
    {
      key: 'vectorStores' as const,
      title: 'Vector Stores',
      icon: Box,
      color: 'text-orange-400',
      iconBg: 'bg-orange-500/10 group-hover:bg-orange-500/20',
      dotColor: 'bg-orange-500 shadow-[0_0_8px_rgba(249,115,22,0.8)]',
      itemColor: 'text-orange-300',
      add: handleAddVectorStore,
      edit: handleEditVectorStore,
      delete: handleDeleteVectorStore,
    },
    {
      key: 'documentStores' as const,
      title: 'Document Stores',
      icon: FileText,
      color: 'text-cyan-400',
      iconBg: 'bg-cyan-500/10 group-hover:bg-cyan-500/20',
      dotColor: 'bg-cyan-500 shadow-[0_0_8px_rgba(6,182,212,0.8)]',
      itemColor: 'text-cyan-300',
      add: handleAddDocumentStore,
      edit: handleEditDocumentStore,
      delete: handleDeleteDocumentStore,
    },
  ], [
    handleAddLLM, handleEditLLM, handleDeleteLLM,
    handleAddTool, handleEditTool, handleDeleteTool,
    handleAddGuardrail, handleEditGuardrail, handleDeleteGuardrail,
    handleAddEmbedder, handleEditEmbedder, handleDeleteEmbedder,
    handleAddVectorStore, handleEditVectorStore, handleDeleteVectorStore,
    handleAddEmbedder, handleEditEmbedder, handleDeleteEmbedder,
    handleAddVectorStore, handleEditVectorStore, handleDeleteVectorStore,
    handleAddDocumentStore, handleEditDocumentStore, handleDeleteDocumentStore
  ]);



  // Determine sidebar content based on collapsed state
  const sidebarContent = collapsed ? (
    <div className="w-16 bg-black/40 border-r border-white/10 flex flex-col items-center">
      {/* Header / Toggle */}
      <div className="flex-none w-full h-10 flex items-center justify-center border-b border-white/10">
        <button onClick={onToggle} className="p-1 text-gray-400 hover:text-white transition-colors" title="Expand sidebar">
          <Layers size={16} />
        </button>
      </div>

      {/* Resources List */}
      <div className="flex-1 w-full overflow-y-auto custom-scrollbar flex flex-col items-center py-2 gap-2">
        {SECTIONS.map((section) => (
          <button
            key={section.key}
            onClick={() => {
              setExpandedSection(section.key);
              if (onToggle) onToggle();
            }}
            className="p-2 hover:bg-white/5 rounded transition-colors relative group"
            title={`${section.title} (${Object.keys(infrastructure[section.key]).length})`}
          >
            <section.icon size={20} className={section.color} />
            <span className="absolute -top-1 -right-1 bg-black/80 text-[10px] text-gray-400 px-1 rounded-full border border-white/10 opacity-0 group-hover:opacity-100 transition-opacity">
              {Object.keys(infrastructure[section.key]).length}
            </span>
          </button>
        ))}
      </div>

      {/* Global Config - Bottom */}

    </div>
  ) : (
    <div className="w-64 bg-black/40 border-r border-white/10 flex flex-col overflow-hidden">
      <div className="flex-none h-10 px-3 border-b border-white/10 flex items-center justify-between">
        <h2 className="text-sm font-semibold">Resources</h2>
        <div className="flex items-center gap-1">
          <button onClick={onToggle} className="p-1 text-gray-400 hover:text-white transition-colors" title="Collapse sidebar">
            <Layers size={16} className="rotate-90" />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto custom-scrollbar">
        {SECTIONS.map((section) => (
          <div key={section.key} className="border-b border-white/10">
            <SectionHeader
              icon={<section.icon size={16} className={section.color} />}
              iconBgClass={section.iconBg}
              title={section.title}
              count={Object.keys(infrastructure[section.key]).length}
              isExpanded={expandedSection === section.key}
              dotColor={section.dotColor}
              onToggle={() => setExpandedSection(expandedSection === section.key ? null : section.key)}
              onAdd={section.add}
            />
            {expandedSection === section.key && (
              <div className="px-4 pb-3 space-y-2 bg-black/20 border-t border-white/5 pt-2">
                {Object.entries(infrastructure[section.key]).map(([name, config]) => (
                  <SectionItem
                    key={name}
                    name={name}
                    config={config}
                    color={section.itemColor}
                    usageCount={section.key === 'llms' ? infrastructure.llmUsage[name] : undefined}
                    onEdit={() => section.edit(name, config as any)}
                    onDelete={() => section.delete(name)}
                  />
                ))}
                {Object.keys(infrastructure[section.key]).length === 0 && (
                  <div className="text-xs text-center py-2 text-gray-600 italic">No {section.title.toLowerCase()} defined</div>
                )}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Global Settings - Bottom */}

    </div>
  );

  return (
    <>
      {sidebarContent}

      {/* Modals */}
      <LLMModal
        isOpen={llmModalOpen}
        onClose={() => { setLlmModalOpen(false); setEditingLLM(null); }}
        onSave={handleSaveLLM}
        existingIds={Object.keys(infrastructure.llms)}
        editId={editingLLM?.id}
        editConfig={editingLLM?.config}
      />

      <ToolModal
        isOpen={toolModalOpen}
        onClose={() => { setToolModalOpen(false); setEditingTool(null); }}
        onSave={handleSaveTool}
        existingIds={Object.keys(infrastructure.tools)}
        editId={editingTool?.id}
        editConfig={editingTool?.config}
      />

      <GuardrailModal
        isOpen={guardrailModalOpen}
        onClose={() => { setGuardrailModalOpen(false); setEditingGuardrail(null); }}
        onSave={handleSaveGuardrail}
        existingIds={Object.keys(infrastructure.guardrails)}
        editId={editingGuardrail?.id}
        editConfig={editingGuardrail?.config}
      />

      <EmbedderModal
        isOpen={embedderModalOpen}
        onClose={() => { setEmbedderModalOpen(false); setEditingEmbedder(null); }}
        onSave={handleSaveEmbedder}
        existingIds={Object.keys(infrastructure.embedders)}
        editId={editingEmbedder?.id}
        editConfig={editingEmbedder?.config}
      />

      <VectorStoreModal
        isOpen={vectorStoreModalOpen}
        onClose={() => { setVectorStoreModalOpen(false); setEditingVectorStore(null); }}
        onSave={handleSaveVectorStore}
        existingIds={Object.keys(infrastructure.vectorStores)}
        editId={editingVectorStore?.id}
        editConfig={editingVectorStore?.config}
      />

      <DocumentStoreModal
        isOpen={documentStoreModalOpen}
        onClose={() => { setDocumentStoreModalOpen(false); setEditingDocumentStore(null); }}
        onSave={handleSaveDocumentStore}
        existingIds={Object.keys(infrastructure.documentStores)}
        embedderOptions={Object.keys(infrastructure.embedders)}
        vectorStoreOptions={Object.keys(infrastructure.vectorStores)}
        editId={editingDocumentStore?.id}
        editConfig={editingDocumentStore?.config}
      />
    </>
  );
});
