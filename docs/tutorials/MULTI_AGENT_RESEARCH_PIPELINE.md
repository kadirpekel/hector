---
layout: default
title: "LangChain vs Hector"
nav_order: 1
parent: Tutorials
description: "See how Hector transforms complex LangChain multi-agent implementations into simple YAML configuration"
---

# Building Multi-Agent Systems: LangChain vs Hector

**TL;DR:** See how Hector transforms complex LangChain multi-agent implementations into simple YAML configuration. What takes 500+ lines of Python code in LangChain becomes 120 lines of YAML in Hector - same functionality, dramatically simpler approach.

---

## Table of Contents

- [Why This Comparison Matters](#why-this-comparison-matters)
- [The Challenge: Multi-Agent Research System](#the-challenge-multi-agent-research-system)
- [LangChain Implementation: The Traditional Way](#langchain-implementation-the-traditional-way)
- [Hector Implementation: The Simple Way](#hector-implementation-the-simple-way)
- [The Dramatic Difference](#the-dramatic-difference)
- [Scaling Beyond the Basics](#scaling-beyond-the-basics)
- [Why Hector Wins](#why-hector-wins)
- [Get Started Today](#get-started-today)

---

## Why This Comparison Matters

**LangChain** is the most popular framework for building AI agent systems - and for good reason. It's powerful, flexible, and has a huge ecosystem. But it comes with a cost: **complexity**.

**Hector** takes a different approach. Instead of requiring you to write hundreds of lines of Python code, Hector lets you define the same sophisticated multi-agent systems in pure YAML configuration.

**This tutorial shows you exactly how much simpler Hector makes multi-agent development** by implementing the same research assistant system in both frameworks.

**What you'll see:**
- ✅ **LangChain approach** - 500+ lines across 8+ Python files
- ✅ **Hector approach** - 120 lines of YAML in a single file
- ✅ **Same functionality** - Identical agent behavior and capabilities
- ✅ **Dramatic simplicity gain** - Focus on what matters, not boilerplate

---

## The Challenge: Multi-Agent Research System

Let's build a practical multi-agent system that many teams need: **an automated research assistant**.

**Requirements:**
1. **Coordinator** - Orchestrates the workflow and manages other agents
2. **Researcher** - Gathers information from web sources
3. **Analyst** - Analyzes findings and identifies key insights
4. **Writer** - Creates structured reports and saves them

**Workflow:** User query → Researcher gathers data → Analyst finds insights → Writer creates report → Coordinator delivers result

This is a perfect example because it's:
- **Real-world useful** - Teams actually need this
- **Multi-agent** - Requires coordination between specialists
- **Tool integration** - Web search, file operations
- **Complex enough** - Shows the frameworks' true differences

---

## LangChain Implementation: The Traditional Way

Let's start with how you'd build this in LangChain. This is the "standard" approach that most developers follow:

### Project Structure (8+ Files)

```
research_assistant/
├── requirements.txt          # Dependencies
├── config.yaml              # Configuration
├── main.py                  # Application entry point
├── agents/
│   ├── __init__.py
│   ├── coordinator.py       # ~120 lines
│   ├── researcher.py        # ~90 lines
│   ├── analyst.py           # ~80 lines
│   └── writer.py            # ~85 lines
├── tools/
│   ├── __init__.py
│   └── web_search.py        # ~60 lines
├── state/
│   ├── __init__.py
│   └── research_state.py    # ~50 lines
└── workflow.py              # ~100 lines
```

### State Management (state/research_state.py)

```python
from typing import List, Dict, Any, Optional
from pydantic import BaseModel, Field
from typing_extensions import TypedDict

class ResearchSource(BaseModel):
    """Represents a research source with metadata."""
    url: str
    title: str
    content: str
    credibility_score: float = Field(ge=0.0, le=1.0)
    date_accessed: str
    source_type: str

class ResearchState(TypedDict):
    """Complete state for the research pipeline."""
    query: str
    research_data: str
    analysis: str
    final_report: str
    current_step: str
    sources: List[ResearchSource]
    errors: List[str]
    metadata: Dict[str, Any]
```

### Coordinator Agent (agents/coordinator.py)

```python
from typing import Dict, Any
from langchain_core.prompts import ChatPromptTemplate
from langchain_openai import ChatOpenAI
from langgraph import StateGraph, END
from state.research_state import ResearchState

class ResearchCoordinator:
    """Coordinates the multi-agent research pipeline."""
    
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.llm = ChatOpenAI(
            model=config["llms"]["coordinator"]["model"],
            temperature=config["llms"]["coordinator"]["temperature"],
            max_tokens=config["llms"]["coordinator"]["max_tokens"]
        )
        
        self.coordination_prompt = ChatPromptTemplate.from_messages([
            ("system", """You are a research coordinator managing specialized agents.
            
            AVAILABLE AGENTS:
            - researcher: Gathers information from web sources
            - analyst: Analyzes findings and identifies insights  
            - writer: Creates structured reports
            
            Coordinate the workflow: researcher → analyst → writer → deliver results.
            Build on previous results and ensure quality at each step.
            """),
            ("human", "Research Query: {query}\n\nCurrent State: {state}")
        ])
    
    def coordinate_research(self, state: ResearchState) -> Dict[str, Any]:
        """Main coordination logic."""
        try:
            coordination_chain = self.coordination_prompt | self.llm
            
            response = coordination_chain.invoke({
                "query": state["query"],
                "state": state["current_step"]
            })
            
            # Parse response and determine next action
            next_step = self._determine_next_step(state, response.content)
            
            return {
                "current_step": next_step,
                "metadata": {
                    **state.get("metadata", {}),
                    "coordinator_decision": response.content
                }
            }
            
        except Exception as e:
            return {
                "current_step": "error",
                "errors": state.get("errors", []) + [str(e)]
            }
    
    def _determine_next_step(self, state: ResearchState, response: str) -> str:
        """Determine the next step based on current state."""
        current = state["current_step"]
        
        if current == "starting":
            return "research"
        elif current == "research_complete":
            return "analyze"
        elif current == "analysis_complete":
            return "write"
        elif current == "writing_complete":
            return "complete"
        else:
            return "error"
    
    def should_continue(self, state: ResearchState) -> str:
        """Determine if workflow should continue."""
        step = state["current_step"]
        
        if step == "complete":
            return END
        elif step == "error":
            return END
        elif step == "research":
            return "research"
        elif step == "analyze":
            return "analyze"
        elif step == "write":
            return "write"
        else:
            return "coordinate"
```

### Research Agent (agents/researcher.py)

```python
from typing import Dict, Any, List
from langchain_core.prompts import ChatPromptTemplate
from langchain_openai import ChatOpenAI
from tools.web_search import WebSearchTool
from state.research_state import ResearchState, ResearchSource

class ResearchAgent:
    """Specialized agent for information gathering."""
    
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.llm = ChatOpenAI(
            model=config["llms"]["worker"]["model"],
            temperature=config["llms"]["worker"]["temperature"],
            max_tokens=config["llms"]["worker"]["max_tokens"]
        )
        
        self.web_search = WebSearchTool(config["tools"]["web_search"])
        
        self.research_prompt = ChatPromptTemplate.from_messages([
            ("system", """You are a research specialist who gathers comprehensive information.
            
            RESEARCH PROCESS:
            1. Break down the query into searchable components
            2. Use web search tools to gather information
            3. Evaluate source quality and relevance
            4. Extract key information and insights
            5. Organize findings by theme/category
            6. Provide structured output with citations
            
            Focus on accuracy, comprehensiveness, and source diversity.
            Always include source URLs and assess credibility.
            """),
            ("human", "Research Query: {query}\n\nPrevious Context: {context}")
        ])
    
    def conduct_research(self, state: ResearchState) -> Dict[str, Any]:
        """Conduct comprehensive research on the query."""
        try:
            query = state["query"]
            context = state.get("metadata", {})
            
            # Perform web searches
            search_results = self.web_search.search(query, max_results=10)
            
            # Process results with LLM
            research_chain = self.research_prompt | self.llm
            
            analysis = research_chain.invoke({
                "query": query,
                "context": str(context)
            })
            
            # Convert results to structured format
            sources = []
            for result in search_results:
                source = ResearchSource(
                    url=result.get("url", ""),
                    title=result.get("title", ""),
                    content=result.get("content", ""),
                    credibility_score=self._assess_credibility(result),
                    date_accessed=self._get_current_date(),
                    source_type="web"
                )
                sources.append(source)
            
            return {
                "research_data": analysis.content,
                "sources": sources,
                "current_step": "research_complete",
                "metadata": {
                    **state.get("metadata", {}),
                    "research_timestamp": self._get_current_date(),
                    "sources_found": len(sources)
                }
            }
            
        except Exception as e:
            return {
                "current_step": "error",
                "errors": state.get("errors", []) + [f"Research error: {str(e)}"]
            }
    
    def _assess_credibility(self, result: Dict[str, Any]) -> float:
        """Assess source credibility based on various factors."""
        # Implement credibility scoring logic
        base_score = 0.5
        
        # Check domain reputation
        if any(domain in result.get("url", "") for domain in [".edu", ".gov", ".org"]):
            base_score += 0.3
        
        # Check for HTTPS
        if result.get("url", "").startswith("https"):
            base_score += 0.1
        
        # Check content length (longer content often more credible)
        content_length = len(result.get("content", ""))
        if content_length > 1000:
            base_score += 0.1
        
        return min(base_score, 1.0)
    
    def _get_current_date(self) -> str:
        """Get current date in ISO format."""
        from datetime import datetime
        return datetime.now().isoformat()
```

### Workflow Orchestration (workflow.py)

```python
from typing import Dict, Any
from langgraph import StateGraph, END
from state.research_state import ResearchState
from agents.coordinator import ResearchCoordinator
from agents.researcher import ResearchAgent
from agents.analyst import AnalystAgent
from agents.writer import WriterAgent

class ResearchWorkflow:
    """Complete multi-agent research workflow using LangGraph."""
    
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        
        # Initialize all agents
        self.coordinator = ResearchCoordinator(config)
        self.researcher = ResearchAgent(config)
        self.analyst = AnalystAgent(config)
        self.writer = WriterAgent(config)
        
        # Build the workflow graph
        self.workflow = self._build_workflow()
    
    def _build_workflow(self) -> StateGraph:
        """Build the LangGraph workflow."""
        workflow = StateGraph(ResearchState)
        
        # Add nodes for each agent
        workflow.add_node("coordinate", self.coordinator.coordinate_research)
        workflow.add_node("research", self.researcher.conduct_research)
        workflow.add_node("analyze", self.analyst.analyze_findings)
        workflow.add_node("write", self.writer.create_report)
        
        # Define the entry point
        workflow.set_entry_point("coordinate")
        
        # Add conditional edges based on coordinator decisions
        workflow.add_conditional_edges(
            "coordinate",
            self.coordinator.should_continue,
            {
                "research": "research",
                "analyze": "analyze", 
                "write": "write",
                END: END
            }
        )
        
        # Add edges back to coordinator for next step
        workflow.add_edge("research", "coordinate")
        workflow.add_edge("analyze", "coordinate")
        workflow.add_edge("write", "coordinate")
        
        return workflow.compile()
    
    def run_research(self, query: str) -> Dict[str, Any]:
        """Execute the complete research workflow."""
        initial_state = {
            "query": query,
            "research_data": "",
            "analysis": "",
            "final_report": "",
            "current_step": "starting",
            "sources": [],
            "errors": [],
            "metadata": {}
        }
        
        try:
            result = self.workflow.invoke(initial_state)
            return result
        except Exception as e:
            return {
                **initial_state,
                "current_step": "error",
                "errors": [f"Workflow error: {str(e)}"]
            }
```

### Main Application (main.py)

```python
import yaml
import argparse
from typing import Dict, Any
from workflow import ResearchWorkflow

class ResearchApp:
    """Main application for the LangChain research pipeline."""
    
    def __init__(self, config_path: str = "config.yaml"):
        with open(config_path, 'r') as f:
            self.config = yaml.safe_load(f)
        
        self.workflow = ResearchWorkflow(self.config)
    
    def research(self, query: str) -> str:
        """Conduct research and return final report."""
        result = self.workflow.run_research(query)
        
        if result["current_step"] == "error":
            return f"Research failed: {', '.join(result['errors'])}"
        
        return result.get("final_report", "No report generated")

def main():
    """CLI interface for the research pipeline."""
    parser = argparse.ArgumentParser(description="Multi-Agent Research Pipeline")
    parser.add_argument("query", help="Research query")
    parser.add_argument("--config", default="config.yaml", help="Config file path")
    
    args = parser.parse_args()
    
    try:
        app = ResearchApp(args.config)
        result = app.research(args.query)
        print(result)
    except Exception as e:
        print(f"Application error: {e}")

if __name__ == "__main__":
    main()
```

### Configuration File (config.yaml)

```yaml
llms:
  coordinator:
    model: "gpt-4o"
    temperature: 0.3
    max_tokens: 4000
  
  worker:
    model: "gpt-4o-mini"
    temperature: 0.5
    max_tokens: 3000

tools:
  web_search:
    enabled: true
    max_results: 10
    timeout: 30

workflow:
  max_iterations: 20
  enable_logging: true
```

### Dependencies (requirements.txt)

```txt
langchain>=0.1.0
langchain-openai>=0.1.0
langgraph>=0.1.0
pydantic>=2.5.0
requests>=2.31.0
beautifulsoup4>=4.12.0
python-dotenv>=1.0.0
```

### What You Need to Do

**To run this LangChain system:**

1. **Set up Python environment** (15+ minutes)
2. **Install 7+ dependencies** with potential version conflicts
3. **Write 500+ lines of code** across 8+ files
4. **Handle state management** manually with TypedDict and Pydantic
5. **Implement error handling** for each agent and workflow step
6. **Debug complex workflows** when something goes wrong
7. **Manage deployment** with Python packaging and dependencies

**Total setup time:** 2-3 hours for an experienced developer

## Hector Implementation: The Simple Way

Now let's see how Hector solves the exact same problem. Same functionality, dramatically different approach.

### Complete Working System (Single File)

```yaml
# research-assistant.yaml - Everything you need in 120 lines
agents:
  # Orchestrator coordinates the workflow  
  coordinator:
    name: "Research Coordinator"
    description: "Coordinates multi-agent research workflow"
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"  # Built-in orchestration magic
      max_iterations: 10
      enable_streaming: true
    tools: ["agent_call"]
    prompt:
      system_role: |
        You coordinate research using 3 specialists:
        researcher → analyst → writer
        
        WORKFLOW:
        1. Call researcher with the query to gather information
        2. Call analyst with research data to identify insights  
        3. Call writer with analysis to create structured report
        
        Always build on previous results and ensure quality.
      
      reasoning_instructions: |
        1. Break down the research request into clear steps
        2. Call each agent in sequence: researcher → analyst → writer
        3. Pass results between agents to build comprehensive output
        4. Verify each step completed successfully before proceeding

  # Specialist agents with focused roles
  researcher:
    name: "Web Researcher"
    description: "Gathers information from web sources"
    llm: "gpt-4o-mini"
    tools: ["execute_command"]
    prompt:
      system_role: |
        You are a research specialist who gathers comprehensive information.
        Use web searches to find relevant, credible sources.
        
        RESEARCH PROCESS:
        1. Break down the query into searchable components
        2. Use curl commands to search multiple sources
        3. Evaluate source quality and relevance
        4. Extract key information and insights
        5. Organize findings by theme/category
        6. Provide structured output with citations
        
        Focus on accuracy, comprehensiveness, and source diversity.
        Always include source URLs and assess credibility.

  analyst:
    name: "Research Analyst"
    description: "Analyzes research findings for insights"
    llm: "gpt-4o-mini"
    prompt:
      system_role: |
        You analyze research findings to identify key themes, insights, and implications.
        
        ANALYSIS PROCESS:
        1. Review all research data comprehensively
        2. Identify patterns, trends, and key themes
        3. Extract actionable insights and implications
        4. Highlight contradictions or gaps in data
        5. Synthesize findings into clear conclusions
        6. Prepare structured analysis for report writing
        
        Focus on depth, accuracy, and practical implications.

  writer:
    name: "Report Writer"
    description: "Creates structured reports from analysis"
    llm: "gpt-4o-mini"
    tools: ["write_file"]
    prompt:
      system_role: |
        You create well-structured, professional reports from research analysis.
        
        WRITING PROCESS:
        1. Structure content with clear headings and flow
        2. Include executive summary and key findings
        3. Present detailed analysis with supporting evidence
        4. Add actionable recommendations
        5. Include proper citations and sources
        6. Save final report to file
        
        Use professional tone, clear formatting, and comprehensive coverage.

# LLM Configuration
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.3
    max_tokens: 4000
    
  gpt-4o-mini:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.5
    max_tokens: 3000

# Tool Configuration
tools:
  execute_command:
    type: "command"
    allowed_commands: ["curl"]
    max_execution_time: "30s"
    
  write_file:
    type: "write_file"
    working_directory: "./reports"
    allowed_extensions: [".md", ".txt", ".json"]
    max_file_size: "10MB"
```

### Running Your System

```bash
# Start Hector (takes 2 minutes to set up)
hector serve --config research-assistant.yaml

# Use your research assistant
hector call coordinator "Research sustainable AI development practices"
```

**What happens automatically:**
1. **Coordinator** receives your request and creates an execution plan
2. **Researcher** gathers information using web searches via curl
3. **Analyst** processes findings to identify key themes and insights  
4. **Writer** creates a structured report and saves it to `./reports/`
5. **You get** a comprehensive research report with sources and analysis

### The Full LangChain Implementation

For comparison, here's the complete LangChain code you'd need (abbreviated - see `/examples/langchain/` for full version):

```python
# Just the core files (8+ files total, 500+ lines)

# state/research_state.py - State management
class ResearchState(TypedDict):
    query: str
    research_data: str
    analysis: str
    final_report: str
    current_step: str
    sources: List[ResearchSource]
    errors: List[str]
    metadata: Dict[str, Any]

# agents/coordinator.py - Orchestration logic
class ResearchCoordinator:
    def coordinate_research(self, state: ResearchState) -> Dict[str, Any]:
        # 50+ lines of coordination logic
        coordination_chain = self.coordination_prompt | self.llm
        response = coordination_chain.invoke({"query": state["query"], "state": state["current_step"]})
        next_step = self._determine_next_step(state, response.content)
        # ... error handling, state updates, etc.

# workflow.py - LangGraph workflow  
class ResearchWorkflow:
    def _build_workflow(self) -> StateGraph:
        workflow = StateGraph(ResearchState)
        workflow.add_node("coordinate", self.coordinator.coordinate_research)
        workflow.add_node("research", self.researcher.conduct_research)
        workflow.add_node("analyze", self.analyst.analyze_findings)
        workflow.add_node("write", self.writer.create_report)
        workflow.set_entry_point("coordinate")
        workflow.add_conditional_edges("coordinate", self.coordinator.should_continue, {...})
        # ... more workflow configuration
        return workflow.compile()

# Plus: main.py, requirements.txt, config.yaml, tools/, agents/researcher.py, agents/analyst.py, agents/writer.py
```

---

## The Dramatic Difference

Let's break down what we just saw:

### Code Volume
- **LangChain:** 500+ lines across 8+ Python files
- **Hector:** 120 lines in 1 YAML file
- **Reduction:** 75% less code

### Setup Complexity
- **LangChain:** Python environment, 7+ dependencies, version management
- **Hector:** Single binary, zero dependencies
- **Time to run:** LangChain (2-3 hours) vs Hector (2 minutes)

### State Management
- **LangChain:** Manual TypedDict, Pydantic models, explicit state passing
- **Hector:** Automatic - handled by supervisor reasoning engine
- **You write:** Complex state logic vs simple agent definitions

### Error Handling
- **LangChain:** Manual try/catch in every method, custom error types
- **Hector:** Built-in error handling, automatic retries, graceful degradation
- **Debugging:** Stack traces vs clear agent logs

### Orchestration
- **LangChain:** LangGraph workflow, conditional edges, node management
- **Hector:** `reasoning: engine: "supervisor"` - done
- **Complexity:** 100+ lines of workflow code vs 3 lines of config

### Tool Integration
- **LangChain:** Custom tool classes, manual registration, parameter validation
- **Hector:** Built-in tools with simple configuration
- **Maintenance:** Custom code vs declarative config

### Deployment
- **LangChain:** Docker, Python packaging, dependency management
- **Hector:** Single binary, environment variables, done
- **Production:** Complex vs simple

**The bottom line:** Hector gives you the same sophisticated multi-agent capabilities with 75% less complexity. You focus on **what** your agents should do, not **how** to make them work together.

While other frameworks require you to write extensive Python code, Hector's declarative approach offers compelling benefits at every level:

### Configuration vs Programming

**What you write in other frameworks:**
```python
# Just the coordinator logic alone (100+ lines)
from langgraph import StateGraph, END
from typing import TypedDict

class ResearchState(TypedDict):
    query: str
    research_data: str
    analysis: str
    final_report: str
    current_step: str

class ResearchCoordinator:
    def __init__(self, config):
        self.config = config
        self.workflow = self._build_workflow()
    
    def _build_workflow(self):
        workflow = StateGraph(ResearchState)
        workflow.add_node("research", self.research_step)
        workflow.add_node("analyze", self.analyze_step)
        workflow.add_node("write", self.write_step)
        # ... 80+ more lines of workflow logic
        return workflow.compile()
    
    def research_step(self, state):
        # Custom implementation required
        pass
    # ... plus error handling, state management, etc.
```

**What you write in Hector:**
```yaml
coordinator:
  name: "Research Coordinator"
  llm: "gpt-4o"
  reasoning:
    engine: "supervisor"  # Built-in orchestration
  tools: ["agent_call"]
  prompt:
    system_role: |
      Coordinate research using: researcher → analyst → writer
```

### The Productivity Multiplier

| Task | Traditional Approach | Hector Approach |
|------|---------------------|-----------------|
| **Add new agent** | Create Python class, update workflow, test integration | Add 10 lines to YAML |
| **Change agent behavior** | Modify code, test, redeploy | Edit prompt in YAML |
| **Swap LLM providers** | Update imports, modify initialization | Change `type: "openai"` to `type: "anthropic"` |
| **Add error handling** | Implement try/catch blocks, retry logic | Already included |
| **Add monitoring** | Set up logging, metrics collection | Already included |
| **Deploy to production** | Package app, manage dependencies | Copy YAML file |

### Why This Matters

**For Development Teams:**
- **Faster iteration** - Changes take seconds, not hours
- **Lower barrier to entry** - YAML is accessible to more team members
- **Fewer bugs** - Less custom code means fewer places for errors
- **Better collaboration** - Non-programmers can contribute to agent logic

**For Operations Teams:**
- **Simpler deployment** - Single binary + config file
- **Easier troubleshooting** - Configuration is transparent and readable
- **Better security** - Built-in sandboxing and command whitelisting
- **Consistent patterns** - Same approach across all projects

---

## Scaling Up: Advanced Patterns

The beauty of Hector is that the same YAML approach scales naturally from simple to sophisticated systems.

### Adding Specialization

Need more sophisticated research? Just add specialized agents:

```yaml
agents:
  coordinator:
    # Same simple coordination logic
    
  web_researcher:
    name: "Web Research Specialist"
    tools: ["execute_command"]
    prompt:
      system_role: "Focus on web sources and recent information"
      
  academic_researcher:
    name: "Academic Research Specialist"
    document_stores: ["academic_papers"]
    tools: ["search"]
    prompt:
      system_role: "Focus on peer-reviewed academic sources"
      
  fact_checker:
    name: "Fact Verification Specialist"
    tools: ["execute_command", "search"]
    prompt:
      system_role: "Validate claims and check source credibility"
```

### Adding Intelligence with Document Stores

For domain-specific knowledge, add semantic search:

```yaml
# Add to your configuration
document_stores:
  company_knowledge:
    path: "./knowledge_base"
    source: "directory"
    include_patterns: ["*.md", "*.pdf"]
    
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334
    
embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"

# Agents automatically get access
agents:
  domain_expert:
    document_stores: ["company_knowledge"]
    tools: ["search"]
    prompt:
      system_role: "Use company knowledge to provide expert analysis"
```

### Adding Custom Capabilities with Plugins

Need custom functionality? Hector's plugin system has you covered:

```yaml
# Use custom LLM via plugin
plugins:
  llm_providers:
    custom_model:
      type: "grpc"
      path: "./plugins/custom-llm"
      
llms:
  specialized:
    type: "plugin:custom_model"
    model: "domain-specific-v1"

# Use custom tools via plugin  
plugins:
  tools:
    domain_tools:
      type: "grpc"
      path: "./plugins/domain-tools"
      
agents:
  specialist:
    tools: ["plugin:domain_tools"]
```

### Multi-Agent Orchestration Patterns

For complex workflows, Hector supports sophisticated orchestration:

```yaml
agents:
  supervisor:
    reasoning:
      engine: "supervisor"  # Hierarchical coordination
      max_iterations: 20
    tools: ["agent_call", "todo_write"]  # Task management
    
  parallel_coordinator:
    reasoning:
      engine: "chain-of-thought"
    tools: ["agent_call"]
    prompt:
      system_role: |
        Coordinate parallel research streams:
        - Call multiple researchers simultaneously
        - Synthesize results from parallel streams
        
  quality_controller:
    tools: ["agent_call"]
    prompt:
      system_role: |
        Review outputs and request improvements:
        - Validate completeness and accuracy
        - Request revisions if needed
        - Ensure quality standards
```

---

## Enterprise-Ready Features

Hector isn't just for prototypes - it's built for production from day one.

### Built-in Security

```yaml
tools:
  execute_command:
    type: command
    allowed_commands: ["curl", "jq"]  # Whitelist only
    working_directory: "/tmp/sandbox"  # Restricted access
    max_execution_time: "30s"  # Time limits
    enable_sandboxing: true  # Isolated execution
    
  write_file:
    type: write_file
    working_directory: "./safe_output"  # Path restrictions
    max_file_size: 10485760  # Size limits
```

### Monitoring and Observability

```yaml
# Automatic metrics at /metrics endpoint
# hector_requests_total{agent="coordinator"} 1250
# hector_request_duration_seconds{agent="coordinator"} 12.3
# hector_llm_tokens_total{provider="openai"} 45000

# Built-in structured logging
# 2025-01-01 10:00:00 INFO [coordinator] Starting research task
# 2025-01-01 10:00:01 INFO [researcher] Executing web search
# 2025-01-01 10:00:05 INFO [analyst] Processing research data
```

### A2A Protocol Compliance

```yaml
# Expose agents via standard A2A protocol
global:
  a2a_server:
    host: "0.0.0.0"
    port: 8080

# Agents automatically become A2A-compliant services
# curl -X POST http://localhost:8080/agents/coordinator/message/send
```

### High Availability Deployment

```yaml
# Kubernetes-ready configuration
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hector-research
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: hector
        image: hector:latest
        resources:
          requests:
            memory: "64Mi"    # Lightweight
            cpu: "100m"
        volumeMounts:
        - name: config
          mountPath: /config.yaml
          subPath: research-assistant.yaml
      volumes:
      - name: config
        configMap:
          name: hector-config
```

---

## Migration from Code-Based Frameworks

Moving from traditional frameworks to Hector is straightforward and offers immediate benefits.

### Common Migration Patterns

**From Custom Python Code:**
1. **Identify agent roles** - Map your classes to Hector agents
2. **Extract prompts** - Move system prompts to YAML configuration
3. **Map tools** - Replace custom tools with Hector's built-in tools
4. **Simplify orchestration** - Use supervisor reasoning instead of custom workflow logic

**From LangChain/LangGraph:**
1. **Map StateGraph nodes** - Each node becomes a Hector agent
2. **Convert state management** - Use Hector's automatic state handling
3. **Replace custom tools** - Most tools have Hector equivalents
4. **Simplify deployment** - Single binary replaces Python environment

### Migration Benefits

**Immediate Gains:**
- **Reduced complexity** - 80% less configuration to maintain
- **Faster deployment** - Minutes instead of hours
- **Better reliability** - Built-in error handling and recovery
- **Easier debugging** - Transparent YAML configuration

**Long-term Advantages:**
- **Team scalability** - More team members can contribute
- **Maintenance reduction** - Less custom code to maintain
- **Consistent patterns** - Same approach across all projects
- **Future-proofing** - Plugin system handles custom needs

### Gradual Migration Strategy

You don't need to migrate everything at once:

1. **Start with new projects** - Use Hector for new multi-agent systems
2. **Migrate simple workflows** - Convert straightforward agent chains first
3. **Add complexity gradually** - Use plugins for specialized needs
4. **Maintain hybrid systems** - Hector agents can call external services

---

## Get Started Today

Ready to experience the Hector advantage? Here's how to get started:

### Quick Start (5 minutes)

```bash
# 1. Install Hector
git clone https://github.com/kadirpekel/hector
cd hector && make build

# 2. Set up API key
export OPENAI_API_KEY="sk-..."

# 3. Try the research assistant
hector serve --config configs/research-assistant.yaml
hector call coordinator "Research quantum computing trends"
```

### Next Steps

1. **Experiment with the examples** - Try different research topics
2. **Customize the prompts** - Adapt agents for your specific needs  
3. **Add more agents** - Expand the pipeline with specialized roles
4. **Connect your data** - Set up document stores for domain knowledge
5. **Deploy to production** - Use the built-in A2A server

### Learning Path

**Beginner:** Start with the research assistant example
- Understand agent roles and coordination
- Experiment with different prompts
- Try various LLM providers

**Intermediate:** Add complexity and customization
- Set up document stores and semantic search
- Create specialized agent roles
- Implement quality control workflows

**Advanced:** Enterprise deployment and customization
- Deploy with Kubernetes
- Develop custom plugins
- Integrate with existing systems

### Community and Support

- **Documentation:** [Full configuration reference](../CONFIGURATION.md)
- **Examples:** [More agent patterns](../../configs/)
- **Community:** [GitHub Discussions](https://github.com/kadirpekel/hector/discussions)
- **Plugin Development:** [Plugin system guide](../PLUGINS.md)

---

## Conclusion

Multi-agent systems represent the future of AI applications, and Hector makes this future accessible today. By choosing configuration over code, you can:

- **Build faster** - Focus on agent logic, not infrastructure
- **Scale easier** - Same patterns from prototype to production
- **Maintain simpler** - YAML is readable and maintainable
- **Deploy confidently** - Built-in production features

The question isn't whether to adopt multi-agent systems - it's how quickly you can get started. With Hector, the answer is: **right now**.

**Start building your multi-agent future today.**

---

**Ready to get started?**
- 📁 **Example Config:** [`configs/research-assistant.yaml`](../configs/research-assistant.yaml)
- 📖 **Full Documentation:** [Hector Agent Guide](../AGENTS.md)
- 💬 **Questions?** [Join the discussion](https://github.com/kadirpekel/hector/discussions)

<function_calls>
<invoke name="todo_write">
<parameter name="merge">true
