#!/bin/bash

# Security Scanning Script for Connex Application
# This script performs comprehensive security scanning

set -e

echo "🔒 Starting Security Scan for Connex Application"
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Check if required tools are installed
check_dependencies() {
    print_status "Checking required tools..."
    
    if ! command -v trivy &> /dev/null; then
        print_error "Trivy is not installed. Please install it first."
        print_status "Installation: https://aquasecurity.github.io/trivy/latest/getting-started/installation/"
        exit 1
    fi
    
    if ! command -v gosec &> /dev/null; then
        print_warning "gosec is not installed. Installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    if ! command -v nancy &> /dev/null; then
        print_warning "nancy is not installed. Installing..."
        go install github.com/sonatype-nexus-community/nancy@latest
    fi
}

# Container security scanning
scan_containers() {
    print_status "Scanning containers for vulnerabilities..."
    
    # Build the image first
    docker build -t connex:security-scan .
    
    # Scan with Trivy
    print_status "Running Trivy vulnerability scan..."
    trivy image connex:security-scan --severity HIGH,CRITICAL --format table
    
    # Scan for secrets
    print_status "Running Trivy secret scan..."
    trivy image connex:security-scan --security-checks secret --format table
    
    # Scan for misconfigurations
    print_status "Running Trivy misconfiguration scan..."
    trivy config . --severity HIGH,CRITICAL --format table
}

# Go security analysis
scan_go_code() {
    print_status "Scanning Go code for security issues..."
    
    # Run gosec
    print_status "Running gosec security analysis..."
    gosec ./... -fmt=json -out=gosec-report.json
    
    # Display summary
    if [ -f gosec-report.json ]; then
        print_status "Gosec scan completed. Check gosec-report.json for details."
        # Count issues by severity
        HIGH_ISSUES=$(grep -c '"severity":"HIGH"' gosec-report.json || echo "0")
        MEDIUM_ISSUES=$(grep -c '"severity":"MEDIUM"' gosec-report.json || echo "0")
        LOW_ISSUES=$(grep -c '"severity":"LOW"' gosec-report.json || echo "0")
        
        echo "Issues found:"
        echo "  High: $HIGH_ISSUES"
        echo "  Medium: $MEDIUM_ISSUES"
        echo "  Low: $LOW_ISSUES"
    fi
}

# Dependency vulnerability scanning
scan_dependencies() {
    print_status "Scanning dependencies for vulnerabilities..."
    
    # Run nancy
    print_status "Running nancy dependency scan..."
    nancy sleuth
    
    # Check for outdated dependencies
    print_status "Checking for outdated dependencies..."
    go list -u -m all | grep -E "\[.*\]" || print_status "All dependencies are up to date."
}

# Configuration security check
check_configuration() {
    print_status "Checking configuration security..."
    
    # Check for hardcoded secrets
    print_status "Checking for hardcoded secrets..."
    if grep -r "password\|secret\|key" . --exclude-dir=.git --exclude-dir=vendor --exclude=*.md | grep -v "example\|test\|TODO"; then
        print_warning "Potential hardcoded secrets found. Please review the files above."
    else
        print_status "No obvious hardcoded secrets found."
    fi
    
    # Check for proper file permissions
    print_status "Checking file permissions..."
    find . -name "*.env*" -o -name "*.key" -o -name "*.pem" | while read file; do
        if [ -f "$file" ]; then
            perms=$(stat -c "%a" "$file")
            if [ "$perms" != "600" ] && [ "$perms" != "400" ]; then
                print_warning "Insecure permissions on $file: $perms"
            fi
        fi
    done
}

# Docker Compose security check
check_docker_compose() {
    print_status "Checking Docker Compose security..."
    
    # Check for exposed ports
    if grep -r "ports:" docker-compose*.yml | grep -v "#.*ports:"; then
        print_warning "Exposed ports found in Docker Compose files. Review for production."
    fi
    
    # Check for environment variables
    if grep -r "environment:" docker-compose*.yml | grep -v "#.*environment:"; then
        print_status "Environment variables found. Ensure secrets are not hardcoded."
    fi
}

# Generate security report
generate_report() {
    print_status "Generating security report..."
    
    REPORT_FILE="security-scan-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$REPORT_FILE" << EOF
# Security Scan Report - $(date)

## Scan Summary
- **Date**: $(date)
- **Application**: Connex
- **Scanner**: Trivy, gosec, nancy

## Container Security
\`\`\`
$(trivy image connex:security-scan --severity HIGH,CRITICAL --format json 2>/dev/null || echo "Container scan failed")
\`\`\`

## Go Code Security
\`\`\`
$(cat gosec-report.json 2>/dev/null || echo "Gosec scan failed")
\`\`\`

## Dependencies
\`\`\`
$(nancy sleuth 2>/dev/null || echo "Dependency scan failed")
\`\`\`

## Recommendations
1. Review all HIGH and CRITICAL vulnerabilities
2. Update dependencies with known vulnerabilities
3. Ensure no secrets are hardcoded in configuration
4. Review exposed ports in production
5. Implement proper secrets management

EOF

    print_status "Security report generated: $REPORT_FILE"
}

# Main execution
main() {
    echo "Starting comprehensive security scan..."
    
    check_dependencies
    scan_containers
    scan_go_code
    scan_dependencies
    check_configuration
    check_docker_compose
    generate_report
    
    echo ""
    echo "🔒 Security scan completed!"
    echo "Check the generated report for detailed findings."
}

# Run main function
main "$@" 