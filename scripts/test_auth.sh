#!/bin/bash

# Test script for JWT authentication and role-based access control

BASE_URL="http://localhost:3001"
TOKEN=""

# Colors for output
GREEN="\033[0;32m"
RED="\033[0;31m"
NC="\033[0m" # No Color
BLUE="\033[0;34m"

echo -e "${BLUE}=== Testing JWT Authentication and Role-Based Access Control ===${NC}\n"

# Function to make API requests
function api_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local auth_header=$4

    if [ -n "$data" ]; then
        if [ -n "$auth_header" ]; then
            curl -s -X "$method" -H "Content-Type: application/json" -H "Authorization: $auth_header" -d "$data" "$BASE_URL$endpoint"
        else
            curl -s -X "$method" -H "Content-Type: application/json" -d "$data" "$BASE_URL$endpoint"
        fi
    else
        if [ -n "$auth_header" ]; then
            curl -s -X "$method" -H "Content-Type: application/json" -H "Authorization: $auth_header" "$BASE_URL$endpoint"
        else
            curl -s -X "$method" -H "Content-Type: application/json" "$BASE_URL$endpoint"
        fi
    fi
}

# Test 1: Register a new user with admin role
echo -e "${BLUE}Test 1: Register a new admin user${NC}"
REGISTER_DATA='{"username":"admin","password":"admin123","roles":["admin"]}'
REGISTER_RESPONSE=$(api_request "POST" "/auth/register" "$REGISTER_DATA")
echo "Response: $REGISTER_RESPONSE"

if [[ "$REGISTER_RESPONSE" == *"Account created successfully"* ]]; then
    echo -e "${GREEN}✓ Admin user registration successful${NC}\n"
else
    echo -e "${RED}✗ Admin user registration failed${NC}\n"
fi

# Test 2: Login with admin user
echo -e "${BLUE}Test 2: Login with admin user${NC}"
LOGIN_DATA='{"username":"admin","password":"admin123"}'
LOGIN_RESPONSE=$(api_request "POST" "/auth/login" "$LOGIN_DATA")
echo "Response: $LOGIN_RESPONSE"

if [[ "$LOGIN_RESPONSE" == *"Login successful"* ]]; then
    echo -e "${GREEN}✓ Admin login successful${NC}\n"
    # Extract token from response
    TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d '"' -f 4)
    echo "Token: $TOKEN"
else
    echo -e "${RED}✗ Admin login failed${NC}\n"
    exit 1
fi

# Test 3: Access protected route with valid token
echo -e "${BLUE}Test 3: Access protected route with valid token${NC}"
PROTECTED_RESPONSE=$(api_request "GET" "/api/questions" "" "Bearer $TOKEN")
echo "Response: $PROTECTED_RESPONSE"

if [[ "$PROTECTED_RESPONSE" == *"Questions retrieved successfully"* ]]; then
    echo -e "${GREEN}✓ Protected route access successful${NC}\n"
else
    echo -e "${RED}✗ Protected route access failed${NC}\n"
fi

# Test 4: Create a new role (admin only)
echo -e "${BLUE}Test 4: Create a new role (admin only)${NC}"
ROLE_DATA='{"name":"content_creator","description":"Can create content"}'
ROLE_RESPONSE=$(api_request "POST" "/api/roles" "$ROLE_DATA" "Bearer $TOKEN")
echo "Response: $ROLE_RESPONSE"

if [[ "$ROLE_RESPONSE" == *"Role created successfully"* ]]; then
    echo -e "${GREEN}✓ Role creation successful${NC}\n"
else
    echo -e "${RED}✗ Role creation failed${NC}\n"
fi

# Test 5: Register a user with content_creator role
echo -e "${BLUE}Test 5: Register a content creator user${NC}"
REGISTER_DATA='{"username":"creator","password":"creator123","roles":["content_creator"]}'
REGISTER_RESPONSE=$(api_request "POST" "/auth/register" "$REGISTER_DATA")
echo "Response: $REGISTER_RESPONSE"

if [[ "$REGISTER_RESPONSE" == *"Account created successfully"* ]]; then
    echo -e "${GREEN}✓ Content creator user registration successful${NC}\n"
else
    echo -e "${RED}✗ Content creator user registration failed${NC}\n"
fi

# Test 6: Login with content_creator user
echo -e "${BLUE}Test 6: Login with content creator user${NC}"
LOGIN_DATA='{"username":"creator","password":"creator123"}'
LOGIN_RESPONSE=$(api_request "POST" "/auth/login" "$LOGIN_DATA")
echo "Response: $LOGIN_RESPONSE"

if [[ "$LOGIN_RESPONSE" == *"Login successful"* ]]; then
    echo -e "${GREEN}✓ Content creator login successful${NC}\n"
    # Extract token from response
    CREATOR_TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d '"' -f 4)
    echo "Token: $CREATOR_TOKEN"
else
    echo -e "${RED}✗ Content creator login failed${NC}\n"
    exit 1
fi

# Test 7: Create a question with content_creator role
echo -e "${BLUE}Test 7: Create a question with content creator role${NC}"
QUESTION_DATA='{"title":"Test Question","content":"This is a test question","tags":["test"]}'
QUESTION_RESPONSE=$(api_request "POST" "/api/questions" "$QUESTION_DATA" "Bearer $CREATOR_TOKEN")
echo "Response: $QUESTION_RESPONSE"

if [[ "$QUESTION_RESPONSE" == *"Question created successfully"* ]]; then
    echo -e "${GREEN}✓ Question creation successful${NC}\n"
else
    echo -e "${RED}✗ Question creation failed${NC}\n"
fi

# Test 8: Try to create a role with content_creator role (should fail)
echo -e "${BLUE}Test 8: Try to create a role with content creator role (should fail)${NC}"
ROLE_DATA='{"name":"test_role","description":"Test role"}'
ROLE_RESPONSE=$(api_request "POST" "/api/roles" "$ROLE_DATA" "Bearer $CREATOR_TOKEN")
echo "Response: $ROLE_RESPONSE"

if [[ "$ROLE_RESPONSE" == *"Unauthorized"* ]]; then
    echo -e "${GREEN}✓ Role creation correctly failed due to insufficient permissions${NC}\n"
else
    echo -e "${RED}✗ Role creation unexpectedly succeeded or failed for the wrong reason${NC}\n"
fi

echo -e "${BLUE}=== All tests completed ===${NC}"