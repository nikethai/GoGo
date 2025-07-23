#!/bin/bash

# Azure AD JWT Authentication Test Script
# This script tests the Azure AD JWT authentication functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api"
AZURE_URL="$BASE_URL/azure"

# Test data
TEST_AZURE_TOKEN="eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6InRlc3Qta2V5LWlkIn0.eyJhdWQiOiJ5b3VyLWNsaWVudC1pZCIsImlzcyI6Imh0dHBzOi8vbG9naW4ubWljcm9zb2Z0b25saW5lLmNvbS90ZW5hbnQtaWQvdjIuMCIsImlhdCI6MTYzMDAwMDAwMCwibmJmIjoxNjMwMDAwMDAwLCJleHAiOjk5OTk5OTk5OTksImFwcGlkIjoieW91ci1hcHAtaWQiLCJvaWQiOiJ1c2VyLW9iamVjdC1pZCIsInJvbGVzIjpbIkFkbWluIiwiVXNlciJdLCJzdWIiOiJ1c2VyLXN1YmplY3QiLCJ0aWQiOiJ0ZW5hbnQtaWQiLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0ZXN0dXNlckBkb21haW4uY29tIiwibmFtZSI6IlRlc3QgVXNlciIsImVtYWlsIjoidGVzdHVzZXJAZG9tYWluLmNvbSJ9.test-signature"
INVALID_TOKEN="invalid.token.here"
EXPIRED_TOKEN="eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6InRlc3Qta2V5LWlkIn0.eyJhdWQiOiJ5b3VyLWNsaWVudC1pZCIsImlzcyI6Imh0dHBzOi8vbG9naW4ubWljcm9zb2Z0b25saW5lLmNvbS90ZW5hbnQtaWQvdjIuMCIsImlhdCI6MTYzMDAwMDAwMCwibmJmIjoxNjMwMDAwMDAwLCJleHAiOjE2MzAwMDAwMDAsImFwcGlkIjoieW91ci1hcHAtaWQiLCJvaWQiOiJ1c2VyLW9iamVjdC1pZCIsInJvbGVzIjpbIkFkbWluIiwiVXNlciJdLCJzdWIiOiJ1c2VyLXN1YmplY3QiLCJ0aWQiOiJ0ZW5hbnQtaWQiLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0ZXN0dXNlckBkb21haW4uY29tIiwibmFtZSI6IlRlc3QgVXNlciIsImVtYWlsIjoidGVzdHVzZXJAZG9tYWluLmNvbSJ9.test-signature"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to make HTTP requests and check response
test_endpoint() {
    local method=$1
    local url=$2
    local token=$3
    local expected_status=$4
    local description=$5
    
    print_status "Testing: $description"
    
    if [ -n "$token" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            "$url" 2>/dev/null || echo "000")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            "$url" 2>/dev/null || echo "000")
    fi
    
    # Extract status code (last line)
    status_code=$(echo "$response" | tail -n1)
    # Extract response body (all but last line)
    response_body=$(echo "$response" | head -n -1)
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "✓ Expected status $expected_status, got $status_code"
        if [ -n "$response_body" ] && [ "$response_body" != "" ]; then
            echo "   Response: $response_body"
        fi
    else
        print_error "✗ Expected status $expected_status, got $status_code"
        if [ -n "$response_body" ] && [ "$response_body" != "" ]; then
            echo "   Response: $response_body"
        fi
        return 1
    fi
    echo
}

# Function to check if server is running
check_server() {
    print_status "Checking if server is running..."
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        print_success "Server is running"
    else
        print_error "Server is not running. Please start the server first."
        print_status "Run: go run main.go"
        exit 1
    fi
    echo
}

# Function to check environment configuration
check_config() {
    print_status "Checking Azure AD configuration..."
    
    if [ -f ".env" ]; then
        if grep -q "AZURE_AD_TENANT_ID" .env && grep -q "AZURE_AD_CLIENT_ID" .env; then
            print_success "Azure AD configuration found in .env"
        else
            print_warning "Azure AD configuration not found in .env"
            print_status "Please add AZURE_AD_TENANT_ID and AZURE_AD_CLIENT_ID to .env file"
        fi
    else
        print_warning ".env file not found"
    fi
    echo
}

# Main test function
run_tests() {
    echo "==========================================="
    echo "    Azure AD JWT Authentication Tests"
    echo "==========================================="
    echo
    
    # Check prerequisites
    check_server
    check_config
    
    print_status "Starting Azure AD JWT authentication tests..."
    echo
    
    # Test 1: Access without token (should fail)
    test_endpoint "GET" "$API_URL/users" "" "401" "Access protected endpoint without token"
    
    # Test 2: Access with invalid token (should fail)
    test_endpoint "GET" "$API_URL/users" "$INVALID_TOKEN" "401" "Access with invalid token"
    
    # Test 3: Access with expired token (should fail)
    test_endpoint "GET" "$API_URL/users" "$EXPIRED_TOKEN" "401" "Access with expired token"
    
    # Test 4: Access Azure AD only endpoint without token (should fail)
    test_endpoint "GET" "$AZURE_URL/profile" "" "401" "Access Azure AD endpoint without token"
    
    # Test 5: Access Azure AD only endpoint with invalid token (should fail)
    test_endpoint "GET" "$AZURE_URL/profile" "$INVALID_TOKEN" "401" "Access Azure AD endpoint with invalid token"
    
    # Note: The following tests require a real Azure AD token or a mock JWKS endpoint
    print_warning "The following tests require valid Azure AD configuration and tokens:"
    
    # Test 6: Access with valid Azure AD token (would succeed with real token)
    print_status "Testing: Access with valid Azure AD token (requires real token)"
    print_warning "Skipping - requires valid Azure AD token and configuration"
    echo
    
    # Test 7: Access role-protected endpoint (would test RBAC)
    print_status "Testing: Role-based access control (requires real token)"
    print_warning "Skipping - requires valid Azure AD token with roles"
    echo
    
    # Test 8: Hybrid authentication (would test both JWT types)
    print_status "Testing: Hybrid authentication (requires real tokens)"
    print_warning "Skipping - requires both regular JWT and Azure AD tokens"
    echo
    
    print_status "Basic authentication tests completed!"
    echo
    
    # Instructions for manual testing
    echo "==========================================="
    echo "    Manual Testing Instructions"
    echo "==========================================="
    echo
    print_status "To test with real Azure AD tokens:"
    echo "1. Configure Azure AD application in Azure Portal"
    echo "2. Set AZURE_AD_TENANT_ID and AZURE_AD_CLIENT_ID in .env"
    echo "3. Obtain a valid Azure AD token using:"
    echo "   - Azure CLI: az account get-access-token --resource <client-id>"
    echo "   - Postman with OAuth 2.0 flow"
    echo "   - Your application's login flow"
    echo "4. Replace TEST_AZURE_TOKEN in this script with the real token"
    echo "5. Run the script again"
    echo
    
    print_status "Example curl commands for manual testing:"
    echo
    echo "# Test with Azure AD token"
    echo "curl -H \"Authorization: Bearer <your-azure-ad-token>\" \\"
    echo "     -H \"Content-Type: application/json\" \\"
    echo "     $API_URL/users"
    echo
    echo "# Test Azure AD only endpoint"
    echo "curl -H \"Authorization: Bearer <your-azure-ad-token>\" \\"
    echo "     -H \"Content-Type: application/json\" \\"
    echo "     $AZURE_URL/profile"
    echo
    echo "# Test role-based access"
    echo "curl -H \"Authorization: Bearer <your-azure-ad-token-with-admin-role>\" \\"
    echo "     -H \"Content-Type: application/json\" \\"
    echo "     $API_URL/admin/users"
    echo
}

# Function to test token parsing (offline test)
test_token_parsing() {
    echo "==========================================="
    echo "    Token Parsing Tests (Offline)"
    echo "==========================================="
    echo
    
    print_status "Testing JWT token structure parsing..."
    
    # Test Azure AD token structure (this is a mock token for structure testing)
    MOCK_AZURE_TOKEN="eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6InRlc3Qta2V5LWlkIn0.eyJhdWQiOiJ5b3VyLWNsaWVudC1pZCIsImlzcyI6Imh0dHBzOi8vbG9naW4ubWljcm9zb2Z0b25saW5lLmNvbS90ZW5hbnQtaWQvdjIuMCIsImlhdCI6MTYzMDAwMDAwMCwibmJmIjoxNjMwMDAwMDAwLCJleHAiOjk5OTk5OTk5OTksImFwcGlkIjoieW91ci1hcHAtaWQiLCJvaWQiOiJ1c2VyLW9iamVjdC1pZCIsInJvbGVzIjpbIkFkbWluIiwiVXNlciJdLCJzdWIiOiJ1c2VyLXN1YmplY3QiLCJ0aWQiOiJ0ZW5hbnQtaWQiLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0ZXN0dXNlckBkb21haW4uY29tIiwibmFtZSI6IlRlc3QgVXNlciIsImVtYWlsIjoidGVzdHVzZXJAZG9tYWluLmNvbSJ9.test-signature"
    
    # Decode JWT payload (base64 decode the middle part)
    payload=$(echo "$MOCK_AZURE_TOKEN" | cut -d'.' -f2)
    # Add padding if needed
    case $((${#payload} % 4)) in
        2) payload="${payload}==";;
        3) payload="${payload}=";;
    esac
    
    decoded_payload=$(echo "$payload" | base64 -d 2>/dev/null || echo "Failed to decode")
    
    if [ "$decoded_payload" != "Failed to decode" ]; then
        print_success "Successfully parsed mock Azure AD token structure:"
        echo "$decoded_payload" | python3 -m json.tool 2>/dev/null || echo "$decoded_payload"
    else
        print_error "Failed to parse token structure"
    fi
    echo
}

# Function to show help
show_help() {
    echo "Azure AD JWT Authentication Test Script"
    echo
    echo "Usage: $0 [options]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -p, --parse    Test token parsing only (offline)"
    echo "  -f, --full     Run full test suite (default)"
    echo
    echo "Examples:"
    echo "  $0              # Run full test suite"
    echo "  $0 --parse      # Test token parsing only"
    echo "  $0 --help       # Show this help"
    echo
}

# Parse command line arguments
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -p|--parse)
        test_token_parsing
        exit 0
        ;;
    -f|--full|"")
        test_token_parsing
        run_tests
        ;;
    *)
        print_error "Unknown option: $1"
        show_help
        exit 1
        ;;
esac

print_success "All tests completed!"
echo "For complete testing, configure Azure AD and use real tokens."