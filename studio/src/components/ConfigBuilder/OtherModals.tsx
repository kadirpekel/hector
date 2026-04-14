import React, { useState, useEffect } from 'react';
import { ResourceModal, FormField, TextInput, SelectInput, ComboInput, ToggleInput } from './ResourceModal';
import type { EmbedderConfig, VectorStoreConfig, DocumentStoreConfig } from '../../lib/config-utils';


// Embedding Providers - only these are supported by Hector backend
const EMBEDDING_PROVIDERS = [
    { value: 'openai', label: 'OpenAI' },
    { value: 'ollama', label: 'Ollama' },
    { value: 'cohere', label: 'Cohere' },
];

// Popular embedding models per provider - users can also type custom model names
const EMBEDDING_PROVIDER_MODELS: Record<string, Array<{ value: string; label: string }>> = {
    openai: [
        { value: 'text-embedding-3-small', label: 'text-embedding-3-small (1536d)' },
        { value: 'text-embedding-3-large', label: 'text-embedding-3-large (3072d)' },
        { value: 'text-embedding-ada-002', label: 'text-embedding-ada-002 (legacy)' },
    ],
    ollama: [
        { value: 'nomic-embed-text', label: 'nomic-embed-text (768d)' },
        { value: 'nomic-embed-text-v1.5', label: 'nomic-embed-text-v1.5' },
        { value: 'mxbai-embed-large', label: 'mxbai-embed-large (1024d)' },
        { value: 'all-minilm:l6-v2', label: 'all-minilm:l6-v2 (384d)' },
        { value: 'snowflake-arctic-embed', label: 'snowflake-arctic-embed' },
        { value: 'bge-m3', label: 'bge-m3 (multilingual)' },
    ],
    cohere: [
        { value: 'embed-v4.0', label: 'embed-v4.0 (multimodal, 1536d)' },
        { value: 'embed-english-v3.0', label: 'embed-english-v3.0 (1024d)' },
        { value: 'embed-multilingual-v3.0', label: 'embed-multilingual-v3.0 (1024d)' },
        { value: 'embed-english-light-v3.0', label: 'embed-english-light-v3.0 (384d)' },
        { value: 'embed-multilingual-light-v3.0', label: 'embed-multilingual-light-v3.0 (384d)' },
    ],
};

// Embedder Modal
interface EmbedderModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (id: string, config: EmbedderConfig) => void;
    existingIds: string[];
    editId?: string | null;
    editConfig?: EmbedderConfig | null;
}

export const EmbedderModal: React.FC<EmbedderModalProps> = ({
    isOpen,
    onClose,
    onSave,
    existingIds,
    editId,
    editConfig,
}) => {
    const isEditing = !!editId;

    const [id, setId] = useState('');
    const [provider, setProvider] = useState('openai');
    const [model, setModel] = useState('');
    const [apiKey, setApiKey] = useState('');
    const [baseUrl, setBaseUrl] = useState('');
    const [dimension, setDimension] = useState('');
    const [timeout, setTimeout] = useState('');
    const [batchSize, setBatchSize] = useState('');
    const [encodingFormat, setEncodingFormat] = useState('float');

    useEffect(() => {
        if (isOpen) {
            if (isEditing && editConfig) {
                setId(editId);
                setProvider(editConfig.provider || 'openai');
                setModel(editConfig.model || '');
                setApiKey(editConfig.api_key || '');
                setBaseUrl(editConfig.base_url || '');
                setDimension(editConfig.dimension?.toString() || '');
                setTimeout(editConfig.timeout?.toString() || '');
                setBatchSize(editConfig.batch_size?.toString() || '');
                setEncodingFormat(editConfig.encoding_format || 'float');
            } else {
                setId('');
                setProvider('openai');
                setModel('');
                setApiKey('');
                setBaseUrl('');
                setDimension('');
                setTimeout('');
                setBatchSize('');
                setEncodingFormat('float');
            }
        }
    }, [isOpen, isEditing, editId, editConfig]);

    const handleSave = () => {
        onSave(id, {
            provider,
            model: model || undefined,
            api_key: apiKey || undefined,
            base_url: baseUrl || undefined,
            dimension: dimension ? parseInt(dimension) : undefined,
            timeout: timeout ? parseInt(timeout) : undefined,
            batch_size: batchSize ? parseInt(batchSize) : undefined,
            encoding_format: encodingFormat || undefined,
        });
        onClose();
    };

    const isValid = id && !(!isEditing && existingIds.includes(id));

    return (
        <ResourceModal
            isOpen={isOpen}
            onClose={onClose}
            title={isEditing ? `Edit Embedder: ${editId}` : 'Add Embedder'}
            onSave={handleSave}
            saveDisabled={!isValid}
        >
            <FormField label="ID" required>
                <TextInput value={id} onChange={setId} placeholder="e.g., default" />
            </FormField>

            <FormField label="Provider">
                <SelectInput
                    value={provider}
                    onChange={(v) => {
                        setProvider(v);
                        setModel(''); // Reset model when provider changes
                    }}
                    options={EMBEDDING_PROVIDERS}
                />
            </FormField>

            <FormField label="Model" hint="Select a model or type any custom model name">
                <ComboInput
                    value={model}
                    onChange={setModel}
                    options={EMBEDDING_PROVIDER_MODELS[provider] || []}
                    placeholder="e.g., text-embedding-3-small"
                />
            </FormField>

            <FormField label="API Key" hint="Use ${ENV_VAR} for env variables">
                <TextInput value={apiKey} onChange={setApiKey} placeholder="${OPENAI_API_KEY}" type="password" />
            </FormField>

            <FormField label="Base URL" hint="Custom endpoint (leave empty for default)">
                <TextInput value={baseUrl} onChange={setBaseUrl} placeholder="http://localhost:11434" />
            </FormField>

            <FormField label="Dimension" hint="Embedding dimension (leave empty for model default)">
                <TextInput value={dimension} onChange={setDimension} placeholder="e.g., 1536" type="number" />
            </FormField>

            <div className="grid grid-cols-2 gap-4">
                <FormField label="Timeout (s)">
                    <TextInput value={timeout} onChange={setTimeout} placeholder="30" type="number" />
                </FormField>
                <FormField label="Batch Size">
                    <TextInput value={batchSize} onChange={setBatchSize} placeholder="100" type="number" />
                </FormField>
            </div>

            <FormField label="Encoding Format">
                <SelectInput
                    value={encodingFormat}
                    onChange={setEncodingFormat}
                    options={[
                        { value: 'float', label: 'Float' },
                        { value: 'base64', label: 'Base64' }
                    ]}
                />
            </FormField>
        </ResourceModal>
    );
};


// Vector Store Modal
interface VectorStoreModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (id: string, config: VectorStoreConfig) => void;
    existingIds: string[];
    editId?: string | null;
    editConfig?: VectorStoreConfig | null;
}

export const VectorStoreModal: React.FC<VectorStoreModalProps> = ({
    isOpen,
    onClose,
    onSave,
    existingIds,
    editId,
    editConfig,
}) => {
    const isEditing = !!editId;

    const [id, setId] = useState('');
    const [type, setType] = useState('chromem');
    const [host, setHost] = useState('');
    const [apiKey, setApiKey] = useState('');
    const [collection, setCollection] = useState('');
    const [port, setPort] = useState('');
    const [enableTLS, setEnableTLS] = useState(false);
    const [compress, setCompress] = useState(false);
    const [indexName, setIndexName] = useState('');
    const [environment, setEnvironment] = useState('');
    const [enablePersistence, setEnablePersistence] = useState(true);
    const [persistPath, setPersistPath] = useState('');

    useEffect(() => {
        if (isOpen) {
            if (isEditing && editConfig) {
                setId(editId);
                setType(editConfig.type || 'chromem');
                setHost(editConfig.host || '');
                setApiKey(editConfig.api_key || '');
                setCollection(editConfig.collection || '');
                setPort(editConfig.port?.toString() || '');
                setEnableTLS(editConfig.enable_tls || false);
                setCompress(editConfig.compress || false);
                setIndexName(editConfig.index_name || '');
                setEnvironment(editConfig.environment || '');
                setEnablePersistence(editConfig.enable_persistence !== false);
                setPersistPath(editConfig.persist_path || '');
            } else {
                setId('');
                setType('chromem');
                setHost('');
                setApiKey('');
                setCollection('');
                setPort('');
                setEnableTLS(false);
                setCompress(false);
                setIndexName('');
                setEnvironment('');
                setEnablePersistence(true);
                setPersistPath('');
            }
        }
    }, [isOpen, isEditing, editId, editConfig]);

    const handleSave = () => {
        const config: VectorStoreConfig = {
            type,
            host: host || undefined,
            api_key: apiKey || undefined,
            collection: collection || undefined,
            port: port ? parseInt(port) : undefined,
            enable_tls: enableTLS || undefined,
            compress: compress || undefined,
            index_name: indexName || undefined,
            environment: environment || undefined,
        };

        // Only include chromem-specific fields for chromem type
        if (type === 'chromem') {
            config.enable_persistence = enablePersistence;
            if (enablePersistence && persistPath) {
                config.persist_path = persistPath;
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
            title={isEditing ? `Edit Vector Store: ${editId}` : 'Add Vector Store'}
            onSave={handleSave}
            saveDisabled={!isValid}
        >
            <FormField label="ID" required>
                <TextInput value={id} onChange={setId} placeholder="e.g., default" />
            </FormField>

            <FormField label="Type">
                <SelectInput
                    value={type}
                    onChange={setType}
                    options={[
                        { value: 'chromem', label: 'Chromem (Embedded, Persistent by Default)' },
                        { value: 'qdrant', label: 'Qdrant' },
                        { value: 'pinecone', label: 'Pinecone' },
                        { value: 'milvus', label: 'Milvus' },
                        { value: 'weaviate', label: 'Weaviate' },
                    ]}
                />
            </FormField>

            {type === 'chromem' && (
                <>
                    <ToggleInput
                        checked={enablePersistence}
                        onChange={setEnablePersistence}
                        label="Enable Persistence"
                    />
                    {enablePersistence && (
                        <FormField label="Persist Path" hint="Default: .hector/vectors">
                            <TextInput
                                value={persistPath}
                                onChange={setPersistPath}
                                placeholder=".hector/vectors"
                            />
                        </FormField>
                    )}
                </>
            )}

            {type !== 'chromem' && (
                <>
                    <div className="grid grid-cols-2 gap-4">
                        <FormField label="Host" hint="Server URL or Hostname">
                            <TextInput value={host} onChange={setHost} placeholder="localhost" />
                        </FormField>
                        <FormField label="Port">
                            <TextInput value={port} onChange={setPort} placeholder="6333" type="number" />
                        </FormField>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <ToggleInput checked={enableTLS} onChange={setEnableTLS} label="Enable TLS" />
                        <ToggleInput checked={compress} onChange={setCompress} label="Compress" />
                    </div>

                    <FormField label="API Key" hint="Use ${ENV_VAR} for secrets">
                        <TextInput value={apiKey} onChange={setApiKey} placeholder="${VECTOR_STORE_API_KEY}" type="password" />
                    </FormField>

                    {type === 'pinecone' && (
                        <div className="grid grid-cols-2 gap-4">
                            <FormField label="Environment">
                                <TextInput value={environment} onChange={setEnvironment} placeholder="us-west1-gcp" />
                            </FormField>
                            <FormField label="Index Name">
                                <TextInput value={indexName} onChange={setIndexName} placeholder="my-index" />
                            </FormField>
                        </div>
                    )}
                </>
            )}

            <FormField label="Collection">
                <TextInput value={collection} onChange={setCollection} placeholder="documents" />
            </FormField>
        </ResourceModal>
    );
};

// Document Store Modal (simplified)
interface DocumentStoreModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (id: string, config: DocumentStoreConfig) => void;
    existingIds: string[];
    embedderOptions: string[];
    vectorStoreOptions: string[];
    editId?: string | null;
    editConfig?: DocumentStoreConfig | null;
}


export const DocumentStoreModal: React.FC<DocumentStoreModalProps> = ({
    isOpen,
    onClose,
    onSave,
    existingIds,
    embedderOptions,
    vectorStoreOptions,
    editId,
    editConfig,
}) => {
    const isEditing = !!editId;

    const [id, setId] = useState('');
    const [embedder, setEmbedder] = useState('');
    const [vectorStore, setVectorStore] = useState('');

    // Source Type
    const [sourceType, setSourceType] = useState<'blob' | 'sql' | 'api'>('blob');

    // SQL Source
    const [sqlDSN, setSqlDSN] = useState('');
    const [sqlTables, setSqlTables] = useState<Array<{ table: string; id_column: string; columns: string }>>([]); // columns as comma-separated string for UI

    // API Source
    const [apiUrl, setApiUrl] = useState('');
    const [apiIdField, setApiIdField] = useState('');
    const [apiContentField, setApiContentField] = useState('');
    const [apiHeaders, setApiHeaders] = useState<Array<{ key: string; value: string }>>([]);

    // Blob Source
    const [blobUrl, setBlobUrl] = useState('');
    const [blobPrefix, setBlobPrefix] = useState('');
    const [blobProvider, setBlobProvider] = useState<'file' | 's3' | 'gcs' | 'azure'>('file');

    const [chunkSize, setChunkSize] = useState('1000');
    const [watch, setWatch] = useState(true);
    const [incrementalIndexing, setIncrementalIndexing] = useState(true);

    useEffect(() => {
        if (isOpen) {
            if (isEditing && editConfig) {
                setId(editId);
                setEmbedder(editConfig.embedder || '');
                setVectorStore(editConfig.vector_store || '');
                setChunkSize(editConfig.chunking?.size?.toString() || '1000');
                setWatch(editConfig.watch !== false);
                setIncrementalIndexing(editConfig.incremental_indexing !== false);

                // Determine source type
                if (editConfig.source?.sql) {
                    setSourceType('sql');
                    setSqlDSN(editConfig.source.sql.dsn || '');
                    setSqlTables((editConfig.source.sql.tables || []).map(t => ({
                        table: t.table,
                        id_column: t.id_column,
                        columns: t.columns.join(', ')
                    })));
                } else if (editConfig.source?.api) {
                    setSourceType('api');
                    setApiUrl(editConfig.source.api.url || '');
                    setApiIdField(editConfig.source.api.id_field || '');
                    setApiContentField(editConfig.source.api.content_field || '');
                    setApiHeaders(Object.entries(editConfig.source.api.headers || {})
                        .map(([key, value]) => ({ key, value })));
                    setSqlDSN('');
                    setSqlTables([]);
                } else if (editConfig.source?.blob) {
                    setSourceType('blob');
                    const url = editConfig.source.blob.url || '';
                    setBlobUrl(url);
                    setBlobPrefix(editConfig.source.blob.prefix || '');
                    // Detect provider from URL
                    if (url.startsWith('file://')) setBlobProvider('file');
                    else if (url.startsWith('s3://')) setBlobProvider('s3');
                    else if (url.startsWith('gs://')) setBlobProvider('gcs');
                    else if (url.startsWith('azblob://')) setBlobProvider('azure');
                    else setBlobProvider('file');
                    setSqlDSN('');
                    setSqlTables([]);
                } else {
                    // Default to blob
                    setSourceType('blob');
                    setBlobUrl('file://./docs');
                    setBlobPrefix('');
                    setBlobProvider('file');
                    setSqlDSN('');
                    setSqlTables([]);
                }
            } else {
                setId('');
                setEmbedder(embedderOptions[0] || '');
                setVectorStore(vectorStoreOptions[0] || '');
                setSourceType('blob');
                setBlobUrl('file://./docs');
                setBlobPrefix('');
                setBlobProvider('file');
                setSqlDSN('');
                setSqlTables([]);
                setApiUrl('');
                setApiIdField('');
                setApiContentField('');
                setApiHeaders([]);
                setChunkSize('1000');
                setWatch(true);
                setIncrementalIndexing(true);
            }
        }
    }, [isOpen, isEditing, editId, editConfig, embedderOptions, vectorStoreOptions]);

    const handleSave = () => {
        const config: DocumentStoreConfig = {
            embedder: embedder || undefined,
            vector_store: vectorStore || undefined,
            watch,
            incremental_indexing: incrementalIndexing,
        };

        if (sourceType === 'sql') {
            if (sqlTables.length > 0) {
                config.source = {
                    type: 'sql',
                    sql: {
                        dsn: sqlDSN || undefined,
                        tables: sqlTables.map(t => ({
                            table: t.table,
                            id_column: t.id_column,
                            columns: t.columns.split(',').map(s => s.trim()).filter(Boolean)
                        }))
                    }
                };
            }
        } else if (sourceType === 'api') {
            if (apiUrl && apiIdField && apiContentField) {
                config.source = {
                    type: 'api',
                    api: {
                        url: apiUrl,
                        id_field: apiIdField,
                        content_field: apiContentField,
                        headers: Object.fromEntries(
                            apiHeaders.filter(h => h.key && h.value)
                                .map(h => [h.key, h.value])
                        )
                    }
                };
            }
        } else if (sourceType === 'blob') {
            if (blobUrl) {
                config.source = {
                    type: 'blob',
                    blob: {
                        url: blobUrl,
                        prefix: blobPrefix || undefined,
                    }
                };
            }
        }

        config.chunking = {
            size: parseInt(chunkSize) || 1000,
            overlap: 200,
            strategy: 'overlapping',
        };

        onSave(id, config);
        onClose();
    };

    const isValid = id && !(!isEditing && existingIds.includes(id)) && (
        (sourceType === 'sql' && sqlTables.length > 0) ||
        (sourceType === 'api' && apiUrl && apiIdField && apiContentField) ||
        (sourceType === 'blob' && blobUrl)
    );

    return (
        <ResourceModal
            isOpen={isOpen}
            onClose={onClose}
            title={isEditing ? `Edit Document Store: ${editId}` : 'Add Document Store'}
            onSave={handleSave}
            saveDisabled={!isValid}
        >
            <FormField label="ID" required>
                <TextInput value={id} onChange={setId} placeholder="e.g., docs, knowledge" />
            </FormField>

            <FormField label="Embedder">
                <SelectInput
                    value={embedder}
                    onChange={setEmbedder}
                    options={embedderOptions.map(e => ({ value: e, label: e }))}
                    placeholder="Select embedder..."
                />
            </FormField>

            <FormField label="Vector Store">
                <SelectInput
                    value={vectorStore}
                    onChange={setVectorStore}
                    options={vectorStoreOptions.map(v => ({ value: v, label: v }))}
                    placeholder="Select vector store..."
                />
            </FormField>

            <div className="pt-4 border-t border-white/10 space-y-4">
                <FormField label="Source Type">
                    <SelectInput
                        value={sourceType}
                        onChange={(v) => setSourceType(v as 'blob' | 'sql' | 'api')}
                        options={[
                            { value: 'blob', label: 'Blob Storage (Local & Cloud)' },
                            { value: 'sql', label: 'SQL Database' },
                            { value: 'api', label: 'REST API' },
                        ]}
                    />
                </FormField>

                {sourceType === 'sql' && (
                    <div className="space-y-4 pl-2 border-l-2 border-white/10">
                        <FormField label="DSN (Optional)" hint="Using primary DB if empty">
                            <TextInput value={sqlDSN} onChange={setSqlDSN} placeholder="sqlite://.hector/hector.db" />
                        </FormField>

                        <div className="space-y-2">
                            <div className="flex justify-between items-center text-xs uppercase text-gray-400 font-medium">
                                <span>Tables</span>
                                <button
                                    type="button"
                                    onClick={() => setSqlTables([...sqlTables, { table: '', id_column: 'id', columns: '*' }])}
                                    className="text-hector-green hover:underline"
                                >
                                    + Add Table
                                </button>
                            </div>
                            {sqlTables.length === 0 && <div className="text-xs text-gray-500 italic">No tables configured.</div>}
                            {sqlTables.map((t, idx) => (
                                <div key={idx} className="bg-white/5 p-2 rounded relative group">
                                    <button
                                        onClick={() => setSqlTables(sqlTables.filter((_, i) => i !== idx))}
                                        className="absolute top-1 right-1 text-gray-500 hover:text-red-400 opacity-0 group-hover:opacity-100"
                                    >
                                        remove
                                    </button>
                                    <div className="grid gap-2">
                                        <FormField label="Table Name">
                                            <TextInput
                                                value={t.table}
                                                onChange={(v) => {
                                                    const newTables = [...sqlTables];
                                                    newTables[idx].table = v;
                                                    setSqlTables(newTables);
                                                }}
                                                placeholder="users"
                                            />
                                        </FormField>
                                        <div className="grid grid-cols-2 gap-2">
                                            <FormField label="ID Column">
                                                <TextInput
                                                    value={t.id_column}
                                                    onChange={(v) => {
                                                        const newTables = [...sqlTables];
                                                        newTables[idx].id_column = v;
                                                        setSqlTables(newTables);
                                                    }}
                                                    placeholder="id"
                                                />
                                            </FormField>
                                            <FormField label="Columns (comma-sep)">
                                                <TextInput
                                                    value={t.columns}
                                                    onChange={(v) => {
                                                        const newTables = [...sqlTables];
                                                        newTables[idx].columns = v;
                                                        setSqlTables(newTables);
                                                    }}
                                                    placeholder="name, bio, notes"
                                                />
                                            </FormField>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}

                {sourceType === 'api' && (
                    <div className="space-y-4 pl-2 border-l-2 border-white/10">
                        <FormField label="API URL" required hint="REST API endpoint">
                            <TextInput value={apiUrl} onChange={setApiUrl} placeholder="https://api.example.com/documents" />
                        </FormField>

                        <div className="grid grid-cols-2 gap-4">
                            <FormField label="ID Field" required hint="JSON field for document ID">
                                <TextInput value={apiIdField} onChange={setApiIdField} placeholder="doc_id" />
                            </FormField>
                            <FormField label="Content Field" required hint="JSON field for content">
                                <TextInput value={apiContentField} onChange={setApiContentField} placeholder="body" />
                            </FormField>
                        </div>

                        <div className="space-y-2">
                            <div className="flex justify-between items-center text-xs uppercase text-gray-400 font-medium">
                                <span>Headers (Optional)</span>
                                <button
                                    type="button"
                                    onClick={() => setApiHeaders([...apiHeaders, { key: '', value: '' }])}
                                    className="text-hector-green hover:underline"
                                >
                                    + Add Header
                                </button>
                            </div>
                            {apiHeaders.length === 0 && <div className="text-xs text-gray-500 italic">No headers configured.</div>}
                            {apiHeaders.map((h, idx) => (
                                <div key={idx} className="bg-white/5 p-2 rounded relative group">
                                    <button
                                        onClick={() => setApiHeaders(apiHeaders.filter((_, i) => i !== idx))}
                                        className="absolute top-1 right-1 text-gray-500 hover:text-red-400 opacity-0 group-hover:opacity-100"
                                    >
                                        remove
                                    </button>
                                    <div className="grid grid-cols-2 gap-2">
                                        <FormField label="Header Name">
                                            <TextInput
                                                value={h.key}
                                                onChange={(v) => {
                                                    const newHeaders = [...apiHeaders];
                                                    newHeaders[idx].key = v;
                                                    setApiHeaders(newHeaders);
                                                }}
                                                placeholder="Authorization"
                                            />
                                        </FormField>
                                        <FormField label="Header Value">
                                            <TextInput
                                                value={h.value}
                                                onChange={(v) => {
                                                    const newHeaders = [...apiHeaders];
                                                    newHeaders[idx].value = v;
                                                    setApiHeaders(newHeaders);
                                                }}
                                                placeholder="Bearer ${API_TOKEN}"
                                                type="password"
                                            />
                                        </FormField>
                                    </div>
                                </div>
                            ))}
                        </div>

                        <div className="text-xs text-gray-400 bg-white/5 p-3 rounded">
                            💡 API will be queried and responses indexed into the vector store.
                        </div>
                    </div>
                )}

                {sourceType === 'blob' && (
                    <div className="space-y-4 pl-2 border-l-2 border-white/10">
                        <FormField label="Storage Provider" required>
                            <SelectInput
                                value={blobProvider}
                                onChange={(v) => {
                                    setBlobProvider(v as 'file' | 's3' | 'gcs' | 'azure');
                                    // Auto-update URL prefix when provider changes
                                    const currentPath = blobUrl.replace(/^(file|s3|gs|azblob):\/\//, '');
                                    if (v === 'file') setBlobUrl(`file://${currentPath || '/path/to/docs'}`);
                                    else if (v === 's3') setBlobUrl(`s3://${currentPath || 'bucket-name/prefix'}`);
                                    else if (v === 'gcs') setBlobUrl(`gs://${currentPath || 'bucket-name/prefix'}`);
                                    else if (v === 'azure') setBlobUrl(`azblob://${currentPath || 'container-name/prefix'}`);
                                }}
                                options={[
                                    { value: 'file', label: 'Local Filesystem' },
                                    { value: 's3', label: 'Amazon S3' },
                                    { value: 'gcs', label: 'Google Cloud Storage' },
                                    { value: 'azure', label: 'Azure Blob Storage' },
                                ]}
                            />
                        </FormField>

                        <FormField
                            label="Blob URL"
                            required
                            hint={
                                blobProvider === 'file' ? 'file:///absolute/path/to/docs' :
                                blobProvider === 's3' ? 's3://bucket-name/path?region=us-east-1' :
                                blobProvider === 'gcs' ? 'gs://bucket-name/path' :
                                'azblob://container-name/path'
                            }
                        >
                            <TextInput
                                value={blobUrl}
                                onChange={setBlobUrl}
                                placeholder={
                                    blobProvider === 'file' ? 'file:///Users/yourname/documents' :
                                    blobProvider === 's3' ? 's3://my-bucket/docs?region=us-east-1' :
                                    blobProvider === 'gcs' ? 'gs://my-bucket/docs' :
                                    'azblob://my-container/docs'
                                }
                            />
                        </FormField>

                        <FormField
                            label="Prefix"
                            hint="Optional: Filter to blobs with this prefix (e.g., 'company-docs/')"
                        >
                            <TextInput
                                value={blobPrefix}
                                onChange={setBlobPrefix}
                                placeholder="Optional prefix path"
                            />
                        </FormField>

                        {blobProvider !== 'file' && (
                            <div className="text-xs text-gray-400 bg-blue-500/10 border border-blue-500/20 p-3 rounded space-y-2">
                                <div className="font-semibold text-blue-400">⚠️ Environment Variables Required:</div>
                                {blobProvider === 's3' && (
                                    <div className="space-y-1 font-mono text-xs">
                                        <div>• AWS_ACCESS_KEY_ID</div>
                                        <div>• AWS_SECRET_ACCESS_KEY</div>
                                        <div>• AWS_REGION (or in URL)</div>
                                    </div>
                                )}
                                {blobProvider === 'gcs' && (
                                    <div className="space-y-1 font-mono text-xs">
                                        <div>• GOOGLE_APPLICATION_CREDENTIALS</div>
                                        <div className="text-gray-500">(path to service account JSON)</div>
                                    </div>
                                )}
                                {blobProvider === 'azure' && (
                                    <div className="space-y-1 font-mono text-xs">
                                        <div>• AZURE_STORAGE_ACCOUNT</div>
                                        <div>• AZURE_STORAGE_KEY</div>
                                    </div>
                                )}
                                <div className="text-gray-500 mt-2">
                                    Set these in your Hector agent's environment before indexing.
                                </div>
                            </div>
                        )}

                        <div className="text-xs text-gray-400 bg-white/5 p-3 rounded">
                            💡 Blob storage provides unified access to local files and cloud storage (S3, GCS, Azure).
                            Documents are discovered and indexed from the specified URL.
                        </div>
                    </div>
                )}
            </div>

            <div className="pt-4 border-t border-white/10">
                <FormField label="Chunk Size" hint="Characters per chunk">
                    <TextInput value={chunkSize} onChange={setChunkSize} placeholder="1000" type="number" />
                </FormField>

                <ToggleInput
                    checked={watch}
                    onChange={setWatch}
                    label="Watch for Changes"
                />

                <ToggleInput
                    checked={incrementalIndexing}
                    onChange={setIncrementalIndexing}
                    label="Incremental Indexing"
                />
            </div>
        </ResourceModal>
    );
};

