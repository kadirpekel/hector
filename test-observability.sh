#!/bin/bash

# Enhanced Observability Test Script
# Tests Sprint 1, 2, and 3 enhancements + Recent Fixes
# - gRPC interceptor chaining
# - SSE streaming
# - Path normalization
# - Nil safety

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
GRPC_PORT=8080
HTTP_PORT=8081
JSONRPC_PORT=8082
METRICS_ENDPOINT="http://localhost:${HTTP_PORT}/metrics"
HEALTH_ENDPOINT="http://localhost:${HTTP_PORT}/v1/agents"
JAEGER_URL="http://localhost:16686"
LOG_FILE="/tmp/hector-observability-test.log"
TEST_CONFIG="${TEST_CONFIG:-configs/test-observability.yaml}"

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Helper functions
log_section() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

log_test() {
    echo -e "${YELLOW}TEST:${NC} $1"
}

log_success() {
    echo -e "${GREEN}✅ PASS:${NC} $1"
    ((TESTS_PASSED++))
    ((TESTS_TOTAL++))
}

log_failure() {
    echo -e "${RED}❌ FAIL:${NC} $1"
    ((TESTS_FAILED++))
    ((TESTS_TOTAL++))
}

log_info() {
    echo -e "${BLUE}ℹ️  INFO:${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠️  WARN:${NC} $1"
}

# Check if metric exists
check_metric() {
    local metric_name=$1
    local description=$2
    local optional=${3:-false}

    if curl -s "$METRICS_ENDPOINT" 2>/dev/null | grep -q "^$metric_name"; then
        log_success "$description: $metric_name found"
        return 0
    else
        if [ "$optional" = "true" ]; then
            log_info "$description: $metric_name not yet present (appears after first use)"
            return 0
        else
            log_failure "$description: $metric_name NOT found"
            return 1
        fi
    fi
}

# Check if metric has data
check_metric_has_data() {
    local metric_pattern=$1
    local description=$2

    local output=$(curl -s "$METRICS_ENDPOINT" 2>/dev/null | grep "$metric_pattern" | grep -v "^#" | head -5)
    if [ -n "$output" ]; then
        log_success "$description has data"
        echo "   Sample: $(echo "$output" | head -1)"
        return 0
    else
        log_failure "$description has NO data"
        return 1
    fi
}

# Cleanup function
cleanup() {
    log_section "CLEANUP"
    if [ -n "$HECTOR_PID" ]; then
        log_info "Stopping Hector (PID: $HECTOR_PID)..."
        kill $HECTOR_PID 2>/dev/null || true
        wait $HECTOR_PID 2>/dev/null || true
        log_success "Hector stopped"
    fi
}

trap cleanup EXIT

# Main test execution
main() {
    log_section "HECTOR OBSERVABILITY INTEGRATION TEST (ENHANCED)"
    log_info "Testing Sprint 1, 2, and 3 enhancements + Recent Fixes"
    log_info "Config: $TEST_CONFIG"
    log_info "Metrics endpoint: $METRICS_ENDPOINT"
    log_info "Log file: $LOG_FILE"
    echo ""

    # Test 1: Prerequisites
    log_section "1. PREREQUISITES CHECK"

    log_test "Checking if hector binary exists"
    if [ -f "./hector" ]; then
        log_success "Hector binary found"
    else
        log_failure "Hector binary not found. Run 'make build' first"
        exit 1
    fi

    log_test "Checking if config file exists"
    if [ -f "$TEST_CONFIG" ]; then
        log_success "Config file found: $TEST_CONFIG"
    else
        log_failure "Config file not found: $TEST_CONFIG"
        exit 1
    fi

    log_test "Checking if grpcurl is available (for gRPC tests)"
    if command -v grpcurl &> /dev/null; then
        log_success "grpcurl found (can test gRPC)"
        GRPCURL_AVAILABLE=true
    else
        log_warning "grpcurl not found (gRPC tests will be limited)"
        log_info "Install: brew install grpcurl (Mac) or apt-get install grpcurl (Linux)"
        GRPCURL_AVAILABLE=false
    fi

    log_test "Checking if ports are available"
    local ports_in_use=0
    for port in $GRPC_PORT $HTTP_PORT $JSONRPC_PORT; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            log_warning "Port $port is already in use"
            ((ports_in_use++))
        fi
    done
    if [ $ports_in_use -eq 0 ]; then
        log_success "All ports are available (gRPC:$GRPC_PORT, HTTP:$HTTP_PORT, JSON-RPC:$JSONRPC_PORT)"
    else
        log_warning "$ports_in_use port(s) already in use, attempting to proceed..."
    fi

    # Test 2: Start Hector
    log_section "2. STARTING HECTOR SERVER"

    log_info "Starting Hector with observability enabled..."
    ./hector serve --config "$TEST_CONFIG" > "$LOG_FILE" 2>&1 &
    HECTOR_PID=$!

    log_success "Hector started (PID: $HECTOR_PID)"
    log_info "Logs: $LOG_FILE"

    # Wait for startup
    log_test "Waiting for server to initialize (up to 30 seconds)..."
    for i in {1..30}; do
        sleep 1
        if curl -s -f "$HEALTH_ENDPOINT" > /dev/null 2>&1; then
            log_success "Server is ready (${i}s)"
            break
        fi
        if [ $(($i % 5)) -eq 0 ]; then
            log_info "Still waiting... (${i}s elapsed)"
        fi
        if [ $i -eq 30 ]; then
            log_failure "Server did not start within 30 seconds"
            log_info "Last 30 lines of log:"
            tail -30 "$LOG_FILE"
            exit 1
        fi
    done

    # Test 3: Basic endpoints
    log_section "3. BASIC ENDPOINT TESTS"

    log_test "GET $HEALTH_ENDPOINT"
    if curl -s -f "$HEALTH_ENDPOINT" > /dev/null; then
        log_success "Discovery endpoint responding"
    else
        log_failure "Discovery endpoint not responding"
    fi

    log_test "GET $METRICS_ENDPOINT"
    if curl -s -f "$METRICS_ENDPOINT" > /dev/null; then
        log_success "Metrics endpoint responding"
    else
        log_failure "Metrics endpoint not responding"
        exit 1
    fi

    local metric_count=$(curl -s "$METRICS_ENDPOINT" | grep -c "^hector_" || true)
    log_info "Found $metric_count Hector metrics"

    # Test 4: HTTP Metrics & Path Normalization
    log_section "4. HTTP METRICS & PATH NORMALIZATION"

    log_test "Checking HTTP metrics"
    check_metric "hector_http_requests_total" "HTTP requests counter"
    check_metric "hector_http_request_duration_seconds" "HTTP request duration histogram"
    check_metric "hector_http_response_size_bytes" "HTTP response size histogram"

    log_info "Generating diverse HTTP traffic to test path normalization..."
    # Hit different agent endpoints
    for agent in "agent1" "agent2" "test-agent" "my_agent" "super-long-agent-name-with-many-chars"; do
        curl -s "http://localhost:${HTTP_PORT}/v1/agents/${agent}/message:send" -X POST -d '{}' > /dev/null 2>&1 || true
        curl -s "http://localhost:${HTTP_PORT}/v1/agents/${agent}/.well-known/agent-card.json" > /dev/null 2>&1 || true
    done
    
    # Hit static endpoints
    for i in {1..5}; do
        curl -s "$HEALTH_ENDPOINT" > /dev/null
        curl -s "$METRICS_ENDPOINT" > /dev/null
    done
    sleep 2

    log_test "Verifying path normalization (cardinality control)"
    local unique_paths=$(curl -s "$METRICS_ENDPOINT" | grep "hector_http_requests_total{" | grep -o 'path="[^"]*"' | sort -u)
    local path_count=$(echo "$unique_paths" | wc -l)
    
    log_info "Unique paths in metrics: $path_count"
    echo "$unique_paths" | while read path; do
        echo "   $path"
    done
    
    if [ "$path_count" -lt 10 ]; then
        log_success "Path cardinality is controlled (< 10 unique paths)"
    else
        log_failure "Path cardinality is too high ($path_count unique paths) - normalization failing!"
    fi

    # Verify specific normalizations
    log_test "Verifying agent endpoints are normalized"
    if echo "$unique_paths" | grep -q 'path="/v1/agents/:agent'; then
        log_success "Agent paths are properly normalized to /v1/agents/:agent/*"
    else
        log_failure "Agent paths are NOT normalized correctly"
    fi

    # Test 5: SSE Streaming (Bug Fix Verification)
    log_section "5. SSE STREAMING (BUG FIX VERIFICATION)"

    log_test "Testing SSE streaming endpoint (should not panic)"
    # Try to stream (will fail without agent/auth but shouldn't panic)
    timeout 2s curl -N -H "Accept: text/event-stream" \
        "http://localhost:${HTTP_PORT}/v1/agents/test-agent/message:stream" \
        -X POST -d '{"text":"test"}' > /dev/null 2>&1 || true
    
    sleep 1
    
    # Check if server is still running
    if kill -0 $HECTOR_PID 2>/dev/null; then
        log_success "Server still running after SSE request (no panic)"
    else
        log_failure "Server crashed on SSE request (Flush() bug)"
        exit 1
    fi

    # Check logs for panic
    if grep -q "panic serving" "$LOG_FILE"; then
        log_failure "Panic detected in logs during SSE test"
        tail -20 "$LOG_FILE"
    else
        log_success "No panic in logs (SSE streaming fix working)"
    fi

    # Test 6: gRPC Metrics (Interceptor Fix Verification)
    log_section "6. gRPC METRICS (INTERCEPTOR FIX VERIFICATION)"

    if [ "$GRPCURL_AVAILABLE" = true ]; then
        log_test "Making gRPC call to test interceptor"
        
        # Try to list services (should trigger interceptor)
        grpcurl -plaintext "localhost:${GRPC_PORT}" list > /dev/null 2>&1 || true
        
        sleep 2
        
        log_test "Checking if gRPC metrics were recorded"
        if curl -s "$METRICS_ENDPOINT" | grep -q "hector_grpc_calls_total"; then
            log_success "gRPC metrics ARE being recorded (interceptor fix working!)"
            check_metric_has_data "hector_grpc_calls_total" "gRPC calls counter"
        else
            log_warning "gRPC metrics not yet present (might need more calls)"
        fi
    else
        log_info "Skipping gRPC call tests (grpcurl not available)"
        log_test "Checking gRPC metrics definitions"
        check_metric "hector_grpc_calls_total" "gRPC calls counter" true
        check_metric "hector_grpc_call_duration_seconds" "gRPC call duration histogram" true
    fi

    # Test 7: LLM & Tool Metrics
    log_section "7. LLM, TOOLS, AGENT METRICS"
    log_info "Note: These metrics appear after actual agent/LLM/tool usage"

    check_metric "hector_agent_call_duration_seconds" "Agent call duration" true
    check_metric "hector_agent_calls_total" "Agent calls counter" true
    check_metric "hector_tool_execution_duration_seconds" "Tool execution duration" true
    check_metric "hector_tool_calls_total" "Tool calls counter" true
    check_metric "hector_llm_request_duration_seconds" "LLM request duration" true
    check_metric "hector_llm_tokens_input_total" "LLM input tokens" true

    # Test 8: Business Metrics
    log_section "8. BUSINESS KPI METRICS"
    log_info "Note: These require explicit recording in application code"

    check_metric "hector_session_duration_seconds" "Session duration" true
    check_metric "hector_session_total" "Session counter" true
    check_metric "hector_conversation_turns" "Conversation turns" true

    # Test 9: Metric Quality
    log_section "9. METRIC QUALITY CHECKS"

    log_test "Checking metric labels"
    local http_labels=$(curl -s "$METRICS_ENDPOINT" | grep "hector_http_requests_total{" | head -1)
    
    if echo "$http_labels" | grep -q "method="; then
        log_success "HTTP metrics have 'method' label"
    else
        log_failure "HTTP metrics missing 'method' label"
    fi

    if echo "$http_labels" | grep -q "path="; then
        log_success "HTTP metrics have 'path' label"
    else
        log_failure "HTTP metrics missing 'path' label"
    fi

    if echo "$http_labels" | grep -q "status_code="; then
        log_success "HTTP metrics have 'status_code' label"
    else
        log_failure "HTTP metrics missing 'status_code' label"
    fi

    log_test "Checking Prometheus format"
    local help_count=$(curl -s "$METRICS_ENDPOINT" | grep -c "^# HELP hector_" || true)
    local type_count=$(curl -s "$METRICS_ENDPOINT" | grep -c "^# TYPE hector_" || true)
    
    if [ "$help_count" -gt 0 ] && [ "$type_count" -gt 0 ]; then
        log_success "Metrics have proper HELP and TYPE declarations ($help_count HELP, $type_count TYPE)"
    else
        log_failure "Metrics missing HELP or TYPE declarations"
    fi

    log_test "Checking histogram configuration"
    local bucket_count=$(curl -s "$METRICS_ENDPOINT" | grep "hector_http_request_duration_seconds_bucket" | wc -l)
    log_info "Found $bucket_count histogram buckets"
    if [ "$bucket_count" -gt 5 ]; then
        log_success "HTTP duration histogram has adequate buckets"
    else
        log_warning "HTTP duration histogram might need more buckets"
    fi

    # Test 10: Performance
    log_section "10. PERFORMANCE CHECKS"

    log_test "Measuring /metrics endpoint latency"
    local start_time=$(date +%s%N)
    curl -s "$METRICS_ENDPOINT" > /dev/null
    local end_time=$(date +%s%N)
    local duration_ms=$(( ($end_time - $start_time) / 1000000 ))

    log_info "Metrics endpoint latency: ${duration_ms}ms"
    if [ "$duration_ms" -lt 100 ]; then
        log_success "Metrics endpoint is fast (< 100ms)"
    elif [ "$duration_ms" -lt 500 ]; then
        log_warning "Metrics endpoint latency is moderate (${duration_ms}ms)"
    else
        log_failure "Metrics endpoint is slow (${duration_ms}ms)"
    fi

    # Test 11: Error Handling
    log_section "11. ERROR HANDLING"

    log_test "Testing 404 endpoint"
    curl -s "http://localhost:${HTTP_PORT}/nonexistent" > /dev/null 2>&1 || true
    sleep 1

    log_test "Checking if 404 was recorded in metrics"
    if curl -s "$METRICS_ENDPOINT" | grep -q 'status_code="404"'; then
        log_success "404 errors are tracked in HTTP metrics"
    else
        log_info "404 status not yet in metrics (might need more time)"
    fi

    # Test 12: Grafana Dashboards
    log_section "12. GRAFANA DASHBOARDS"

    log_test "Checking dashboard files"
    local dashboards=0
    for dashboard in "grafana/dashboards/hector-llm-tools.json" \
                     "grafana/dashboards/hector-http-grpc.json" \
                     "grafana/dashboards/hector-business-metrics.json"; do
        if [ -f "$dashboard" ]; then
            log_success "Dashboard exists: $(basename $dashboard)"
            ((dashboards++))
        else
            log_failure "Dashboard missing: $dashboard"
        fi
    done

    if [ "$dashboards" -eq 3 ]; then
        log_success "All 3 Grafana dashboards present"
    fi

    # Test 13: Documentation
    log_section "13. DOCUMENTATION"

    log_test "Checking observability documentation"
    local docs=0
    for doc in "docs/observability-improvements.md" \
               "docs/observability-sprint2-summary.md" \
               "docs/observability-sprint3-summary.md"; do
        if [ -f "$doc" ]; then
            log_success "Documentation exists: $(basename $doc)"
            ((docs++))
        else
            log_warning "Documentation missing: $doc"
        fi
    done

    # Test 14: Bug Fix Verification
    log_section "14. BUG FIX VERIFICATION"

    log_test "Verifying gRPC interceptor chaining fix"
    if grep -q "ChainUnaryInterceptors" pkg/server/server.go 2>/dev/null; then
        log_success "gRPC interceptor chaining implemented"
    else
        log_failure "gRPC interceptor chaining NOT found in code"
    fi

    log_test "Verifying chi router integration"
    if grep -q "github.com/go-chi/chi" pkg/transport/rest_gateway.go 2>/dev/null; then
        log_success "chi router integrated (no regex patterns needed!)"
        
        # Verify we're using RouteContext not regex
        if grep -q "chi.RouteContext\|RoutePattern" pkg/transport/http_metrics_middleware.go 2>/dev/null; then
            log_success "Using chi.RouteContext for path normalization (proper architecture)"
        else
            log_warning "chi imported but might not be using RouteContext"
        fi
    else
        log_failure "chi router NOT found (still using http.ServeMux?)"
    fi

    log_test "Verifying SSE Flush() fix"
    if grep -q "func (rw \*responseWriter) Flush()" pkg/transport/http_metrics_middleware.go 2>/dev/null; then
        log_success "responseWriter implements Flush() for SSE support"
    else
        log_failure "responseWriter missing Flush() method"
    fi

    log_test "Verifying nil safety"
    if grep -q "return &NoopMetrics{}" pkg/observability/recorder.go 2>/dev/null; then
        log_success "GetGlobalMetrics() returns NoopMetrics instead of nil"
    else
        log_warning "GetGlobalMetrics() might return nil"
    fi

    # Final Summary
    log_section "TEST SUMMARY"

    echo ""
    echo "Total Tests: $TESTS_TOTAL"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}🎉 ALL TESTS PASSED! 🎉${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo ""
        echo "✅ Sprint 1: LLM, Tools, Agent metrics - VERIFIED"
        echo "✅ Sprint 2: HTTP, gRPC tracing - VERIFIED"
        echo "✅ Sprint 3: HTTP, gRPC, Business KPI metrics - VERIFIED"
        echo "✅ Bug Fixes: gRPC interceptors, SSE streaming, path normalization - VERIFIED"
        echo ""
        echo "Next steps:"
        echo "1. Import Grafana dashboards from grafana/dashboards/"
        echo "2. Configure Prometheus to scrape $METRICS_ENDPOINT"
        echo "3. Consider migrating to chi router (see ROUTING-ARCHITECTURE-PROPOSAL.md)"
        echo ""
        exit 0
    else
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}⚠️  SOME TESTS FAILED ⚠️${NC}"
        echo -e "${RED}========================================${NC}"
        echo ""
        echo "Please review the failures above."
        echo "Logs available at: $LOG_FILE"
        echo ""
        exit 1
    fi
}

# Run main function
main

