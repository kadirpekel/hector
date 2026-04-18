# Configuration Reference

Complete, authoritative reference for Hector's YAML configuration format. Every field documented with type, default, constraints, and description.

**Schema Version:** Generated from `/schema` endpoint  
**Default Location:** `.hector/config.yaml`

This file documents the **app config** (YAML). For server flags (port, database, auth), see the [CLI Reference](cli.md).

---

## Top-Level Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `version` | string | No | Configuration version |
| `llms` | map[string]object | No | LLM provider configurations |
| `tools` | map[string]object | No | Tool configurations |
| `agents` | map[string]object | No | Agent configurations |
| `guardrails` | map[string]object | No | Guardrail configurations |
| `vector_stores` | map[string]object | No | Vector database configurations |
| `embedders` | map[string]object | No | Embedding provider configurations |
| `document_stores` | map[string]object | No | Document source configurations |
| `defaults` | object | No | Default values |

---

## llms

LLM provider configurations. Each key is a unique LLM name.

### llms.\<name\>

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `provider` | string | _(auto-detected)_ | enum: `anthropic`, `openai`, `gemini`, `ollama`, `deepseek`, `groq`, `mistral`, `cohere` | LLM provider. Auto-detected from environment variables if not set. |
| `model` | string | _(provider default)_ | - | Model identifier |
| `api_key` | string | - | - | API key for authentication (use `${ENV_VAR}`) |
| `base_url` | string | - | - | Custom base URL for API endpoint |
| `temperature` | number | `0.7` | min: 0, max: 2 | Sampling temperature |
| `max_tokens` | integer | _(provider default)_ | min: 0 | Maximum tokens to generate. 0 means use provider default. |
| `max_tool_output_length` | integer | `0` | min: 0 | Maximum output length for tools tokens to avoid context length error |
| `thinking` | object | - | - | Extended thinking configuration (Claude) |

### llms.\<name\>.thinking

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `enabled` | boolean | `true` | - | Enable extended thinking |
| `budget_tokens` | integer | `1024` | min: 1 | Token budget for thinking |

---

## tools

Tool configurations. Each key is a unique tool name.

### tools.\<name\>

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `type` | string | `mcp` | enum: `mcp`, `function`, `command` | Type of tool |
| `enabled` | boolean | `true` | - | Whether the tool is active |
| `description` | string | - | - | What this tool does |
| `url` | string | - | - | MCP server URL (for type=mcp) |
| `transport` | string | - | enum: `stdio`, `sse`, `streamable-http` | MCP transport type |
| `command` | string | - | - | Command to execute MCP server (for type=mcp stdio) |
| `args` | []string | - | - | Arguments for MCP stdio transport |
| `env` | map[string]string | - | - | Environment variables for MCP stdio transport |
| `filter` | []string | - | - | Limit which tools are exposed from MCP server |
| `handler` | string | - | - | Function name (for type=function) |
| `parameters` | object | - | - | Parameters schema (for type=function) |
| `allowed_commands` | []string | - | - | Whitelist of allowed base commands |
| `denied_commands` | []string | - | - | Blacklist of denied base commands |
| `working_directory` | string | - | - | Working directory for command execution |
| `max_execution_time` | string | - | - | Maximum command execution duration |
| `deny_by_default` | boolean | `false` | - | Require explicit allowed_commands whitelist |
| `require_approval` | boolean | `false` | - | Whether this tool requires human approval (HITL) |
| `approval_prompt` | string | - | - | Message shown when requesting approval |

---

## agents

Agent configurations. Each key is a unique agent name (used in URL path).

### agents.\<name\>

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `name` | string | - | - | Human-readable display name for the agent (e.g. 'AI Assistant') |
| `description` | string | - | - | Human-readable description of agent's purpose |
| `visibility` | string | `public` | enum: `public`, `internal`, `private` | Controls agent discovery and access |
| `llm` | string | `default` | - | References a configured LLM by name |
| `tools` | []string | - | - | List of tool names this agent can use |
| `sub_agents` | []string | - | - | Child agents that can receive transferred control |
| `agent_tools` | []string | - | - | Agent names to use as callable tools |
| `instruction` | string | - | - | System prompt that defines agent behavior |
| `instruction_file` | string | - | - | Path to a file containing the system instruction |
| `global_instruction` | string | - | - | Instruction applied to all agents in the tree |
| `reasoning` | object | - | - | Chain-of-thought reasoning loop settings |
| `context` | object | - | - | Working memory and context window settings |
| `guardrails` | string | - | - | References a named guardrails configuration |
| `disable_safety_protocols` | boolean | `false` | - | Disable automatic injection of tool safety and CoT protocols |
| `prompt` | object | - | - | Detailed prompt configuration |
| `skills` | []object | - | - | Agent capabilities for A2A discovery |
| `input_modes` | []string | - | - | Supported input MIME types |
| `output_modes` | []string | - | - | Supported output MIME types |
| `streaming` | boolean | `true` | - | Token-by-token streaming from LLM |
| `document_stores` | []string | - | - | Document stores accessible to this agent |
| `include_context` | boolean | `false` | - | Automatically inject RAG context |
| `include_context_limit` | integer | `5` | min: 1 | Maximum number of documents to include |
| `include_context_max_length` | integer | `500` | min: 1 | Maximum content length per document (chars) |
| `structured_output` | object | - | - | JSON schema response format configuration |
| `type` | string | `llm` | enum: `llm`, `sequential`, `parallel`, `loop`, `remote`, `runner`, `conditional` | Type of agent |
| `max_iterations` | integer | - | min: 0 | Maximum iterations for loop agents |
| `condition_agent` | string | - | - | Agent that evaluates the routing condition |
| `condition_field` | string | `safe` | - | JSON field to check in condition agent output |
| `on_true_agent` | string | - | - | Agent to run when condition is true |
| `on_false_response` | string | `I cannot process this request.` | - | Static response when condition is false |
| `trigger` | object | - | - | Automatic invocation trigger (e.g. cron schedule) |
| `notifications` | []object | - | - | Outbound notifications for agent events |
| `url` | string | - | - | Base URL of remote A2A server |
| `agent_card_url` | string | - | - | URL to fetch agent card from |
| `agent_card_file` | string | - | - | Local file path to agent card JSON |
| `headers` | map[string]string | - | - | Custom headers for remote requests |
| `timeout` | string | `30s` | - | Request timeout |

### agents.\<name\>.reasoning

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `max_iterations` | integer | `100` | min: 1 | Safety limit for reasoning loop iterations |
| `enable_exit_tool` | boolean | `false` | - | Add exit_loop tool for explicit termination |
| `enable_escalate_tool` | boolean | `false` | - | Add escalate tool for parent delegation |
| `termination_conditions` | []string | - | - | Conditions that terminate the reasoning loop |
| `completion_instruction` | string | - | - | Instruction appended to help model know when to stop |

### agents.\<name\>.context

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `strategy` | string | `none` | enum: `none`, `buffer_window`, `token_window`, `summary_buffer` | Context window management strategy |
| `window_size` | integer | `20` | min: 1 | Minimum guaranteed messages (buffer floor). Dynamically reduced if Budget is too small. |
| `budget` | integer | `8000` | min: 1 | Primary token constraint. Dynamically limits WindowSize if set too low. |
| `threshold` | number | `0.85` | min: 0, max: 1 | Percentage of budget that triggers summarization |
| `target` | number | `0.7` | min: 0, max: 1 | Percentage of budget to reduce to after summarization |
| `preserve_recent` | integer | `5` | min: 0 | Minimum number of recent messages to always keep |
| `summarizer_llm` | string | - | - | LLM reference for summarization (uses agent LLM if empty) |

### agents.\<name\>.prompt

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `system_prompt` | string | - | - | Full system prompt (overrides Instruction) |
| `role` | string | - | - | Agent's role |
| `guidance` | string | - | - | Additional instructions |

### agents.\<name\>.skills[]

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `id` | string | - | - | Unique identifier for the skill |
| `name` | string | - | - | Display name |
| `description` | string | - | - | What this skill does |
| `tags` | []string | - | - | Tags for categorization |
| `examples` | []string | - | - | Example prompts this skill handles |

### agents.\<name\>.structured_output

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `schema` | object | - | - | JSON schema the response must conform to |
| `strict` | boolean | `true` | - | Enable strict schema validation |
| `name` | string | `response` | - | Optional name for the schema |

### agents.\<name\>.trigger

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `type` | string | `schedule` | enum: `schedule`, `webhook` | Type of trigger |
| `enabled` | boolean | `true` | - | Whether the trigger is active |
| `cron` | string | - | - | Cron schedule (e.g. '0 9 * * *' for daily at 9am) |
| `timezone` | string | `UTC` | - | Timezone for cron schedule |
| `input` | string | - | - | Static input for triggered runs |
| `path` | string | - | - | URL path for webhook endpoint |
| `methods` | []string | `[POST]` | - | Allowed HTTP methods |
| `secret` | string | - | - | Secret for webhook signature verification |
| `signature_header` | string | `X-Webhook-Signature` | - | HTTP header containing HMAC signature |
| `webhook_input` | object | - | - | Payload transformation configuration |
| `response` | object | - | - | Webhook response configuration |

### agents.\<name\>.trigger.webhook_input

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `template` | string | - | - | Go template for transforming webhook payload to agent input |
| `session_id` | string | - | - | Template for deriving session ID from payload for conversation continuity |
| `extract_fields` | []object | - | - | Fields to extract from payload |

### agents.\<name\>.trigger.webhook_input.extract_fields[]

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `path` | string | - | **Yes** | Path to field in payload |
| `as` | string | - | **Yes** | Name to use in template |

### agents.\<name\>.trigger.response

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `mode` | string | `sync` | enum: `sync`, `async`, `callback` | How to respond to webhook requests |
| `timeout` | integer | - | - | Max wait time for sync mode |
| `callback_url` | string | - | - | URL to POST results to in callback mode |

### agents.\<name\>.notifications[]

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `id` | string | - | **Yes** | Unique identifier for this notification |
| `type` | string | `webhook` | enum: `webhook` | Notification type |
| `enabled` | boolean | `true` | - | Whether the notification is active |
| `events` | []string | - | - | Events that trigger notification |
| `url` | string | - | **Yes** | URL to send notifications to |
| `headers` | map[string]string | - | - | Additional HTTP headers |
| `payload` | object | - | - | Payload configuration |
| `retry` | object | - | - | Retry configuration |

### agents.\<name\>.notifications[].payload

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `template` | string | - | - | Go template for payload generation |

### agents.\<name\>.notifications[].retry

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `max_attempts` | integer | `3` | - | Maximum retry attempts |
| `initial_delay` | integer | - | - | Initial retry delay |
| `max_delay` | integer | - | - | Maximum retry delay |

---

## guardrails

Guardrail configurations. Each key is a unique guardrail name.

### guardrails.\<name\>

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `true` | **Yes** | Whether guardrails are active |
| `input` | object | - | - | Input validation and sanitization settings |
| `output` | object | - | - | Output filtering and redaction settings |
| `tool` | object | - | - | Tool authorization settings |
| `moderation` | object | - | - | LLM-powered content moderation settings |

### guardrails.\<name\>.input

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `chain_mode` | string | `fail_fast` | enum: `fail_fast`, `collect_all` | How to handle multiple guardrails |
| `length` | object | - | - | Input length constraints |
| `injection` | object | - | - | Prompt injection protection |
| `sanitizer` | object | - | - | Input cleaning and normalization |
| `pattern` | object | - | - | Regex-based validation |

### guardrails.\<name\>.input.length

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `true` | **Yes** | Enable length validation |
| `min_length` | integer | `1` | - | Minimum input length |
| `max_length` | integer | `100000` | - | Maximum input length |
| `action` | string | `block` | enum: `allow`, `block`, `warn` | Action on violation |
| `severity` | string | `medium` | enum: `low`, `medium`, `high`, `critical` | Severity level |

### guardrails.\<name\>.input.injection

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `true` | **Yes** | Enable injection detection |
| `patterns` | []string | - | - | Additional regex patterns to detect |
| `case_sensitive` | boolean | `false` | - | Case sensitive matching |
| `action` | string | `block` | enum: `allow`, `block`, `warn` | Action on violation |
| `severity` | string | `high` | enum: `low`, `medium`, `high`, `critical` | Severity level |

### guardrails.\<name\>.input.sanitizer

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `true` | **Yes** | Enable sanitization |
| `trim_whitespace` | boolean | `true` | - | Trim leading/trailing whitespace |
| `normalize_unicode` | boolean | `false` | - | Normalize unicode characters |
| `max_length` | integer | - | - | Truncate if exceeded (0=no limit) |
| `strip_html` | boolean | `false` | - | Remove HTML tags |

### guardrails.\<name\>.input.pattern

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `false` | **Yes** | Enable pattern validation |
| `allow_patterns` | []string | - | - | Patterns that input must match |
| `block_patterns` | []string | - | - | Patterns that input must NOT match |
| `action` | string | `block` | enum: `allow`, `block`, `warn` | Action on violation |
| `severity` | string | `medium` | enum: `low`, `medium`, `high`, `critical` | Severity level |

### guardrails.\<name\>.output

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `chain_mode` | string | `fail_fast` | enum: `fail_fast`, `collect_all` | How to handle multiple guardrails |
| `pii` | object | - | - | Detect and redact personally identifiable information |
| `content` | object | - | - | Block or warn about harmful content |

### guardrails.\<name\>.output.pii

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `true` | **Yes** | Enable PII detection |
| `detect_email` | boolean | `true` | - | Detect email addresses |
| `detect_phone` | boolean | `true` | - | Detect phone numbers |
| `detect_ssn` | boolean | `true` | - | Detect social security numbers |
| `detect_credit_card` | boolean | `true` | - | Detect credit card numbers |
| `redact_mode` | string | `mask` | enum: `mask`, `remove`, `hash` | How to redact detected PII |
| `action` | string | `modify` | enum: `allow`, `block`, `modify`, `warn` | Action on detection |
| `severity` | string | `high` | enum: `low`, `medium`, `high`, `critical` | Severity level |

### guardrails.\<name\>.output.content

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `false` | **Yes** | Enable content filtering |
| `blocked_keywords` | []string | - | - | Case-insensitive keywords to block |
| `blocked_patterns` | []string | - | - | Regex patterns to block |
| `action` | string | `block` | enum: `allow`, `block`, `warn` | Action on violation |
| `severity` | string | `high` | enum: `low`, `medium`, `high`, `critical` | Severity level |

### guardrails.\<name\>.tool

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `chain_mode` | string | `fail_fast` | enum: `fail_fast`, `collect_all` | How to handle multiple guardrails |
| `authorization` | object | - | - | Control which tools can be called |

### guardrails.\<name\>.tool.authorization

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `false` | **Yes** | Enable tool authorization |
| `allowed_tools` | []string | - | - | Whitelist of allowed tools (glob patterns) |
| `blocked_tools` | []string | - | - | Blacklist of blocked tools (glob patterns) |
| `action` | string | `block` | enum: `allow`, `block`, `warn` | Action on violation |
| `severity` | string | `high` | enum: `low`, `medium`, `high`, `critical` | Severity level |

### guardrails.\<name\>.moderation

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `enabled` | boolean | `false` | **Yes** | Enable LLM moderation |
| `strategy` | string | `openai` | enum: `none`, `openai`, `lakera`, `prompt` | Moderation approach |
| `action` | string | `block` | enum: `block`, `warn` | Action on flagged content |
| `openai` | object | - | - | OpenAI Moderation API settings |
| `lakera` | object | - | - | Lakera Guard API settings |
| `prompt` | object | - | - | Custom LLM prompt settings |

### guardrails.\<name\>.moderation.openai

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `model` | string | `omni-moderation-latest` | - | Moderation model (omni-moderation-latest or text-moderation-latest) |
| `threshold` | number | `0.8` | min: 0, max: 1 | Score threshold for flagging |

### guardrails.\<name\>.moderation.lakera

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `project_id` | string | - | - | Lakera project ID for policy configuration |
| `breakdown` | boolean | - | - | Enable detailed detector breakdown |
| `endpoint` | string | - | - | Custom API endpoint URL |

### guardrails.\<name\>.moderation.prompt

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `llm` | string | - | - | LLM to use for moderation |
| `template` | string | - | - | Custom moderation prompt template |
| `safe_field` | string | `safe` | - | Field name for safe/unsafe result |

---

## vector_stores

Vector database configurations. Each key is a unique store name.

### vector_stores.\<name\>

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `type` | string | - | **Yes** | Vector store type (e.g., chromem, qdrant, pinecone) |
| `host` | string | - | - | Database host |
| `port` | integer | - | - | Database port |
| `api_key` | string | - | - | API key |
| `enable_tls` | boolean | - | - | Enable TLS connection |
| `persist_path` | string | - | - | Persistence path (for embedded stores) |
| `compress` | boolean | - | - | Enable compression |
| `collection` | string | - | - | Collection name |
| `index_name` | string | - | - | Index name (Pinecone) |
| `environment` | string | - | - | Environment (Pinecone) |
| `enable_persistence` | boolean | - | - | Enable persistence |

---

## embedders

Embedding provider configurations. Each key is a unique embedder name.

### embedders.\<name\>

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `provider` | string | - | - | Embedding provider |
| `model` | string | - | - | Model name |
| `api_key` | string | - | - | API key |
| `base_url` | string | - | - | Custom API endpoint |
| `dimension` | integer | - | - | Embedding dimension |
| `timeout` | integer | - | - | Request timeout in seconds |
| `batch_size` | integer | - | - | Batch size for embedding requests |
| `encoding_format` | string | - | - | Encoding format |
| `user` | string | - | - | User identifier |
| `input_type` | string | - | - | Input type (Cohere) |
| `output_dimension` | integer | - | - | Output dimension (for dimensionality reduction) |
| `truncate` | string | - | - | Truncation mode |

---

## document_stores

Document source configurations for RAG. Each key is a unique store name.

### document_stores.\<name\>

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `source` | object | - | **Yes** | Document source configuration |
| `chunking` | object | - | - | Chunking strategy |
| `vector_store` | string | - | - | Vector store reference |
| `embedder` | string | - | - | Embedder reference |
| `collection` | string | - | - | Collection name |
| `watch` | boolean | - | - | Watch for file changes |
| `incremental_indexing` | boolean | - | - | Enable incremental indexing |
| `search` | object | - | - | Search configuration |
| `indexing` | object | - | - | Indexing configuration |
| `mcp_parsers` | object | - | - | MCP parser configuration |

### document_stores.\<name\>.source

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `type` | string | - | **Yes** | Source type (blob, sql, api, collection) |
| `include` | []string | - | - | Include patterns |
| `exclude` | []string | - | - | Exclude patterns |
| `max_file_size` | integer | - | - | Maximum file size in bytes |
| `collection` | string | - | - | Collection name |
| `sql` | object | - | - | SQL source config |
| `api` | object | - | - | API source config |
| `blob` | object | - | - | Blob storage config |

### document_stores.\<name\>.source.sql

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `dsn` | string | - | - | Database connection string |
| `tables` | []object | - | **Yes** | Table configurations |

### document_stores.\<name\>.source.sql.tables[]

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `table` | string | - | **Yes** | Table name |
| `columns` | []string | - | **Yes** | Columns to index |
| `id_column` | string | - | **Yes** | Primary key column |
| `updated_column` | string | - | - | Column for incremental updates |
| `where_clause` | string | - | - | Filter clause |
| `metadata_columns` | []string | - | - | Additional metadata columns |

### document_stores.\<name\>.source.api

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `url` | string | - | **Yes** | API endpoint URL |
| `headers` | map[string]string | - | - | HTTP headers |
| `id_field` | string | - | **Yes** | Field for document ID |
| `content_field` | string | - | **Yes** | Field for document content |

### document_stores.\<name\>.source.blob

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `url` | string | - | **Yes** | Blob storage URL |
| `prefix` | string | - | - | Path prefix |

### document_stores.\<name\>.chunking

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `strategy` | string | - | - | Chunking strategy |
| `size` | integer | - | - | Chunk size |
| `overlap` | integer | - | - | Chunk overlap |
| `min_size` | integer | - | - | Minimum chunk size |
| `max_size` | integer | - | - | Maximum chunk size |
| `preserve_words` | boolean | - | - | Preserve word boundaries |

### document_stores.\<name\>.search

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `top_k` | integer | - | - | Number of results to return |
| `threshold` | number | - | - | Similarity threshold |
| `enable_hyde` | boolean | - | - | Enable HyDE (Hypothetical Document Embeddings) |
| `hyde_llm` | string | - | - | LLM reference for HyDE |
| `enable_rerank` | boolean | - | - | Enable result reranking |
| `rerank_llm` | string | - | - | LLM reference for reranking |
| `rerank_max_results` | integer | - | - | Maximum results after reranking |
| `enable_multi_query` | boolean | - | - | Enable multi-query retrieval |
| `multi_query_llm` | string | - | - | LLM reference for multi-query |
| `multi_query_count` | integer | - | - | Number of query variations |

### document_stores.\<name\>.indexing

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `max_concurrent` | integer | - | - | Maximum concurrent indexing operations |
| `retry` | object | - | - | Retry configuration |

### document_stores.\<name\>.indexing.retry

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `max_retries` | integer | - | - | Maximum retry attempts |
| `base_delay` | integer | - | - | Base delay between retries (ms) |
| `max_delay` | integer | - | - | Maximum delay (ms) |
| `jitter` | number | - | - | Jitter factor (0-1) |

### document_stores.\<name\>.mcp_parsers

| Property | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `tool_names` | []string | - | **Yes** | MCP tool names for parsing |
| `extensions` | []string | - | - | File extensions to process |
| `priority` | integer | - | - | Parser priority |
| `prefer_native` | boolean | - | - | Prefer native parsers |
| `path_prefix` | string | - | - | Path prefix for files |

---

## defaults

Default values for agent configurations.

### defaults

| Property | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `llm` | string | - | - | Default LLM for agents that don't specify one |
