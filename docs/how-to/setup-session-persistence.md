---
title: Setup Session Persistence
description: Configure persistent session storage for conversation continuity across restarts
---

# Setup Session Persistence

Learn how to configure session persistence to maintain conversation history and context across application restarts.

---

## Overview

Session persistence stores conversation metadata and working memory in a database, enabling:

- **Conversation resumption** after server restarts
- **Distributed deployments** with shared session state
- **Long-running conversations** that survive process crashes
- **Multi-agent isolation** with shared infrastructure

### Architecture

Hector uses a three-layer memory system:

```
┌─────────────────────────────────────────────────────┐
│ 1. SESSION STORE (NEW!)                             │
│    - Session metadata (created_at, agent_id)        │
│    - Working memory (conversation history)          │
│    - Database: SQL (SQLite, PostgreSQL, MySQL)      │
│    - Survives: Process restarts ✅                  │
├─────────────────────────────────────────────────────┤
│ 2. WORKING MEMORY                                   │
│    - Active conversation context                    │
│    - Managed by: summary_buffer, buffer_window      │
│    - Backed by: Session store                       │
├─────────────────────────────────────────────────────┤
│ 3. LONG-TERM MEMORY                                 │
│    - Semantic recall across conversations           │
│    - Vector database (Qdrant, Pinecone, etc.)      │
│    - Isolated by: agent_id + session_id             │
└─────────────────────────────────────────────────────┘
```

**Key Point:** Session persistence (layer 1) is the foundation that backs working memory, while long-term memory provides semantic search capabilities.

---

## Quick Start

### 1. Basic SQLite Setup

Simplest setup for local development:

```yaml
# config.yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./data/sessions.db
      max_conns: 10
      max_idle: 2

agents:
  assistant:
    llm: "gpt-4o"
    session_store: "main-db"  # Reference the global store
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
```

**Start server:**

```bash
./hector serve --config config.yaml
```

**Test resumption:**

```bash
# First conversation
curl -X POST http://localhost:8091/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "context_id": "my-session",
      "role": 1,
      "content": [{"text": "Remember: my project ID is ALPHA-789"}]
    }
  }'

# Restart server
pkill hector
./hector serve --config config.yaml &

# Resume conversation - agent remembers!
curl -X POST http://localhost:8091/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "context_id": "my-session",
      "role": 1,
      "content": [{"text": "What is my project ID?"}]
    }
  }'
```

---

## Configuration Reference

### Global Session Stores

Define session stores once, reference them by name:

```yaml
session_stores:
  # Store name: can be referenced by multiple agents
  production-db:
    backend: sql
    sql:
      driver: postgres
      host: postgres.example.com
      port: 5432
      user: hector
      password: ${DB_PASSWORD}  # From environment
      database: hector_sessions
      ssl_mode: require
      max_conns: 50
      max_idle: 10
      conn_max_lifetime: 3600  # 1 hour

  dev-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./dev-sessions.db
      max_conns: 5
```

### Agent Configuration

Reference session stores by name:

```yaml
agents:
  assistant:
    llm: "gpt-4o"
    session_store: "production-db"  # References global store
    
  researcher:
    llm: "claude-3-5-sonnet-20241022"
    session_store: "production-db"  # Shares DB, isolated by agent_id
    
  dev-agent:
    llm: "gpt-4o-mini"
    session_store: "dev-db"  # Uses separate DB
```

**Multi-Agent Isolation:** Even when sharing a database, agents are isolated by `agent_id` in the sessions table. Agent A's session "test" is completely separate from Agent B's session "test".

---

## Database Setup

### SQLite (Development)

**Pros:**
- ✅ No setup required
- ✅ Single file database
- ✅ Perfect for local development

**Cons:**
- ❌ Not suitable for distributed deployments
- ❌ Limited concurrent writes

**Configuration:**

```yaml
session_stores:
  local:
    backend: sql
    sql:
      driver: sqlite
      database: ./data/sessions.db
      max_conns: 10        # SQLite supports limited concurrency
      max_idle: 2
```

**Schema auto-created:**

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    metadata TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
```

---

### PostgreSQL (Production)

**Pros:**
- ✅ Battle-tested reliability
- ✅ Excellent concurrent write performance
- ✅ Distributed deployment support
- ✅ Full ACID guarantees

**Setup:**

```bash
# 1. Install PostgreSQL
brew install postgresql  # macOS
apt install postgresql   # Ubuntu

# 2. Create database
createdb hector_sessions

# 3. Create user
psql -c "CREATE USER hector WITH PASSWORD 'secure-password';"
psql -c "GRANT ALL PRIVILEGES ON DATABASE hector_sessions TO hector;"
```

**Configuration:**

```yaml
session_stores:
  postgres:
    backend: sql
    sql:
      driver: postgres
      host: localhost
      port: 5432
      user: hector
      password: ${HECTOR_DB_PASSWORD}  # From environment
      database: hector_sessions
      ssl_mode: disable  # Use 'require' in production
      max_conns: 100
      max_idle: 25
      conn_max_lifetime: 3600
```

**Environment variable:**

```bash
export HECTOR_DB_PASSWORD="secure-password"
./hector serve --config config.yaml
```

---

### MySQL (Alternative)

**Setup:**

```bash
# 1. Install MySQL
brew install mysql  # macOS
apt install mysql-server  # Ubuntu

# 2. Create database
mysql -u root -p -e "CREATE DATABASE hector_sessions;"
mysql -u root -p -e "CREATE USER 'hector'@'localhost' IDENTIFIED BY 'secure-password';"
mysql -u root -p -e "GRANT ALL PRIVILEGES ON hector_sessions.* TO 'hector'@'localhost';"
```

**Configuration:**

```yaml
session_stores:
  mysql:
    backend: sql
    sql:
      driver: mysql
      host: localhost
      port: 3306
      user: hector
      password: ${HECTOR_DB_PASSWORD}
      database: hector_sessions
      max_conns: 100
      max_idle: 25
      conn_max_lifetime: 3600
```

---

## CLI Session Support

### Using --session Flag

Resume conversations across CLI invocations:

```bash
# First call - store information
./hector call --config config.yaml --session=work assistant \
  "Remember: meeting at 3pm"

# Later - agent remembers!
./hector call --config config.yaml --session=work assistant \
  "When is the meeting?"
```

**Interactive chat:**

```bash
# With custom session ID
./hector chat --config config.yaml --session=my-chat assistant

# Auto-generated session ID (displayed on start)
./hector chat --config config.yaml assistant
# Output: 💾 Session ID: cli-chat-1729612345
#         Resume later with: --session=cli-chat-1729612345
```

### Session Management

**Create session:**

```bash
SESSION_ID=$(uuidgen)  # Generate unique ID
./hector call --session=$SESSION_ID assistant "Hello"
```

**Resume session:**

```bash
./hector call --session=$SESSION_ID assistant "Continue"
```

**Check session in database:**

```bash
sqlite3 ./data/sessions.db "SELECT id, agent_id, created_at FROM sessions;"
```

---

## Multi-Agent Deployments

### Shared Database, Isolated Sessions

Multiple agents can share a single database with proper isolation:

```yaml
session_stores:
  shared-db:
    backend: sql
    sql:
      driver: postgres
      host: db.example.com
      database: hector_sessions
      user: hector
      password: ${DB_PASSWORD}
      max_conns: 200  # Shared across all agents

agents:
  customer-support:
    session_store: "shared-db"
    llm: "gpt-4o"
    
  sales-assistant:
    session_store: "shared-db"
    llm: "claude-3-5-sonnet-20241022"
    
  technical-advisor:
    session_store: "shared-db"
    llm: "gpt-4o"
```

**Isolation guarantees:**

- ✅ Sessions are isolated by `agent_id` + `session_id`
- ✅ Agent A cannot access Agent B's sessions
- ✅ Shared connection pool for efficiency
- ✅ Each agent maintains independent conversation history

---

## Testing Session Persistence

### Verification Script

```bash
#!/bin/bash

# 1. Start server
./hector serve --config config.yaml --port 8090 &
SERVER_PID=$!
sleep 3

# 2. Store information
curl -X POST http://localhost:8091/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "context_id": "test-session",
      "role": 1,
      "content": [{"text": "My favorite color is blue"}]
    }
  }'

# 3. Restart server
kill $SERVER_PID
sleep 2
./hector serve --config config.yaml --port 8090 &
sleep 3

# 4. Verify agent remembers
RESPONSE=$(curl -s -X POST http://localhost:8091/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "context_id": "test-session",
      "role": 1,
      "content": [{"text": "What is my favorite color?"}]
    }
  }')

if echo "$RESPONSE" | grep -q "blue"; then
    echo "✅ Session persistence works!"
else
    echo "❌ Session persistence failed"
fi
```

### Database Inspection

**View sessions:**

```bash
# SQLite
sqlite3 ./data/sessions.db "SELECT * FROM sessions;"

# PostgreSQL
psql -h localhost -U hector hector_sessions \
  -c "SELECT id, agent_id, created_at FROM sessions;"
```

**View messages:**

```bash
# SQLite
sqlite3 ./data/sessions.db \
  "SELECT session_id, role, substr(content, 1, 50) FROM messages;"

# PostgreSQL
psql -h localhost -U hector hector_sessions \
  -c "SELECT session_id, role, left(content, 50) FROM messages;"
```

---

## Performance Tuning

### Connection Pooling

```yaml
session_stores:
  production:
    backend: sql
    sql:
      # Connection pool settings
      max_conns: 100          # Maximum connections
      max_idle: 25            # Idle connections to keep
      conn_max_lifetime: 3600 # Close connections after 1 hour
```

**Guidelines:**

- **max_conns**: Set to 2-3x expected concurrent sessions
- **max_idle**: Set to 25% of max_conns
- **conn_max_lifetime**: 1-4 hours depending on database policy

### Database Optimization

**Indexes (auto-created by Hector):**

```sql
CREATE INDEX idx_sessions_agent_id ON sessions(agent_id);
CREATE INDEX idx_sessions_updated_at ON sessions(updated_at);
CREATE INDEX idx_messages_session_id ON messages(session_id);
```

**Maintenance:**

```sql
-- PostgreSQL: Vacuum regularly
VACUUM ANALYZE sessions;
VACUUM ANALYZE messages;

-- MySQL: Optimize tables
OPTIMIZE TABLE sessions;
OPTIMIZE TABLE messages;
```

---

## Security Considerations

### Connection Security

```yaml
session_stores:
  secure-db:
    backend: sql
    sql:
      driver: postgres
      host: db.prod.example.com
      ssl_mode: require        # ✅ Require SSL
      password: ${DB_PASSWORD} # ✅ From environment, never hardcode
```

### Access Control

```bash
# ✅ Good: Least privilege
GRANT SELECT, INSERT, UPDATE, DELETE ON sessions TO hector;
GRANT SELECT, INSERT, UPDATE, DELETE ON messages TO hector;

# ❌ Bad: Too broad
GRANT ALL PRIVILEGES ON DATABASE hector_sessions TO hector;
```

### Data Encryption

**At rest (database-level):**

```bash
# PostgreSQL with encryption
initdb --data-checksums /var/lib/postgresql/data

# MySQL with encryption
[mysqld]
innodb_encrypt_tables = ON
```

**In transit:**

```yaml
session_stores:
  encrypted:
    backend: sql
    sql:
      driver: postgres
      ssl_mode: require
      ssl_cert: /path/to/client-cert.pem
      ssl_key: /path/to/client-key.pem
      ssl_root_cert: /path/to/ca-cert.pem
```

---

## Troubleshooting

### Session Not Found

**Symptom:** Agent doesn't remember conversation after restart

**Check:**

```bash
# 1. Verify database exists
ls -la ./data/sessions.db  # SQLite

# 2. Check sessions table
sqlite3 ./data/sessions.db "SELECT COUNT(*) FROM sessions;"

# 3. Verify agent_id and session_id match
sqlite3 ./data/sessions.db \
  "SELECT id, agent_id FROM sessions WHERE id='your-session-id';"
```

**Solution:** Ensure you're using the same `context_id` in requests.

### Database Connection Errors

**Symptom:** `failed to connect to session store`

**Check:**

```bash
# PostgreSQL
pg_isready -h localhost -p 5432

# MySQL
mysqladmin ping -h localhost

# SQLite
ls -la ./data/sessions.db && sqlite3 ./data/sessions.db "SELECT 1;"
```

**Solution:** Verify connection parameters and database is running.

### Schema Migration Errors

**Symptom:** `table already exists` or `column not found`

**Solution:** Drop and recreate (⚠️ loses data):

```bash
# SQLite
rm ./data/sessions.db

# PostgreSQL
psql -h localhost -U hector hector_sessions -c "DROP TABLE messages; DROP TABLE sessions;"
```

---

## Production Deployment

### Docker Compose Example

```yaml
version: '3.8'

services:
  hector:
    image: hector:latest
    environment:
      - HECTOR_DB_PASSWORD=secure-password
    volumes:
      - ./config.yaml:/app/config.yaml
    ports:
      - "8080:8080"
    depends_on:
      - postgres

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: hector_sessions
      POSTGRES_USER: hector
      POSTGRES_PASSWORD: secure-password
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

volumes:
  postgres-data:
```

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hector
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: hector
        image: hector:latest
        env:
        - name: HECTOR_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
```

---

## Migration Guide

### From In-Memory to Persistent

**Before (no persistence):**

```yaml
agents:
  assistant:
    llm: "gpt-4o"
    memory:
      working:
        strategy: "summary_buffer"
```

**After (with persistence):**

```yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./data/sessions.db

agents:
  assistant:
    llm: "gpt-4o"
    session_store: "main-db"  # ← Add this line
    memory:
      working:
        strategy: "summary_buffer"
```

**No other changes needed!** Working memory automatically uses persistent storage.

---

## Best Practices

### ✅ Do

- Use PostgreSQL for production deployments
- Set appropriate connection pool sizes
- Store passwords in environment variables
- Enable SSL for database connections
- Monitor database size and performance
- Implement session cleanup (e.g., delete sessions older than 90 days)

### ❌ Don't

- Don't hardcode passwords in config files
- Don't use SQLite for distributed systems
- Don't share session IDs between users
- Don't ignore connection pool warnings
- Don't run without database backups in production

---

## Next Steps

- **[Memory Configuration](../core-concepts/memory.md)** - Working & long-term memory
- **[Sessions & Streaming](../core-concepts/sessions.md)** - Session lifecycle
- **[CLI Reference](../reference/cli.md)** - Command-line session support
- **[Configuration Reference](../reference/configuration.md)** - All session_stores options
- **[Deploy to Production](deploy-production.md)** - Production deployment guide

---

## Related Topics

- **[Architecture](../reference/architecture.md)** - How session persistence works internally
- **[Security](../core-concepts/security.md)** - Session security best practices
- **[Multi-Agent Systems](../core-concepts/multi-agent.md)** - Multi-agent session isolation

