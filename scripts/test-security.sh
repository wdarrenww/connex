#!/bin/bash

# Manual Security Testing Script for Connex Application
# This script tests security enhancements against a running application

set -e

echo "🔒 Starting Manual Security Testing"
echo "==================================="

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
BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_EMAIL="security-test@example.com"
TEST_PASSWORD="SecurePass123!"

# Check if application is running
check_app_running() {
    print_status "Checking if application is running..."
    
    if curl -s -f "$BASE_URL/health" > /dev/null; then
        print_status "Application is running at $BASE_URL"
    else
        print_error "Application is not running at $BASE_URL"
        print_status "Please start the application first: make dev-docker"
        exit 1
    fi
}

# Test 1: Security Headers
test_security_headers() {
    print_test "Testing Security Headers..."
    
    response=$(curl -s -I "$BASE_URL/api/auth/login")
    
    # Check for required security headers
    headers=(
        "X-Content-Type-Options: nosniff"
        "X-Frame-Options: DENY"
        "X-XSS-Protection: 1; mode=block"
        "Referrer-Policy: strict-origin-when-cross-origin"
        "Permissions-Policy:"
        "Cross-Origin-Embedder-Policy:"
        "Cross-Origin-Opener-Policy:"
        "Cross-Origin-Resource-Policy:"
        "Content-Security-Policy:"
    )
    
    for header in "${headers[@]}"; do
        if echo "$response" | grep -q "$header"; then
            print_status "✓ Found header: $header"
        else
            print_error "✗ Missing header: $header"
        fi
    done
    
    # Check CSP specifically
    if echo "$response" | grep -q "Content-Security-Policy.*nonce-"; then
        print_status "✓ CSP contains nonce"
    else
        print_error "✗ CSP missing nonce"
    fi
    
    if echo "$response" | grep -q "unsafe-inline"; then
        print_error "✗ CSP contains unsafe-inline"
    else
        print_status "✓ CSP does not contain unsafe-inline"
    fi
}

# Test 2: Password Policy
test_password_policy() {
    print_test "Testing Password Policy..."
    
    # Test weak passwords
    weak_passwords=(
        "short"
        "password123!"
        "PASSWORD123!"
        "Password!"
        "Password123"
    )
    
    for password in "${weak_passwords[@]}"; do
        response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
            -H "Content-Type: application/json" \
            -d "{\"name\":\"Test User\",\"email\":\"test-$(date +%s)@example.com\",\"password\":\"$password\"}")
        
        http_code="${response: -3}"
        
        if [ "$http_code" = "400" ]; then
            print_status "✓ Weak password rejected: $password"
        else
            print_error "✗ Weak password accepted: $password (HTTP $http_code)"
        fi
    done
    
    # Test strong password
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"Test User\",\"email\":\"test-$(date +%s)@example.com\",\"password\":\"$TEST_PASSWORD\"}")
    
    http_code="${response: -3}"
    
    if [ "$http_code" = "201" ]; then
        print_status "✓ Strong password accepted"
    else
        print_error "✗ Strong password rejected (HTTP $http_code)"
    fi
}

# Test 3: Email Validation
test_email_validation() {
    print_test "Testing Email Validation..."
    
    # Test invalid emails
    invalid_emails=(
        "invalid-email"
        "testexample.com"
        "test@"
        ""
    )
    
    for email in "${invalid_emails[@]}"; do
        response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
            -H "Content-Type: application/json" \
            -d "{\"name\":\"Test User\",\"email\":\"$email\",\"password\":\"$TEST_PASSWORD\"}")
        
        http_code="${response: -3}"
        
        if [ "$http_code" = "400" ]; then
            print_status "✓ Invalid email rejected: $email"
        else
            print_error "✗ Invalid email accepted: $email (HTTP $http_code)"
        fi
    done
    
    # Test valid email
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"Test User\",\"email\":\"valid-$(date +%s)@example.com\",\"password\":\"$TEST_PASSWORD\"}")
    
    http_code="${response: -3}"
    
    if [ "$http_code" = "201" ]; then
        print_status "✓ Valid email accepted"
    else
        print_error "✗ Valid email rejected (HTTP $http_code)"
    fi
}

# Test 4: XSS Protection
test_xss_protection() {
    print_test "Testing XSS Protection..."
    
    # Test XSS payloads
    xss_payloads=(
        "<script>alert('xss')</script>"
        "<img src=x onerror=alert('xss')>"
        "javascript:alert('xss')"
        "<svg onload=alert('xss')>"
    )
    
    for payload in "${xss_payloads[@]}"; do
        response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
            -H "Content-Type: application/json" \
            -d "{\"name\":\"$payload\",\"email\":\"test-$(date +%s)@example.com\",\"password\":\"$TEST_PASSWORD\"}")
        
        http_code="${response: -3}"
        
        if [ "$http_code" = "400" ]; then
            print_status "✓ XSS payload rejected: $payload"
        else
            print_error "✗ XSS payload accepted: $payload (HTTP $http_code)"
        fi
    done
}

# Test 5: Rate Limiting
test_rate_limiting() {
    print_test "Testing Rate Limiting..."
    
    # Make multiple rapid requests
    for i in {1..10}; do
        response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"email\":\"test$i@example.com\",\"password\":\"wrongpassword\"}")
        
        http_code="${response: -3}"
        
        if [ "$http_code" = "429" ]; then
            print_status "✓ Rate limiting working (request $i)"
            break
        fi
    done
    
    if [ "$http_code" != "429" ]; then
        print_warning "⚠ Rate limiting may not be working (no 429 response)"
    fi
}

# Test 6: Error Handling
test_error_handling() {
    print_test "Testing Error Handling..."
    
    # Test invalid JSON
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "invalid json")
    
    http_code="${response: -3}"
    
    if [ "$http_code" = "400" ]; then
        print_status "✓ Invalid JSON properly rejected"
        
        # Check response doesn't contain sensitive info
        if echo "$response" | grep -q "stack_trace\|internal_error\|database"; then
            print_error "✗ Error response contains sensitive information"
        else
            print_status "✓ Error response properly sanitized"
        fi
    else
        print_error "✗ Invalid JSON not properly handled (HTTP $http_code)"
    fi
}

# Test 7: JWT Token Validation
test_jwt_validation() {
    print_test "Testing JWT Token Validation..."
    
    # Test with invalid token
    response=$(curl -s -w "%{http_code}" -X GET "$BASE_URL/api/users/1" \
        -H "Authorization: Bearer invalid.token.here")
    
    http_code="${response: -3}"
    
    if [ "$http_code" = "401" ]; then
        print_status "✓ Invalid JWT properly rejected"
    else
        print_error "✗ Invalid JWT not properly handled (HTTP $http_code)"
    fi
    
    # Test without token
    response=$(curl -s -w "%{http_code}" -X GET "$BASE_URL/api/users/1")
    
    http_code="${response: -3}"
    
    if [ "$http_code" = "401" ]; then
        print_status "✓ Missing JWT properly rejected"
    else
        print_error "✗ Missing JWT not properly handled (HTTP $http_code)"
    fi
}

# Test 8: Request Size Limiting
test_request_size_limit() {
    print_test "Testing Request Size Limiting..."
    
    # Create large payload (2MB)
    large_payload=$(printf '{"name":"%s","email":"test@example.com","password":"%s"}' \
        "$(printf 'a%.0s' {1..1000000})" "$TEST_PASSWORD")
    
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "$large_payload")
    
    http_code="${response: -3}"
    
    if [ "$http_code" = "413" ]; then
        print_status "✓ Large request properly rejected"
    else
        print_warning "⚠ Large request not rejected (HTTP $http_code)"
    fi
}

# Test 9: Security Monitoring
test_security_monitoring() {
    print_test "Testing Security Monitoring..."
    
    # Test failed login attempts
    for i in {1..5}; do
        curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"email\":\"monitoring-test@example.com\",\"password\":\"wrongpassword\"}" > /dev/null
    done
    
    print_status "✓ Failed login attempts sent (check logs for monitoring)"
    
    # Test suspicious user agent
    curl -s -X GET "$BASE_URL/api/auth/login" \
        -H "User-Agent: sqlmap/1.0" > /dev/null
    
    print_status "✓ Suspicious user agent sent (check logs for monitoring)"
}

# Test 10: CSRF Protection
test_csrf_protection() {
    print_test "Testing CSRF Protection..."
    
    # Test without CSRF token
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"Test User\",\"email\":\"csrf-test@example.com\",\"password\":\"$TEST_PASSWORD\"}")
    
    http_code="${response: -3}"
    
    if [ "$http_code" = "403" ]; then
        print_status "✓ CSRF protection working"
    else
        print_warning "⚠ CSRF protection may not be working (HTTP $http_code)"
    fi
}

# Generate test report
generate_report() {
    print_status "Generating security test report..."
    
    REPORT_FILE="security-test-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$REPORT_FILE" << EOF
# Security Test Report - $(date)

## Test Summary
- **Date**: $(date)
- **Application**: Connex
- **Base URL**: $BASE_URL
- **Test Password**: $TEST_PASSWORD

## Test Results

### 1. Security Headers
- [ ] X-Content-Type-Options
- [ ] X-Frame-Options
- [ ] X-XSS-Protection
- [ ] Referrer-Policy
- [ ] Permissions-Policy
- [ ] Cross-Origin-Embedder-Policy
- [ ] Cross-Origin-Opener-Policy
- [ ] Cross-Origin-Resource-Policy
- [ ] Content-Security-Policy (with nonce)

### 2. Password Policy
- [ ] Weak passwords rejected
- [ ] Strong passwords accepted
- [ ] Complexity requirements enforced

### 3. Email Validation
- [ ] Invalid emails rejected
- [ ] Valid emails accepted
- [ ] Format validation working

### 4. XSS Protection
- [ ] XSS payloads rejected
- [ ] Input sanitization working

### 5. Rate Limiting
- [ ] Rate limiting enforced
- [ ] 429 responses returned

### 6. Error Handling
- [ ] Invalid JSON handled
- [ ] No sensitive information leaked
- [ ] Structured error responses

### 7. JWT Validation
- [ ] Invalid tokens rejected
- [ ] Missing tokens rejected
- [ ] Token validation working

### 8. Request Size Limiting
- [ ] Large requests rejected
- [ ] Size limits enforced

### 9. Security Monitoring
- [ ] Failed login attempts logged
- [ ] Suspicious requests detected
- [ ] Security events recorded

### 10. CSRF Protection
- [ ] CSRF tokens required
- [ ] Protection working

## Recommendations
1. Review any failed tests
2. Check application logs for security events
3. Verify monitoring is working
4. Test with real attack tools if needed

EOF

    print_status "Security test report generated: $REPORT_FILE"
}

# Main execution
main() {
    echo "Starting comprehensive security testing..."
    
    check_app_running
    test_security_headers
    test_password_policy
    test_email_validation
    test_xss_protection
    test_rate_limiting
    test_error_handling
    test_jwt_validation
    test_request_size_limit
    test_security_monitoring
    test_csrf_protection
    generate_report
    
    echo ""
    echo "🔒 Security testing completed!"
    echo "Check the generated report for detailed results."
}

# Run main function
main "$@" 