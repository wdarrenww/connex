#!/bin/bash

# Load Testing Runner Script
# This script runs various load testing scenarios using k6

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL=${BASE_URL:-"http://localhost:8080"}
OUTPUT_DIR=${OUTPUT_DIR:-"./load-test-results"}
K6_IMAGE=${K6_IMAGE:-"grafana/k6:latest"}

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo -e "${BLUE}🚀 Starting Load Testing Suite${NC}"
echo -e "${BLUE}Base URL: ${BASE_URL}${NC}"
echo -e "${BLUE}Output Directory: ${OUTPUT_DIR}${NC}"
echo ""

# Function to run a test
run_test() {
    local test_name=$1
    local test_file=$2
    local output_file="$OUTPUT_DIR/${test_name}-$(date +%Y%m%d-%H%M%S).json"
    
    echo -e "${YELLOW}📊 Running ${test_name}...${NC}"
    
    docker run --rm \
        -v "$(pwd)/tests/load:/scripts" \
        -e K6_OUT="json=$output_file" \
        "$K6_IMAGE" run \
        --env BASE_URL="$BASE_URL" \
        "/scripts/$test_file"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ ${test_name} completed successfully${NC}"
        echo -e "${BLUE}   Results saved to: ${output_file}${NC}"
    else
        echo -e "${RED}❌ ${test_name} failed${NC}"
        return 1
    fi
    
    echo ""
}

# Function to run a quick smoke test
run_smoke_test() {
    echo -e "${YELLOW}🔥 Running Smoke Test...${NC}"
    
    docker run --rm \
        -v "$(pwd)/tests/load:/scripts" \
        "$K6_IMAGE" run \
        --env BASE_URL="$BASE_URL" \
        --vus 1 \
        --duration 30s \
        "/scripts/smoke-test.js"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Smoke test passed${NC}"
    else
        echo -e "${RED}❌ Smoke test failed${NC}"
        exit 1
    fi
    
    echo ""
}

# Function to generate summary report
generate_report() {
    local report_file="$OUTPUT_DIR/load-test-summary-$(date +%Y%m%d-%H%M%S).md"
    
    echo -e "${YELLOW}📋 Generating Summary Report...${NC}"
    
    cat > "$report_file" << EOF
# Load Testing Summary Report

Generated on: $(date)

## Test Configuration
- Base URL: $BASE_URL
- Output Directory: $OUTPUT_DIR

## Test Results

### Performance Metrics
- Load Test: \`load-test.js\`
- Stress Test: \`stress-test.js\`
- Performance Test: \`performance-test.js\`

### Key Findings
- Response time percentiles
- Error rates
- Throughput metrics
- Resource utilization

### Recommendations
- Performance optimizations
- Infrastructure scaling
- Monitoring improvements

## Detailed Results
Check individual JSON files in the output directory for detailed metrics.
EOF
    
    echo -e "${GREEN}✅ Summary report generated: ${report_file}${NC}"
    echo ""
}

# Function to check if application is ready
check_app_ready() {
    echo -e "${YELLOW}🔍 Checking if application is ready...${NC}"
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$BASE_URL/health" > /dev/null; then
            echo -e "${GREEN}✅ Application is ready${NC}"
            return 0
        fi
        
        echo -e "${YELLOW}   Attempt $attempt/$max_attempts - Application not ready yet...${NC}"
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo -e "${RED}❌ Application is not ready after $max_attempts attempts${NC}"
    return 1
}

# Main execution
main() {
    echo -e "${BLUE}🎯 Load Testing Suite for Connex Application${NC}"
    echo ""
    
    # Check if application is ready
    check_app_ready
    
    # Run smoke test first
    run_smoke_test
    
    # Run comprehensive load tests
    echo -e "${BLUE}📈 Running Comprehensive Load Tests${NC}"
    echo ""
    
    # Load test
    run_test "load-test" "load-test.js"
    
    # Performance test
    run_test "performance-test" "performance-test.js"
    
    # Stress test (optional - can be skipped for quick runs)
    if [ "${SKIP_STRESS:-false}" != "true" ]; then
        run_test "stress-test" "stress-test.js"
    else
        echo -e "${YELLOW}⏭️  Skipping stress test (SKIP_STRESS=true)${NC}"
        echo ""
    fi
    
    # Generate summary report
    generate_report
    
    echo -e "${GREEN}🎉 Load testing suite completed successfully!${NC}"
    echo -e "${BLUE}📁 Check results in: ${OUTPUT_DIR}${NC}"
}

# Handle command line arguments
case "${1:-}" in
    "smoke")
        check_app_ready
        run_smoke_test
        ;;
    "load")
        check_app_ready
        run_test "load-test" "load-test.js"
        ;;
    "performance")
        check_app_ready
        run_test "performance-test" "performance-test.js"
        ;;
    "stress")
        check_app_ready
        run_test "stress-test" "stress-test.js"
        ;;
    "quick")
        check_app_ready
        run_smoke_test
        run_test "load-test" "load-test.js"
        run_test "performance-test" "performance-test.js"
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  smoke       - Run smoke test only"
        echo "  load        - Run load test only"
        echo "  performance - Run performance test only"
        echo "  stress      - Run stress test only"
        echo "  quick       - Run smoke, load, and performance tests"
        echo "  help        - Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  BASE_URL        - Application URL (default: http://localhost:8080)"
        echo "  OUTPUT_DIR      - Output directory (default: ./load-test-results)"
        echo "  SKIP_STRESS     - Skip stress test (default: false)"
        echo ""
        echo "Examples:"
        echo "  $0                    # Run all tests"
        echo "  $0 smoke              # Run smoke test only"
        echo "  $0 quick              # Run quick test suite"
        echo "  BASE_URL=http://staging.example.com $0  # Test staging environment"
        ;;
    *)
        main
        ;;
esac 