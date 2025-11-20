---
hide:
  - navigation
  - toc
template: home.html
is_homepage: true
---

<div class="hero-container">
  <h1 class="hero-slogan">
    Production-Ready AI Agents.<br>
    <span class="text-gradient">Zero Code Required.</span>
  </h1>

  <div class="hero-intro">
    <p>
      Deploy observable, secure, and scalable AI agent systems with hot reload, distributed configuration, and A2A-native federation.
    </p>
    <div class="hero-cta" markdown="1">
      <a href="getting-started/quick-start/" class="btn btn-primary">Get Started</a>
      <a href="https://github.com/kadirpekel/hector" class="btn btn-secondary">
        <span class="twemoji"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 496 512"><path d="M165.9 397.4c0 2-2.3 3.6-5.2 3.6-3.3.3-5.6-1.3-5.6-3.6 0-2 2.3-3.6 5.2-3.6 3-.3 5.6 1.3 5.6 3.6zm-31.1-4.5c-.7 2 1.3 4.3 4.3 4.9 2.6 1 5.6 0 6.2-2s-1.3-4.3-4.3-5.2c-2.6-.7-5.5.3-6.2 2.3zm44.2-1.7c-2.9.7-4.9 2.6-4.6 4.9.3 2 2.9 3.3 5.9 2.6 2.9-.7 4.9-2.6 4.6-4.6-.3-1.9-3-3.2-5.9-2.9zM244.8 8C106.1 8 0 113.3 0 252c0 110.9 69.8 205.8 169.5 239.2 12.8 2.3 17.3-5.6 17.3-12.1 0-6.2-.3-40.4-.3-61.4 0 0-70 15-84.7-29.8 0 0-11.4-29.1-27.8-36.6 0 0-22.9-15.7 1.6-15.4 0 0 24.9 2 38.6 25.8 21.9 38.6 58.6 27.5 72.9 20.9 2.3-16 8.8-27.1 16-33.7-55.9-6.2-112.3-14.3-112.3-110.5 0-27.5 7.6-41.3 23.6-58.9-2.6-6.5-11.1-33.3 2.6-67.9 20.9-6.5 69 27 69 27 20-5.6 41.5-8.5 62.8-8.5s42.8 2.9 62.8 8.5c0 0 48.1-33.6 69-27 13.7 34.7 5.2 61.4 2.6 67.9 16 17.7 25.8 31.5 25.8 58.9 0 96.5-58.9 104.2-114.8 110.5 9.2 7.9 17 22.9 17 46.4 0 33.7-.3 75.4-.3 83.6 0 6.5 4.6 14.4 17.3 12.1C428.2 457.8 496 362.9 496 252 496 113.3 383.5 8 244.8 8zM97.2 352.9c-1.3 1-1 3.3.7 5.2 1.6 1.6 3.9 2.3 5.2 1 1.3-1 1-3.3-.7-5.2-1.6-1.6-3.9-2.3-5.2-1zm-10.8-8.1c-.7 1.3.3 2.9 2.3 3.9 1.6 1 3.6.7 4.3-.7.7-1.3-.3-2.9-2.3-3.9-2-.6-3.6-.3-4.3.7zm32.4 35.6c-1.6 1.3-1 4.3 1.3 6.2 2.3 2.3 5.2 2.6 6.5 1 1.3-1.3.7-4.3-1.3-6.2-2.2-2.3-5.2-2.6-6.5-1zm-11.4-14.7c-1.6 1-1.6 3.6 0 5.9 1.6 2.3 4.3 3.3 5.6 2.3 1.6-1.3 1.6-3.9 0-6.2-1.4-2.3-4-3.3-5.6-2z"/></svg></span>
        GitHub
      </a>
    </div>
  </div>

  <div class="hero-demo-window">
    <div class="window-header">
      <div class="window-dot red"></div>
      <div class="window-dot yellow"></div>
      <div class="window-dot green"></div>
      <div class="window-title">hector-server — -zsh — 80x24</div>
    </div>
    <div class="window-content">
      <div class="terminal-typing" id="typewriter-target"></div>
    </div>
  </div>
</div>

<!-- A2A Federation Story Section -->
<div class="story-section">
  <div class="story-container">
    <div class="story-text">
      <h2 class="story-title">A2A-Native Federation</h2>
      <p>
        Hector is built on the <strong>Agent-to-Agent (A2A) protocol</strong>. Unlike traditional systems with central orchestrators, Hector enables true peer-to-peer federation.
      </p>
      <p>
        Agents communicate directly, forming federated networks where each agent maintains autonomy while collaborating seamlessly.
      </p>
    </div>
    <div class="story-visual">
      <svg id="a2a-federation" viewBox="50 60 400 300" preserveAspectRatio="xMidYMin meet" xmlns="http://www.w3.org/2000/svg">
      <!-- Enhanced neon glow filters for 3D effect -->
      <defs>
        <filter id="glow-yellow" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="glow-purple" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="glow-cyan" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>

        <!-- Laser beam gradient -->
        <linearGradient id="laser-yellow" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" style="stop-color:rgba(251, 191, 36, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#fbbf24; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(251, 191, 36, 0); stop-opacity:0" />
        </linearGradient>
        <linearGradient id="laser-cyan" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" style="stop-color:rgba(6, 182, 212, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#06b6d4; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(6, 182, 212, 0); stop-opacity:0" />
        </linearGradient>
        <linearGradient id="laser-purple" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" style="stop-color:rgba(139, 92, 246, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#8b5cf6; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(139, 92, 246, 0); stop-opacity:0" />
        </linearGradient>
      </defs>

      <!-- Laser-like connections with animated glow -->
      <g class="connections">
        <!-- A ↔ B: Horizontal laser beam -->
        <line x1="170" y1="105" x2="330" y2="105"
              stroke="rgba(251, 191, 36, 0.3)" stroke-width="2" fill="none"/>
        <line x1="170" y1="105" x2="330" y2="105"
              stroke="url(#laser-yellow)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2s" repeatCount="indefinite"/>
        </line>
        <text x="250" y="85" text-anchor="middle" fill="rgba(255, 255, 255, 0.9)"
              font-size="11" font-weight="600" font-family="Inter, sans-serif">A2A Protocol</text>

        <!-- A → C: L-shape laser -->
        <path d="M 120 130 L 120 305 Q 120 325 140 325 L 200 325"
              stroke="rgba(6, 182, 212, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 120 130 L 120 305 Q 120 325 140 325 L 200 325"
              stroke="url(#laser-cyan)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2.3s" repeatCount="indefinite"/>
        </path>

        <!-- B → C: L-shape laser -->
        <path d="M 380 130 L 380 305 Q 380 325 360 325 L 300 325"
              stroke="rgba(139, 92, 246, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 380 130 L 380 305 Q 380 325 360 325 L 300 325"
              stroke="url(#laser-purple)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2.5s" repeatCount="indefinite"/>
        </path>
      </g>

      <!-- Animated light particles (smaller, brighter) -->
      <g class="particles">
        <circle r="3" fill="#fbbf24" filter="url(#glow-yellow)">
          <animateMotion dur="3s" repeatCount="indefinite" path="M 170 105 L 330 105"/>
        </circle>
        <circle r="3" fill="#06b6d4" filter="url(#glow-cyan)">
          <animateMotion dur="4s" repeatCount="indefinite" path="M 120 130 L 120 305 Q 120 325 140 325 L 200 325"/>
        </circle>
        <circle r="3" fill="#8b5cf6" filter="url(#glow-purple)">
          <animateMotion dur="4.5s" repeatCount="indefinite" path="M 380 130 L 380 305 Q 380 325 360 325 L 300 325"/>
        </circle>
      </g>

      <!-- 3D Neon Cube Nodes -->
      <g class="nodes">
        <!-- Agent A - 3D Cube (Yellow/Orange) -->
        <g class="node" id="node-a">
          <!-- Back face (darker) -->
          <path d="M 78,83 L 165,83 L 173,76 L 86,76 Z" fill="rgba(251, 191, 36, 0.4)"/>
          <!-- Top face (lighter) -->
          <path d="M 78,83 L 86,76 L 86,124 L 78,131 Z" fill="rgba(251, 191, 36, 0.6)"/>
          <!-- Front face (brightest) with neon edge -->
          <rect x="78" y="83" width="87" height="48" rx="4"
                fill="rgba(251, 191, 36, 0.15)"
                stroke="#fbbf24" stroke-width="2" filter="url(#glow-yellow)"/>
          <text x="121" y="111" text-anchor="middle" fill="#fbbf24"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">Agent A</text>
        </g>

        <!-- Agent B - 3D Cube (Purple) -->
        <g class="node" id="node-b">
          <!-- Back face -->
          <path d="M 338,83 L 422,83 L 430,76 L 346,76 Z" fill="rgba(139, 92, 246, 0.4)"/>
          <!-- Top face -->
          <path d="M 338,83 L 346,76 L 346,124 L 338,131 Z" fill="rgba(139, 92, 246, 0.6)"/>
          <!-- Front face with neon edge -->
          <rect x="338" y="83" width="84" height="48" rx="4"
                fill="rgba(139, 92, 246, 0.15)"
                stroke="#8b5cf6" stroke-width="2" filter="url(#glow-purple)"/>
          <text x="380" y="111" text-anchor="middle" fill="#8b5cf6"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">Agent B</text>
        </g>

        <!-- Agent C - 3D Cube (Cyan) -->
        <g class="node" id="node-c">
          <!-- Back face -->
          <path d="M 208,303 L 295,303 L 303,296 L 216,296 Z" fill="rgba(6, 182, 212, 0.4)"/>
          <!-- Top face -->
          <path d="M 208,303 L 216,296 L 216,344 L 208,351 Z" fill="rgba(6, 182, 212, 0.6)"/>
          <!-- Front face with neon edge -->
          <rect x="208" y="303" width="87" height="48" rx="4"
                fill="rgba(6, 182, 212, 0.15)"
                stroke="#06b6d4" stroke-width="2" filter="url(#glow-cyan)"/>
          <text x="251" y="331" text-anchor="middle" fill="#06b6d4"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">Agent C</text>
        </g>
      </g>
    </svg>
    </div>
  </div>
</div>

<!-- RAG Ready Story Section -->
<div class="story-section">
  <div class="story-container">
    <div class="story-text">
      <h2 class="story-title">RAG Ready</h2>
      <p>
        Hector comes <strong>RAG-ready</strong> out of the box. Connect SQL databases, APIs, and document stores to build comprehensive knowledge bases.
      </p>
      <p>
        Choose from Ollama, OpenAI, or Cohere embedders to convert your data into vectors, automatically indexed into vector databases like Qdrant. Agents retrieve relevant context using semantic search, augmenting their responses with accurate, context-aware answers.
      </p>
    </div>
    <div class="story-visual">
      <svg id="rag-visualization" viewBox="50 60 350 240" preserveAspectRatio="xMidYMin meet" xmlns="http://www.w3.org/2000/svg">
      <!-- Enhanced neon glow filters for 3D effect -->
      <defs>
        <filter id="rag-glow-orange" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="rag-glow-purple" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="rag-glow-blue" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="rag-glow-cyan" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>

        <!-- Laser beam gradients -->
        <linearGradient id="rag-laser-yellow">
          <stop offset="0%" style="stop-color:rgba(251, 191, 36, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#fbbf24; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(251, 191, 36, 0); stop-opacity:0" />
        </linearGradient>
        <linearGradient id="rag-laser-purple">
          <stop offset="0%" style="stop-color:rgba(139, 92, 246, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#8b5cf6; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(139, 92, 246, 0); stop-opacity:0" />
        </linearGradient>
        <linearGradient id="rag-laser-blue">
          <stop offset="0%" style="stop-color:rgba(59, 130, 246, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#3b82f6; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(59, 130, 246, 0); stop-opacity:0" />
        </linearGradient>
        <linearGradient id="rag-laser-cyan">
          <stop offset="0%" style="stop-color:rgba(6, 182, 212, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#06b6d4; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(6, 182, 212, 0); stop-opacity:0" />
        </linearGradient>
      </defs>

      <!-- Laser-like connections -->
      <g class="connections">
        <!-- SQL → Index -->
        <path d="M 170 85 L 220 150" stroke="rgba(251, 191, 36, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 170 85 L 220 150" stroke="url(#rag-laser-yellow)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2s" repeatCount="indefinite"/>
        </path>

        <!-- API → Index -->
        <path d="M 170 150 L 220 150" stroke="rgba(139, 92, 246, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 170 150 L 220 150" stroke="url(#rag-laser-purple)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2.2s" repeatCount="indefinite"/>
        </path>

        <!-- Docs → Index -->
        <path d="M 170 215 L 220 150" stroke="rgba(59, 130, 246, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 170 215 L 220 150" stroke="url(#rag-laser-blue)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2.4s" repeatCount="indefinite"/>
        </path>

        <!-- Index → Vector DB -->
        <path d="M 220 150 L 280 150" stroke="rgba(6, 182, 212, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 220 150 L 280 150" stroke="url(#rag-laser-cyan)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="1.8s" repeatCount="indefinite"/>
        </path>
        <text x="250" y="135" text-anchor="middle" fill="rgba(255, 255, 255, 0.9)"
              font-size="11" font-weight="600" font-family="Inter, sans-serif">Index</text>
      </g>

      <!-- Animated light particles -->
      <g class="particles">
        <circle r="3" fill="#fbbf24" filter="url(#rag-glow-orange)">
          <animateMotion dur="4s" repeatCount="indefinite" path="M 170 85 L 220 150 L 280 150"/>
        </circle>
        <circle r="3" fill="#8b5cf6" filter="url(#rag-glow-purple)">
          <animateMotion dur="4.5s" repeatCount="indefinite" path="M 170 150 L 220 150 L 280 150"/>
        </circle>
        <circle r="3" fill="#3b82f6" filter="url(#rag-glow-blue)">
          <animateMotion dur="5s" repeatCount="indefinite" path="M 170 215 L 220 150 L 280 150"/>
        </circle>
      </g>

      <!-- 3D Neon Cube Nodes -->
      <g class="nodes">
        <!-- SQL - 3D Cube -->
        <g class="node" id="rag-node-sql">
          <path d="M 78,63 L 165,63 L 171,57 L 84,57 Z" fill="rgba(251, 191, 36, 0.4)"/>
          <path d="M 78,63 L 84,57 L 84,104 L 78,110 Z" fill="rgba(251, 191, 36, 0.6)"/>
          <rect x="78" y="63" width="87" height="47" rx="4"
                fill="rgba(251, 191, 36, 0.15)"
                stroke="#fbbf24" stroke-width="2" filter="url(#rag-glow-orange)"/>
          <text x="121" y="90" text-anchor="middle" fill="#fbbf24"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">SQL</text>
        </g>

        <!-- API - 3D Cube -->
        <g class="node" id="rag-node-api">
          <path d="M 78,128 L 165,128 L 171,122 L 84,122 Z" fill="rgba(139, 92, 246, 0.4)"/>
          <path d="M 78,128 L 84,122 L 84,169 L 78,175 Z" fill="rgba(139, 92, 246, 0.6)"/>
          <rect x="78" y="128" width="87" height="47" rx="4"
                fill="rgba(139, 92, 246, 0.15)"
                stroke="#8b5cf6" stroke-width="2" filter="url(#rag-glow-purple)"/>
          <text x="121" y="155" text-anchor="middle" fill="#8b5cf6"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">API</text>
        </g>

        <!-- Docs - 3D Cube -->
        <g class="node" id="rag-node-docs">
          <path d="M 78,193 L 165,193 L 171,187 L 84,187 Z" fill="rgba(59, 130, 246, 0.4)"/>
          <path d="M 78,193 L 84,187 L 84,234 L 78,240 Z" fill="rgba(59, 130, 246, 0.6)"/>
          <rect x="78" y="193" width="87" height="47" rx="4"
                fill="rgba(59, 130, 246, 0.15)"
                stroke="#3b82f6" stroke-width="2" filter="url(#rag-glow-blue)"/>
          <text x="121" y="220" text-anchor="middle" fill="#3b82f6"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">Docs</text>
        </g>

        <!-- Vector DB - 3D Cube -->
        <g class="node" id="rag-node-vector">
          <path d="M 288,128 L 372,128 L 378,122 L 294,122 Z" fill="rgba(6, 182, 212, 0.4)"/>
          <path d="M 288,128 L 294,122 L 294,169 L 288,175 Z" fill="rgba(6, 182, 212, 0.6)"/>
          <rect x="288" y="128" width="84" height="47" rx="4"
                fill="rgba(6, 182, 212, 0.15)"
                stroke="#06b6d4" stroke-width="2" filter="url(#rag-glow-cyan)"/>
          <text x="330" y="155" text-anchor="middle" fill="#06b6d4"
                font-size="13" font-weight="700" font-family="Inter, sans-serif">Vector DB</text>
        </g>
      </g>
    </svg>
    </div>
  </div>
</div>

<!-- Distributed Configuration Story Section -->
<div class="story-section">
  <div class="story-container">
    <div class="story-text">
      <h2 class="story-title">Distributed Configuration</h2>
      <p>
        Start simple with file-based configuration for development, then seamlessly scale to enterprise-grade distributed backends. Hector supports <strong>Consul</strong> for service mesh environments and <strong>ZooKeeper</strong> for big data infrastructure.
      </p>
      <p>
        Enable hot reload with <code>--config-watch</code> for zero-downtime configuration updates. Changes are detected reactively—no polling required. Configuration updates trigger graceful reloads, keeping your agents running smoothly across all instances.
      </p>
    </div>
    <div class="story-visual">
      <svg id="config-visualization" viewBox="50 60 400 260" preserveAspectRatio="xMidYMin meet" xmlns="http://www.w3.org/2000/svg">
      <!-- Enhanced neon glow filters for 3D effect -->
      <defs>
        <filter id="config-glow-orange" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="config-glow-purple" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="config-glow-blue" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
        <filter id="config-glow-cyan" x="-100%" y="-100%" width="300%" height="300%">
          <feGaussianBlur stdDeviation="4" result="coloredBlur"/>
          <feMerge>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="coloredBlur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>

        <!-- Laser beam gradients -->
        <linearGradient id="config-laser-yellow">
          <stop offset="0%" style="stop-color:rgba(251, 191, 36, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#fbbf24; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(251, 191, 36, 0); stop-opacity:0" />
        </linearGradient>
        <linearGradient id="config-laser-purple">
          <stop offset="0%" style="stop-color:rgba(139, 92, 246, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#8b5cf6; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(139, 92, 246, 0); stop-opacity:0" />
        </linearGradient>
        <linearGradient id="config-laser-blue">
          <stop offset="0%" style="stop-color:rgba(59, 130, 246, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#3b82f6; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(59, 130, 246, 0); stop-opacity:0" />
        </linearGradient>
        <linearGradient id="config-laser-cyan">
          <stop offset="0%" style="stop-color:rgba(6, 182, 212, 0); stop-opacity:0" />
          <stop offset="50%" style="stop-color:#06b6d4; stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgba(6, 182, 212, 0); stop-opacity:0" />
        </linearGradient>
      </defs>

      <!-- Laser-like connections -->
      <g class="connections">
        <!-- File → Config -->
        <path d="M 120 110 L 120 175 L 200 175" stroke="rgba(251, 191, 36, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 120 110 L 120 175 L 200 175" stroke="url(#config-laser-yellow)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2s" repeatCount="indefinite"/>
        </path>

        <!-- Consul → Config -->
        <path d="M 250 110 L 250 150" stroke="rgba(139, 92, 246, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 250 110 L 250 150" stroke="url(#config-laser-purple)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2.2s" repeatCount="indefinite"/>
        </path>

        <!-- ZooKeeper → Config -->
        <path d="M 380 110 L 380 175 L 300 175" stroke="rgba(59, 130, 246, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 380 110 L 380 175 L 300 175" stroke="url(#config-laser-blue)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="2.4s" repeatCount="indefinite"/>
        </path>

        <!-- Config → Runtime -->
        <path d="M 250 200 L 250 250" stroke="rgba(6, 182, 212, 0.3)" stroke-width="2" fill="none"/>
        <path d="M 250 200 L 250 250" stroke="url(#config-laser-cyan)" stroke-width="1.5" fill="none">
          <animate attributeName="opacity" values="0.5;1;0.5" dur="1.8s" repeatCount="indefinite"/>
        </path>
        <text x="275" y="225" text-anchor="middle" fill="rgba(255, 255, 255, 0.9)"
              font-size="11" font-weight="600" font-family="Inter, sans-serif">Hot Reload</text>
      </g>

      <!-- Animated light particles -->
      <g class="particles">
        <circle r="3" fill="#fbbf24" filter="url(#config-glow-orange)">
          <animateMotion dur="4s" repeatCount="indefinite" path="M 120 110 L 120 175 L 200 175 L 250 175 L 250 200 L 250 250"/>
        </circle>
        <circle r="3" fill="#8b5cf6" filter="url(#config-glow-purple)">
          <animateMotion dur="4.5s" repeatCount="indefinite" path="M 250 110 L 250 150 L 250 175 L 250 200 L 250 250"/>
        </circle>
        <circle r="3" fill="#3b82f6" filter="url(#config-glow-blue)">
          <animateMotion dur="5s" repeatCount="indefinite" path="M 380 110 L 380 175 L 300 175 L 250 175 L 250 200 L 250 250"/>
        </circle>
      </g>

      <!-- 3D Neon Cube Nodes -->
      <g class="nodes">
        <!-- File - 3D Cube -->
        <g class="node" id="config-node-file">
          <path d="M 78,63 L 165,63 L 171,57 L 84,57 Z" fill="rgba(251, 191, 36, 0.4)"/>
          <path d="M 78,63 L 84,57 L 84,104 L 78,110 Z" fill="rgba(251, 191, 36, 0.6)"/>
          <rect x="78" y="63" width="87" height="47" rx="4"
                fill="rgba(251, 191, 36, 0.15)"
                stroke="#fbbf24" stroke-width="2" filter="url(#config-glow-orange)"/>
          <text x="121" y="90" text-anchor="middle" fill="#fbbf24"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">File</text>
        </g>

        <!-- Consul - 3D Cube -->
        <g class="node" id="config-node-consul">
          <path d="M 208,63 L 292,63 L 298,57 L 214,57 Z" fill="rgba(139, 92, 246, 0.4)"/>
          <path d="M 208,63 L 214,57 L 214,104 L 208,110 Z" fill="rgba(139, 92, 246, 0.6)"/>
          <rect x="208" y="63" width="84" height="47" rx="4"
                fill="rgba(139, 92, 246, 0.15)"
                stroke="#8b5cf6" stroke-width="2" filter="url(#config-glow-purple)"/>
          <text x="250" y="90" text-anchor="middle" fill="#8b5cf6"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">Consul</text>
        </g>

        <!-- ZooKeeper - 3D Cube -->
        <g class="node" id="config-node-zk">
          <path d="M 338,63 L 422,63 L 428,57 L 344,57 Z" fill="rgba(59, 130, 246, 0.4)"/>
          <path d="M 338,63 L 344,57 L 344,104 L 338,110 Z" fill="rgba(59, 130, 246, 0.6)"/>
          <rect x="338" y="63" width="84" height="47" rx="4"
                fill="rgba(59, 130, 246, 0.15)"
                stroke="#3b82f6" stroke-width="2" filter="url(#config-glow-blue)"/>
          <text x="380" y="83" text-anchor="middle" fill="#3b82f6"
                font-size="12" font-weight="700" font-family="Inter, sans-serif">Zoo</text>
          <text x="380" y="98" text-anchor="middle" fill="#3b82f6"
                font-size="12" font-weight="700" font-family="Inter, sans-serif">Keeper</text>
        </g>

        <!-- Config - 3D Cube -->
        <g class="node" id="config-node-config">
          <path d="M 208,153 L 292,153 L 298,147 L 214,147 Z" fill="rgba(6, 182, 212, 0.4)"/>
          <path d="M 208,153 L 214,147 L 214,194 L 208,200 Z" fill="rgba(6, 182, 212, 0.6)"/>
          <rect x="208" y="153" width="84" height="47" rx="4"
                fill="rgba(6, 182, 212, 0.15)"
                stroke="#06b6d4" stroke-width="2" filter="url(#config-glow-cyan)"/>
          <text x="250" y="180" text-anchor="middle" fill="#06b6d4"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">Config</text>
        </g>

        <!-- Runtime - 3D Cube -->
        <g class="node" id="config-node-runtime">
          <path d="M 208,253 L 292,253 L 298,247 L 214,247 Z" fill="rgba(6, 182, 212, 0.4)"/>
          <path d="M 208,253 L 214,247 L 214,294 L 208,300 Z" fill="rgba(6, 182, 212, 0.6)"/>
          <rect x="208" y="253" width="84" height="47" rx="4"
                fill="rgba(6, 182, 212, 0.15)"
                stroke="#06b6d4" stroke-width="2" filter="url(#config-glow-cyan)"/>
          <text x="250" y="280" text-anchor="middle" fill="#06b6d4"
                font-size="14" font-weight="700" font-family="Inter, sans-serif">Runtime</text>
        </g>
      </g>
    </svg>
    </div>
  </div>
</div>

<!-- Programmatic API Story Section -->
<div class="story-section">
  <div class="story-container">
    <div class="story-text">
      <h2 class="story-title">Programmatic API</h2>
      <p>
        When you need to customize agent behavior, integrate with existing systems, or build dynamic agent workflows, Hector's <strong>programmatic API</strong> gives you full control.
      </p>
      <p>
        Build agents programmatically, combine them with config-based agents, or embed agents directly into your Go applications. The configuration system is built on top of the programmatic API, meaning you can mix and match seamlessly.
      </p>
    </div>
    <div class="story-visual">
      <div class="programmatic-code-block">
```go
// Load agents from YAML configuration
cfg, _ := config.LoadConfig(config.LoaderOptions{
    Path: "configs/agents.yaml",
})

// Build agents from config
configBuilder, _ := hector.NewConfigAgentBuilder(cfg)
configAgents, _ := configBuilder.BuildAllAgents()

// Build a custom agent programmatically
llm, _ := hector.NewLLMProvider("openai").
    Model("gpt-4o-mini").
    APIKeyFromEnv("OPENAI_API_KEY").
    Build()

programmaticAgent, _ := hector.NewAgent("custom").
    WithName("Custom Agent").
    WithLLMProvider(llm).
    Build()

// Combine config-based and programmatic agents
runtime.NewRuntimeBuilder().
    WithAgents(configAgents).
    WithAgent(programmaticAgent).
    Start()
```
      </div>
    </div>
  </div>
</div>

<div class="features-grid-section">
  <h2 class="section-header">Everything you need for production</h2>
</div>

<div class="grid cards" markdown="1">

-   :zap: __Zero-Code Configuration__

    Pure YAML agent definition. No Python/Go required—define sophisticated agents declaratively.

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
