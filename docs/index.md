---
title: Hector Documentation
description: Declarative A2A-Native AI Agent Platform - Complete Documentation
hide:
  - navigation
  - toc
---

<style>
.md-content h1:first-child {
  display: none;
}
.md-content {
  padding-top: 0.5rem;
}
</style>

<p class="hero-slogan">Production-Ready AI Agents. Zero Code Required.</p>

<div class="hero-section">
  <div class="hero-demo">
    <video controls autoplay muted loop playsinline class="hector-video">
      <source src="assets/hector_chat.mp4" type="video/mp4">
      Your browser does not support the video tag.
    </video>
  </div>

  <div class="hero-content">

```bash
export OPENAI_API_KEY="sk-..." MCP_URL="http://localhost:3000"

cat > weather.yaml << 'EOF'
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"

tools:
  weather:
    type: "mcp"
    server_url: "${MCP_URL}"

agents:
  weather_assistant:
    name: Weather Assistant
    llm: "gpt-4o"
    tools: [weather]
EOF

hector serve --config weather.yaml

# Open your browser and experience the interactive chat interface
open http://localhost:8080
```

  </div>
</div>

<div class="hero-intro">
  <p>Deploy observable, secure, and scalable AI agent systems with hot reload, distributed configuration, and A2A-native federation. Configure in YAML or build programmatically in Go with full support for document stores, observability, and all agent features.</p>

  <p>Built for real-world complexity: orchestrate multiple specialized agents, equip them with advanced reasoning strategies like chain-of-thought and tree search, give them long-term memory with RAG and vector stores, connect them to any tool or API, and stream responses in real-time. Enable production-ready human-in-the-loop workflows with async state persistence and checkpoint recovery that survive server restarts. All while maintaining production-grade observability and security.</p>
</div>

<!-- A2A Federation Story Section -->
<div class="a2a-story-section">
  <h2 class="a2a-story-title">A2A-Native Federation</h2>
  
  <div class="a2a-content-row">
    <div class="a2a-story-content">
      <p class="a2a-story-text">
        Hector is built on the <strong>Agent-to-Agent (A2A) protocol</strong> — an open standard for agent communication. Unlike traditional systems with central orchestrators that create bottlenecks and single points of failure, Hector enables true peer-to-peer federation.
      </p>
      <p class="a2a-story-text">
        Deploy agents across different services, languages, and infrastructures. They communicate directly using the A2A protocol, forming federated networks where each agent maintains autonomy while collaborating seamlessly — no central control, no lock-in.
      </p>
    </div>
    
    <div class="a2a-visualization">
      <svg id="a2a-federation" viewBox="50 60 400 300" preserveAspectRatio="xMidYMin meet" xmlns="http://www.w3.org/2000/svg">
      <!-- Glow filters - subtle for nodes only -->
      <defs>
        <filter id="glow-yellow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="glow-purple" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="glow-cyan" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
      </defs>
      
      <!-- Connections -->
      <g class="connections">
        <!-- A ↔ B: Horizontal dotted line -->
        <line id="conn-ab" x1="170" y1="105" x2="330" y2="105" 
              stroke="rgba(251, 191, 36, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        <text x="250" y="85" text-anchor="middle" fill="rgba(255, 255, 255, 0.8)" 
              font-size="11" font-family="sans-serif">A2A Protocol</text>
        
        <!-- A → C: L-shape with rounded corner (connects to C's left middle) -->
        <path id="conn-ac" d="M 120 130 L 120 305 Q 120 325 140 325 L 200 325" 
              stroke="rgba(6, 182, 212, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        
        <!-- B → C: L-shape with rounded corner (connects to C's right middle) -->
        <path id="conn-bc" d="M 380 130 L 380 305 Q 380 325 360 325 L 300 325" 
              stroke="rgba(139, 92, 246, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
      </g>
      
      <!-- Animated particles -->
      <g class="particles">
        <circle id="particle-ab" r="5" fill="#fbbf24">
          <animateMotion dur="3s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 170 105 L 330 105"/>
        </circle>
        <circle id="particle-ac" r="5" fill="#06b6d4">
          <animateMotion dur="4s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 120 130 L 120 305 Q 120 325 140 325 L 200 325"/>
        </circle>
        <circle id="particle-bc" r="5" fill="#8b5cf6">
          <animateMotion dur="4.5s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 380 130 L 380 305 Q 380 325 360 325 L 300 325"/>
        </circle>
      </g>
      
      <!-- Nodes -->
      <g class="nodes">
        <g class="node" id="node-a" filter="url(#glow-yellow)">
          <rect x="70" y="80" width="100" height="50" rx="12" 
                fill="#fbbf24" stroke="none"/>
          <text x="120" y="108" text-anchor="middle" fill="rgba(0, 0, 0, 0.9)" 
                font-size="14" font-weight="bold" font-family="sans-serif">Agent A</text>
        </g>
        <g class="node" id="node-b" filter="url(#glow-purple)">
          <rect x="330" y="80" width="100" height="50" rx="12" 
                fill="#8b5cf6" stroke="none"/>
          <text x="380" y="108" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">Agent B</text>
        </g>
        <g class="node" id="node-c" filter="url(#glow-cyan)">
          <rect x="200" y="300" width="100" height="50" rx="12" 
                fill="#06b6d4" stroke="none"/>
          <text x="250" y="328" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">Agent C</text>
        </g>
      </g>
    </svg>
    </div>
  </div>
  
  <div class="a2a-story-features">
    <div class="a2a-feature-item">
      <strong>Peer-to-Peer</strong>
      <span>Direct agent communication</span>
    </div>
    <div class="a2a-feature-item">
      <strong>No Central Control</strong>
      <span>Federated, autonomous agents</span>
    </div>
    <div class="a2a-feature-item">
      <strong>Standards-Based</strong>
      <span>Full A2A protocol implementation</span>
    </div>
  </div>
</div>

<!-- RAG Ready Story Section -->
<div class="a2a-story-section">
  <h2 class="a2a-story-title">RAG Ready</h2>
  
  <div class="a2a-content-row">
    <div class="a2a-story-content">
      <p class="a2a-story-text">
        Hector comes <strong>RAG-ready</strong> out of the box. Connect SQL databases, APIs, and document stores to build comprehensive knowledge bases. Choose from Ollama, OpenAI, or Cohere embedders to convert your data into vectors, automatically indexed into vector databases like Qdrant.
      </p>
      <p class="a2a-story-text">
        Agents retrieve relevant context from vector databases using semantic search, augmenting their responses with accurate, context-aware answers. No complex setup required—everything works out of the box.
      </p>
    </div>
    
    <div class="a2a-visualization">
      <svg id="rag-visualization" viewBox="50 60 350 240" preserveAspectRatio="xMidYMin meet" xmlns="http://www.w3.org/2000/svg">
      <!-- Glow filters - subtle for nodes only -->
      <defs>
        <filter id="rag-glow-orange" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="rag-glow-purple" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="rag-glow-blue" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="rag-glow-cyan" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
      </defs>
      
      <!-- Connections -->
      <g class="connections">
        <!-- Data Sources → Index convergence point → Vector DB -->
        <!-- SQL → Index (from right edge of SQL box at y-center) -->
        <path id="conn-sql-index" d="M 170 85 L 220 150" 
              stroke="rgba(251, 191, 36, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        <!-- API → Index (from right edge of API box at y-center) -->
        <path id="conn-api-index" d="M 170 150 L 220 150" 
              stroke="rgba(139, 92, 246, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        <!-- Docs → Index (from right edge of Docs box at y-center) -->
        <path id="conn-docs-index" d="M 170 215 L 220 150" 
              stroke="rgba(59, 130, 246, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        
        <!-- Index → Vector DB (from convergence point to left edge of Vector DB) -->
        <path id="conn-index-vector" d="M 220 150 L 280 150" 
              stroke="rgba(6, 182, 212, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        <text x="250" y="135" text-anchor="middle" fill="rgba(255, 255, 255, 0.8)" 
              font-size="11" font-family="sans-serif">Index</text>
      </g>
      
      <!-- Animated particles -->
      <g class="particles">
        <!-- Indexing flow particles -->
        <circle id="rag-particle-sql" r="4" fill="#fbbf24">
          <animateMotion dur="4s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 170 85 L 220 150 L 280 150"/>
        </circle>
        <circle id="rag-particle-api" r="4" fill="#8b5cf6">
          <animateMotion dur="4.5s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 170 150 L 220 150 L 280 150"/>
        </circle>
        <circle id="rag-particle-docs" r="4" fill="#3b82f6">
          <animateMotion dur="5s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 170 215 L 220 150 L 280 150"/>
        </circle>
        
        <!-- Particle on Index → Vector DB connection (cyan to match Vector DB) -->
        <circle id="rag-particle-index" r="4" fill="#06b6d4">
          <animateMotion dur="3s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 220 150 L 280 150"/>
        </circle>
      </g>
      
      <!-- Nodes -->
      <g class="nodes">
        <!-- Data Sources (Left) - All same size, properly spaced -->
        <g class="node" id="rag-node-sql" filter="url(#rag-glow-orange)">
          <rect x="70" y="60" width="100" height="50" rx="12" 
                fill="#fbbf24" stroke="none"/>
          <text x="120" y="88" text-anchor="middle" fill="rgba(0, 0, 0, 0.9)" 
                font-size="14" font-weight="bold" font-family="sans-serif">SQL</text>
        </g>
        <g class="node" id="rag-node-api" filter="url(#rag-glow-purple)">
          <rect x="70" y="125" width="100" height="50" rx="12" 
                fill="#8b5cf6" stroke="none"/>
          <text x="120" y="153" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">API</text>
        </g>
        <g class="node" id="rag-node-docs" filter="url(#rag-glow-blue)">
          <rect x="70" y="190" width="100" height="50" rx="12" 
                fill="#3b82f6" stroke="none"/>
          <text x="120" y="218" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">Docs</text>
        </g>
        
        <!-- Vector DB (Right-Center) - Vertically aligned with API, better positioned -->
        <g class="node" id="rag-node-vector" filter="url(#rag-glow-cyan)">
          <rect x="280" y="125" width="100" height="50" rx="12" 
                fill="#06b6d4" stroke="none"/>
          <text x="330" y="153" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">Vector DB</text>
        </g>
      </g>
    </svg>
    </div>
  </div>
  
  <div class="a2a-story-features">
    <div class="a2a-feature-item">
      <strong>Multi-Source</strong>
      <span>SQL, API, and document stores</span>
    </div>
    <div class="a2a-feature-item">
      <strong>Embedders</strong>
      <span>Ollama, OpenAI, and Cohere</span>
    </div>
    <div class="a2a-feature-item">
      <strong>Vector Databases</strong>
      <span>Qdrant and custom vector stores</span>
    </div>
  </div>
</div>

<!-- Distributed Configuration Story Section -->
<div class="a2a-story-section">
  <h2 class="a2a-story-title">Distributed Configuration</h2>
  
  <div class="a2a-content-row">
    <div class="a2a-story-content">
      <p class="a2a-story-text">
        Start simple with file-based configuration for development, then seamlessly scale to enterprise-grade distributed backends. Hector supports <strong>Consul</strong> for service mesh environments and <strong>ZooKeeper</strong> for big data infrastructure.
      </p>
      <p class="a2a-story-text">
        Enable hot reload with <code>--config-watch</code> for zero-downtime configuration updates. Changes are detected reactively—no polling required. Configuration updates trigger graceful reloads, keeping your agents running smoothly across all instances.
      </p>
    </div>
    
    <div class="a2a-visualization">
      <svg id="config-visualization" viewBox="50 60 400 260" preserveAspectRatio="xMidYMin meet" xmlns="http://www.w3.org/2000/svg">
      <!-- Glow filters - subtle for nodes only -->
      <defs>
        <filter id="config-glow-orange" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="config-glow-purple" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="config-glow-blue" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="config-glow-cyan" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
      </defs>
      
      <!-- Connections -->
      <g class="connections">
        <!-- Config Sources → Config (all start from bottom middle, join Config at left/top/right middle) -->
        <!-- File → Config: bottom middle → vertical down → horizontal right to Config left middle -->
        <path id="conn-file-config" d="M 120 110 L 120 175 L 200 175" 
              stroke="rgba(251, 191, 36, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        <!-- Consul → Config: bottom middle → vertical down to Config top middle -->
        <path id="conn-consul-config" d="M 250 110 L 250 150" 
              stroke="rgba(139, 92, 246, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        <!-- ZooKeeper → Config: bottom middle → vertical down → horizontal left to Config right middle -->
        <path id="conn-zk-config" d="M 380 110 L 380 175 L 300 175" 
              stroke="rgba(59, 130, 246, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        
        <!-- Config → Runtime: bottom middle → vertical down to Runtime top middle -->
        <path id="conn-config-runtime" d="M 250 200 L 250 250" 
              stroke="rgba(6, 182, 212, 0.6)" stroke-width="2.5" 
              stroke-dasharray="5,5" fill="none"/>
        <text x="265" y="225" text-anchor="middle" fill="rgba(255, 255, 255, 0.8)" 
              font-size="11" font-family="sans-serif">Hot Reload</text>
      </g>
      
      <!-- Animated particles -->
      <g class="particles">
        <!-- Config source particles -->
        <circle id="config-particle-file" r="4" fill="#fbbf24">
          <animateMotion dur="4s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 120 110 L 120 175 L 200 175 L 250 175 L 250 200 L 250 250"/>
        </circle>
        <circle id="config-particle-consul" r="4" fill="#8b5cf6">
          <animateMotion dur="4.5s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 250 110 L 250 150 L 250 175 L 250 200 L 250 250"/>
        </circle>
        <circle id="config-particle-zk" r="4" fill="#3b82f6">
          <animateMotion dur="5s" repeatCount="indefinite" 
                         calcMode="spline" keyTimes="0;1" 
                         keySplines="0.42 0 0.58 1"
                         path="M 380 110 L 380 175 L 300 175 L 250 175 L 250 200 L 250 250"/>
        </circle>
      </g>
      
      <!-- Nodes -->
      <g class="nodes">
        <!-- Config Sources (Top row, horizontal) -->
        <g class="node" id="config-node-file" filter="url(#config-glow-orange)">
          <rect x="70" y="60" width="100" height="50" rx="12" 
                fill="#fbbf24" stroke="none"/>
          <text x="120" y="88" text-anchor="middle" fill="rgba(0, 0, 0, 0.9)" 
                font-size="14" font-weight="bold" font-family="sans-serif">File</text>
        </g>
        <g class="node" id="config-node-consul" filter="url(#config-glow-purple)">
          <rect x="200" y="60" width="100" height="50" rx="12" 
                fill="#8b5cf6" stroke="none"/>
          <text x="250" y="88" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">Consul</text>
        </g>
        <g class="node" id="config-node-zk" filter="url(#config-glow-blue)">
          <rect x="330" y="60" width="100" height="50" rx="12" 
                fill="#3b82f6" stroke="none"/>
          <text x="380" y="88" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">ZooKeeper</text>
        </g>
        
        <!-- Config (Center, middle row) -->
        <g class="node" id="config-node-config" filter="url(#config-glow-cyan)">
          <rect x="200" y="150" width="100" height="50" rx="12" 
                fill="#06b6d4" stroke="none"/>
          <text x="250" y="178" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">Config</text>
        </g>
        
        <!-- Runtime (Bottom) -->
        <g class="node" id="config-node-runtime" filter="url(#config-glow-cyan)">
          <rect x="200" y="250" width="100" height="50" rx="12" 
                fill="#06b6d4" stroke="none"/>
          <text x="250" y="278" text-anchor="middle" fill="rgba(255, 255, 255, 0.95)" 
                font-size="14" font-weight="bold" font-family="sans-serif">Runtime</text>
        </g>
      </g>
    </svg>
    </div>
  </div>
  
  <div class="a2a-story-features">
    <div class="a2a-feature-item">
      <strong>Hot Reload</strong>
      <span>Zero-downtime configuration updates</span>
    </div>
    <div class="a2a-feature-item">
      <strong>Distributed Backends</strong>
      <span>Consul and ZooKeeper support</span>
    </div>
    <div class="a2a-feature-item">
      <strong>Reactive Watching</strong>
      <span>Instant updates, no polling</span>
    </div>
  </div>
</div>

<!-- Programmatic API Story Section -->
<div class="a2a-story-section">
  <h2 class="a2a-story-title">Programmatic API</h2>
  
  <div class="a2a-content-row">
    <div class="a2a-story-content">
      <p class="a2a-story-text">
        When you need to customize agent behavior, integrate with existing systems, or build dynamic agent workflows, Hector's <strong>programmatic API</strong> gives you full control. Build agents programmatically, combine them with config-based agents, or embed agents directly into your Go applications.
      </p>
      <p class="a2a-story-text">
        The configuration system is built on top of the programmatic API, not the other way around. This means you can mix and match: load agents from YAML files, build custom agents programmatically, and combine them all in a single runtime. Perfect for embedding agents into existing applications or building higher-level abstractions.
      </p>
    </div>
    
    <div class="a2a-visualization">
      <div class="programmatic-code-block">
```go
// Load agents from YAML configuration
cfg, _ := config.LoadConfig(config.LoaderOptions{
    Path: "configs/agents.yaml",
})

// Build agents from config (uses programmatic API internally)
configBuilder, _ := hector.NewConfigAgentBuilder(cfg)
configAgents, _ := configBuilder.BuildAllAgents()

// Build a custom agent programmatically
llm, _ := hector.NewLLMProvider("openai").
    Model("gpt-4o-mini").
    APIKeyFromEnv("OPENAI_API_KEY").
    Build()

reasoning, _ := hector.NewReasoning("chain-of-thought").
    MaxIterations(100).
    Build()

programmaticAgent, _ := hector.NewAgent("custom").
    WithName("Custom Agent").
    WithLLMProvider(llm).
    WithReasoningStrategy(reasoning).
    WithSystemPrompt("You are a helpful assistant.").
    Build()

// Combine config-based and programmatic agents
runtime.NewRuntimeBuilder().
    WithAgents(configAgents).      // From YAML config
    WithAgent(programmaticAgent).   // Built programmatically
    Start()
```
      </div>
    </div>
  </div>
  
  <div class="a2a-story-features">
    <div class="a2a-feature-item">
      <strong>Flexible Building</strong>
      <span>Mix config and code seamlessly</span>
    </div>
    <div class="a2a-feature-item">
      <strong>Chained API</strong>
      <span>Fluent, readable builder pattern</span>
    </div>
    <div class="a2a-feature-item">
      <strong>Full Control</strong>
      <span>Every feature available programmatically</span>
    </div>
  </div>
</div>

[Get Started →](getting-started/quick-start.md){ .md-button .md-button--primary }

## Key Features

<div class="grid cards" markdown>

-   :zap: __Zero-Code Configuration__

    Pure YAML agent definition. No Python/Go required—define sophisticated agents declaratively. Or use the [programmatic API](core-concepts/programmatic-api.md) for Go code integration with full support for document stores, observability, and all agent features.

-   :chart_with_upwards_trend: __Production Observability__

    Built-in Prometheus metrics and OpenTelemetry tracing. Monitor latency, token usage, costs, and errors.

-   :shield: __Security-First__

    JWT auth, RBAC, and command sandboxing out of the box. Production-grade security controls.

-   :arrows_counterclockwise: __Hot Reload__

    Update configurations without downtime. Reload from Consul/Etcd/ZooKeeper automatically.

-   :link: __A2A Protocol Native__

    Standards-compliant agent communication and federation for distributed, interoperable deployments.

-   :rocket: __Resource Efficient__

    Single 30MB binary, <100ms startup. 10-20x less resource usage than Python frameworks.

</div>

