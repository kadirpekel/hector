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
  <div class="a2a-story-content">
    <h2 class="a2a-story-title">A2A-Native Federation</h2>
    <p class="a2a-story-text">
      Hector is built on the <strong>Agent-to-Agent (A2A) protocol</strong> — an open standard for agent communication. Unlike traditional systems with central orchestrators that create bottlenecks and single points of failure, Hector enables true peer-to-peer federation.
    </p>
    <p class="a2a-story-text">
      Deploy agents across different services, languages, and infrastructures. They communicate directly using the A2A protocol, forming federated networks where each agent maintains autonomy while collaborating seamlessly — no central control, no lock-in.
    </p>
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
  
  <div class="a2a-visualization">
    <canvas id="a2a-federation"></canvas>
  </div>
</div>

<script>
(function() {
  const canvas = document.getElementById('a2a-federation');
  if (!canvas) return;
  
  const ctx = canvas.getContext('2d');
  const dpr = window.devicePixelRatio || 1;
  
  // Responsive sizing
  function resizeCanvas() {
    const container = canvas.parentElement;
    const size = Math.min(container.clientWidth, container.clientHeight || 500);
    canvas.style.width = size + 'px';
    canvas.style.height = size + 'px';
    canvas.width = size * dpr;
    canvas.height = size * dpr;
    ctx.scale(dpr, dpr);
  }
  
  resizeCanvas();
  window.addEventListener('resize', resizeCanvas);
  
  // Box dimensions
  const boxWidth = 80;
  const boxHeight = 50;
  const boxRadius = 12;
  
  // Connections: A↔B (horizontal), A→C (L-shape), B→C (L-shape)
  const connections = [
    { from: 0, to: 1, type: 'horizontal', color: 'rgba(237, 66, 103, 0.5)', label: 'A2A Protocol' },
    { from: 0, to: 2, type: 'lshape', color: 'rgba(237, 66, 103, 0.5)', label: 'A2A' },
    { from: 1, to: 2, type: 'lshape', color: 'rgba(16, 185, 129, 0.5)', label: 'A2A' }
  ];
  
  // Animated dots
  const dots = [];
  connections.forEach((conn, i) => {
    dots.push({
      connection: conn,
      progress: i / connections.length,
      speed: 0.008
    });
  });
  
  let animationFrame;
  
  function getNodes() {
    const w = canvas.width / dpr;
    const h = canvas.height / dpr;
    const padding = 40;
    const topY = padding + boxHeight / 2;
    const bottomY = h - padding - boxHeight / 2;
    const leftX = padding + boxWidth / 2;
    const rightX = w - padding - boxWidth / 2;
    const centerX = w / 2; // Bottom-middle
    
    // Triangle layout: A (top-left), B (top-right), C (bottom-middle)
    return [
      { x: leftX, y: topY, label: 'Agent A', color: '#ed4267' },
      { x: rightX, y: topY, label: 'Agent B', color: '#10b981' },
      { x: centerX, y: bottomY, label: 'Agent C', color: '#ed4267' }
    ];
  }
  
  // Draw 90-degree cornered connection forming rounded rectangle
  function drawCorneredConnection(ctx, from, to, conn) {
    ctx.strokeStyle = conn.color;
    ctx.lineWidth = 2.5;
    ctx.setLineDash([]);
    ctx.beginPath();
    
    if (conn.type === 'horizontal') {
      // A ↔ B: Connect from left side of A to right side of B at middle
      const leftSideOfA = from.x - boxWidth / 2;
      const rightSideOfB = to.x + boxWidth / 2;
      const y = from.y; // Middle (vertically centered)
      
      ctx.moveTo(leftSideOfA, y);
      ctx.lineTo(rightSideOfB, y);
      
      // Draw label above
      if (conn.label) {
        ctx.fillStyle = 'rgba(255, 255, 255, 0.7)';
        ctx.font = '10px sans-serif';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(conn.label, (leftSideOfA + rightSideOfB) / 2, y - 15);
      }
    } else if (conn.type === 'lshape') {
      // A → C or B → C: Start from bottom middle, go down, then turn to connect to C at middle
      const bottomMiddleX = from.x; // Center X of the box
      const bottomOfFrom = from.y + boxHeight / 2;
      
      if (from.x < to.x) {
        // A → C: down from bottom middle of A, turn right, connect to left side of C at middle
        const leftSideOfC = to.x - boxWidth / 2;
        const middleY = to.y; // Middle (vertically centered) of C
        
        ctx.moveTo(bottomMiddleX, bottomOfFrom);
        ctx.lineTo(bottomMiddleX, middleY);
        ctx.lineTo(leftSideOfC, middleY);
        
        // Draw label above horizontal segment
        if (conn.label) {
          ctx.fillStyle = 'rgba(255, 255, 255, 0.7)';
          ctx.font = '10px sans-serif';
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle';
          const labelX = (bottomMiddleX + leftSideOfC) / 2;
          ctx.fillText(conn.label, labelX, middleY - 15);
        }
      } else {
        // B → C: down from bottom middle of B, turn left, connect to right side of C at middle
        const rightSideOfC = to.x + boxWidth / 2;
        const middleY = to.y; // Middle (vertically centered) of C
        
        ctx.moveTo(bottomMiddleX, bottomOfFrom);
        ctx.lineTo(bottomMiddleX, middleY);
        ctx.lineTo(rightSideOfC, middleY);
        
        // Draw label above horizontal segment
        if (conn.label) {
          ctx.fillStyle = 'rgba(255, 255, 255, 0.7)';
          ctx.font = '10px sans-serif';
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle';
          const labelX = (bottomMiddleX + rightSideOfC) / 2;
          ctx.fillText(conn.label, labelX, middleY - 15);
        }
      }
    }
    
    ctx.stroke();
  }
  
  // Get point along 90-degree cornered path
  function getPointOnCorneredPath(from, to, progress, conn) {
    if (conn.type === 'horizontal') {
      // A ↔ B: horizontal from left side of A to right side of B at middle
      const leftSideOfA = from.x - boxWidth / 2;
      const rightSideOfB = to.x + boxWidth / 2;
      const y = from.y; // Middle (vertically centered)
      return {
        x: leftSideOfA + (rightSideOfB - leftSideOfA) * progress,
        y: y
      };
    } else if (conn.type === 'lshape') {
      const bottomMiddleX = from.x; // Center X of the box
      const bottomOfFrom = from.y + boxHeight / 2;
      
      if (from.x < to.x) {
        // A → C: down from bottom middle, turn right to left side of C at middle
        const leftSideOfC = to.x - boxWidth / 2;
        const middleY = to.y; // Middle (vertically centered) of C
        
        const verticalLength = Math.abs(middleY - bottomOfFrom);
        const horizontalLength = Math.abs(leftSideOfC - bottomMiddleX);
        const totalLength = verticalLength + horizontalLength;
        
        const vProgress = verticalLength / totalLength;
        
        if (progress < vProgress) {
          // Vertical segment: down from bottom middle
          const t = progress / vProgress;
          return { x: bottomMiddleX, y: bottomOfFrom + (middleY - bottomOfFrom) * t };
        } else {
          // Horizontal segment: right to C's left side
          const t = (progress - vProgress) / (1 - vProgress);
          return { x: bottomMiddleX + (leftSideOfC - bottomMiddleX) * t, y: middleY };
        }
      } else {
        // B → C: down from bottom middle, turn left to right side of C at middle
        const rightSideOfC = to.x + boxWidth / 2;
        const middleY = to.y; // Middle (vertically centered) of C
        
        const verticalLength = Math.abs(middleY - bottomOfFrom);
        const horizontalLength = Math.abs(rightSideOfC - bottomMiddleX);
        const totalLength = verticalLength + horizontalLength;
        
        const vProgress = verticalLength / totalLength;
        
        if (progress < vProgress) {
          // Vertical segment: down from bottom middle
          const t = progress / vProgress;
          return { x: bottomMiddleX, y: bottomOfFrom + (middleY - bottomOfFrom) * t };
        } else {
          // Horizontal segment: left to C's right side
          const t = (progress - vProgress) / (1 - vProgress);
          return { x: bottomMiddleX + (rightSideOfC - bottomMiddleX) * t, y: middleY };
        }
      }
    }
  }
  
  function draw() {
    const w = canvas.width / dpr;
    const h = canvas.height / dpr;
    const nodes = getNodes();
    
    // Clear canvas
    ctx.clearRect(0, 0, w, h);
    
    // Draw subtle white transparent background
    ctx.fillStyle = 'rgba(255, 255, 255, 0.03)';
    roundRect(ctx, 0, 0, w, h, 12);
    ctx.fill();
    
    // Draw connections with 90-degree corners
    connections.forEach(conn => {
      const from = nodes[conn.from];
      const to = nodes[conn.to];
      drawCorneredConnection(ctx, from, to, conn);
    });
    
    // Draw animated dots along cornered paths
    dots.forEach((dot) => {
      const conn = dot.connection;
      const from = nodes[conn.from];
      const to = nodes[conn.to];
      
      dot.progress += dot.speed;
      if (dot.progress >= 1) dot.progress = 0;
      
      const point = getPointOnCorneredPath(from, to, dot.progress, conn);
      
      // Dot with subtle glow
      const gradient = ctx.createRadialGradient(point.x, point.y, 0, point.x, point.y, 8);
      gradient.addColorStop(0, conn.color.replace('0.5', '1'));
      gradient.addColorStop(0.5, conn.color.replace('0.5', '0.6'));
      gradient.addColorStop(1, conn.color.replace('0.5', '0'));
      
      ctx.fillStyle = gradient;
      ctx.beginPath();
      ctx.arc(point.x, point.y, 8, 0, 2 * Math.PI);
      ctx.fill();
      
      // Core dot
      ctx.fillStyle = conn.color.replace('0.5', '1');
      ctx.beginPath();
      ctx.arc(point.x, point.y, 4, 0, 2 * Math.PI);
      ctx.fill();
    });
    
    // Draw agent boxes - bigger, slick style
    nodes.forEach(node => {
      const x = node.x - boxWidth / 2;
      const y = node.y - boxHeight / 2;
      
      // Box shadow
      ctx.fillStyle = 'rgba(0, 0, 0, 0.2)';
      roundRect(ctx, x + 2, y + 2, boxWidth, boxHeight, boxRadius);
      ctx.fill();
      
      // Main box with gradient
      const gradient = ctx.createLinearGradient(x, y, x, y + boxHeight);
      gradient.addColorStop(0, node.color);
      gradient.addColorStop(1, adjustColor(node.color, -20));
      ctx.fillStyle = gradient;
      roundRect(ctx, x, y, boxWidth, boxHeight, boxRadius);
      ctx.fill();
      
      // Subtle border
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
      ctx.lineWidth = 1;
      roundRect(ctx, x, y, boxWidth, boxHeight, boxRadius);
      ctx.stroke();
      
      // Label
      ctx.fillStyle = 'rgba(255, 255, 255, 0.95)';
      ctx.font = 'bold 12px sans-serif';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(node.label, node.x, node.y);
    });
    
    animationFrame = requestAnimationFrame(draw);
  }
  
  // Helper for rounded rectangles
  function roundRect(ctx, x, y, width, height, radius) {
    ctx.beginPath();
    ctx.moveTo(x + radius, y);
    ctx.lineTo(x + width - radius, y);
    ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
    ctx.lineTo(x + width, y + height - radius);
    ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
    ctx.lineTo(x + radius, y + height);
    ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
    ctx.lineTo(x, y + radius);
    ctx.quadraticCurveTo(x, y, x + radius, y);
    ctx.closePath();
  }
  
  // Helper to adjust color brightness
  function adjustColor(color, amount) {
    const num = parseInt(color.replace('#', ''), 16);
    const r = Math.max(0, Math.min(255, (num >> 16) + amount));
    const g = Math.max(0, Math.min(255, ((num >> 8) & 0x00FF) + amount));
    const b = Math.max(0, Math.min(255, (num & 0x0000FF) + amount));
    return '#' + ((r << 16) | (g << 8) | b).toString(16).padStart(6, '0');
  }
  
  draw();
  
  // Cleanup
  window.addEventListener('beforeunload', () => {
    if (animationFrame) cancelAnimationFrame(animationFrame);
  });
})();
</script>

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

