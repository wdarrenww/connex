#!/bin/bash

# Security Testing Runner Script
# This script runs all security tests for the Connex application

set -e

echo "🔒 Starting Security Testing Suite"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

# Configuration
TEST_RESULTS_DIR="test-results"
SECURITY_REPORT="security-test-report-$(date +%Y%m%d-%H%M%S).md"

# Create test results directory
mkdir -p "$TEST_RESULTS_DIR"

# Function to run unit tests
run_unit_tests() {
    print_test "Running Unit Security Tests..."
    
    if go test -v ./tests/security/... 2>&1 | tee "$TEST_RESULTS_DIR/unit-tests.log"; then
        print_status "✓ Unit tests passed"
        return 0
    else
        print_error "✗ Unit tests failed"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_test "Running Integration Security Tests..."
    
    if go test -v -tags=integration ./tests/integration/... 2>&1 | tee "$TEST_RESULTS_DIR/integration-tests.log"; then
        print_status "✓ Integration tests passed"
        return 0
    else
        print_error "✗ Integration tests failed"
        return 1
    fi
}

# Function to run manual security tests
run_manual_tests() {
    print_test "Running Manual Security Tests..."
    
    # Check if application is running
    if ! curl -s -f "http://localhost:8080/health" > /dev/null; then
        print_warning "Application not running. Starting development environment..."
        make dev-docker &
        sleep 30
    fi
    
    if ./scripts/test-security.sh 2>&1 | tee "$TEST_RESULTS_DIR/manual-tests.log"; then
        print_status "✓ Manual tests completed"
        return 0
    else
        print_error "✗ Manual tests failed"
        return 1
    fi
}

# Function to run security scanning
run_security_scan() {
    print_test "Running Security Scanning..."
    
    if ./scripts/security-scan.sh 2>&1 | tee "$TEST_RESULTS_DIR/security-scan.log"; then
        print_status "✓ Security scan completed"
        return 0
    else
        print_error "✗ Security scan failed"
        return 1
    fi
}

# Function to generate comprehensive report
generate_report() {
    print_status "Generating comprehensive security test report..."
    
    cat > "$SECURITY_REPORT" << EOF
# Comprehensive Security Test Report

**Date**: $(date)
**Application**: Connex
**Test Suite**: Security Testing

## Test Summary

### Unit Tests
- **Status**: $(if [ -f "$TEST_RESULTS_DIR/unit-tests.log" ] && grep -q "PASS" "$TEST_RESULTS_DIR/unit-tests.log"; then echo "✅ PASSED"; else echo "❌ FAILED"; fi)
- **Log File**: $TEST_RESULTS_DIR/unit-tests.log

### Integration Tests
- **Status**: $(if [ -f "$TEST_RESULTS_DIR/integration-tests.log" ] && grep -q "PASS" "$TEST_RESULTS_DIR/integration-tests.log"; then echo "✅ PASSED"; else echo "❌ FAILED"; fi)
- **Log File**: $TEST_RESULTS_DIR/integration-tests.log

### Manual Tests
- **Status**: $(if [ -f "$TEST_RESULTS_DIR/manual-tests.log" ] && grep -q "Security testing completed" "$TEST_RESULTS_DIR/manual-tests.log"; then echo "✅ PASSED"; else echo "❌ FAILED"; fi)
- **Log File**: $TEST_RESULTS_DIR/manual-tests.log

### Security Scan
- **Status**: $(if [ -f "$TEST_RESULTS_DIR/security-scan.log" ] && grep -q "Security scan completed" "$TEST_RESULTS_DIR/security-scan.log"; then echo "✅ PASSED"; else echo "❌ FAILED"; fi)
- **Log File**: $TEST_RESULTS_DIR/security-scan.log

## Security Features Tested

### 1. Authentication & Authorization
- [ ] JWT token validation
- [ ] Password policy enforcement
- [ ] Session management

### 2. Input Validation
- [ ] Email format validation
- [ ] Password complexity requirements
- [ ] XSS protection
- [ ] SQL injection protection

### 3. Security Headers
- [ ] X-Content-Type-Options
- [ ] X-Frame-Options
- [ ] X-XSS-Protection
- [ ] Content-Security-Policy
- [ ] Referrer-Policy
- [ ] Modern security headers

### 4. Rate Limiting
- [ ] IP-based rate limiting
- [ ] Authentication endpoint protection
- [ ] Request throttling

### 5. Error Handling
- [ ] Information leakage prevention
- [ ] Structured error responses
- [ ] No sensitive data in logs

### 6. Security Monitoring
- [ ] Failed login attempt detection
- [ ] Suspicious request detection
- [ ] Security event logging

### 7. CSRF Protection
- [ ] CSRF token validation
- [ ] State-changing request protection

### 8. Request Size Limiting
- [ ] Large request rejection
- [ ] Memory protection

## Test Results

### Unit Test Results
\`\`\`
$(if [ -f "$TEST_RESULTS_DIR/unit-tests.log" ]; then tail -20 "$TEST_RESULTS_DIR/unit-tests.log"; else echo "No unit test results available"; fi)
\`\`\`

### Integration Test Results
\`\`\`
$(if [ -f "$TEST_RESULTS_DIR/integration-tests.log" ]; then tail -20 "$TEST_RESULTS_DIR/integration-tests.log"; else echo "No integration test results available"; fi)
\`\`\`

### Manual Test Results
\`\`\`
$(if [ -f "$TEST_RESULTS_DIR/manual-tests.log" ]; then tail -20 "$TEST_RESULTS_DIR/manual-tests.log"; else echo "No manual test results available"; fi)
\`\`\`

### Security Scan Results
\`\`\`
$(if [ -f "$TEST_RESULTS_DIR/security-scan.log" ]; then tail -20 "$TEST_RESULTS_DIR/security-scan.log"; else echo "No security scan results available"; fi)
\`\`\`

## Recommendations

1. **Review Failed Tests**: Address any failed tests immediately
2. **Monitor Logs**: Check application logs for security events
3. **Regular Testing**: Run security tests regularly in CI/CD
4. **Penetration Testing**: Consider professional penetration testing
5. **Security Updates**: Keep dependencies updated

## Next Steps

1. Fix any identified security issues
2. Implement additional security measures if needed
3. Set up automated security testing in CI/CD pipeline
4. Schedule regular security audits

---

**Report Generated**: $(date)
**Test Environment**: $(uname -a)
**Go Version**: $(go version)
EOF

    print_status "Comprehensive security test report generated: $SECURITY_REPORT"
}

# Main execution
main() {
    echo "Starting comprehensive security testing suite..."
    
    # Initialize counters
    PASSED=0
    FAILED=0
    
    # Run unit tests
    if run_unit_tests; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Run integration tests
    if run_integration_tests; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Run manual tests
    if run_manual_tests; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Run security scan
    if run_security_scan; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
    
    # Generate report
    generate_report
    
    # Summary
    echo ""
    echo "🔒 Security Testing Suite Completed"
    echo "=================================="
    echo "✅ Passed: $PASSED"
    echo "❌ Failed: $FAILED"
    echo "📋 Report: $SECURITY_REPORT"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        print_status "All security tests passed! 🎉"
        exit 0
    else
        print_error "Some security tests failed. Please review the report."
        exit 1
    fi
}

# Run main function
main "$@" 