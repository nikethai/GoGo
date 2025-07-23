# Azure AD JWT Authentication Integration

This document explains how to integrate Azure AD JWT authentication with the Gogo API.

## Overview

The Gogo API now supports Azure AD JWT token validation alongside the existing JWT authentication system. This allows users to authenticate using tokens issued by Azure Active Directory.

## Features

- **Azure AD JWT Token Validation**: Validates tokens issued by Azure AD using public key cryptography
- **Hybrid Authentication**: Supports both regular JWT tokens and Azure AD tokens
- **Role-Based Access Control**: Supports Azure AD roles for authorization
- **Automatic Key Retrieval**: Fetches public keys from Azure AD JWKS endpoint
- **Claims Conversion**: Converts Azure AD claims to internal format

## Configuration

### Environment Variables

Add the following environment variables to your `.env` file:

```env
# Azure AD Configuration
AZURE_AD_TENANT_ID=your_azure_ad_tenant_id
AZURE_AD_CLIENT_ID=your_azure_ad_client_id
```

### Getting Azure AD Configuration

1. **Tenant ID**: Found in Azure Portal > Azure Active Directory > Properties > Tenant ID
2. **Client ID**: The Application (client) ID of your registered Azure AD application

## Usage

### 1. Azure AD Only Authentication

Use `AzureADAuth` middleware for routes that should only accept Azure AD tokens:

```go
import (
    "main/internal/middleware"
)

// Apply Azure AD authentication to specific routes
router.Use(middleware.AzureADAuth)
```

### 2. Hybrid Authentication

Use `HybridAuth` middleware for routes that should accept both regular JWT and Azure AD tokens:

```go
// Apply hybrid authentication (accepts both JWT types)
router.Use(middleware.HybridAuth)
```

### 3. Role-Based Access Control

Use Azure AD specific role middleware:

```go
// Require specific Azure AD role
router.Use(middleware.RequireAzureADRole("Admin"))
```

## Token Format

### Azure AD Token Structure

Azure AD tokens contain the following claims:

```json
{
  "aud": "your-client-id",
  "iss": "https://login.microsoftonline.com/tenant-id/v2.0",
  "iat": 1234567890,
  "nbf": 1234567890,
  "exp": 1234567890,
  "appid": "application-id",
  "oid": "object-id",
  "roles": ["Admin", "User"],
  "sub": "subject-id",
  "tid": "tenant-id",
  "preferred_username": "user@domain.com",
  "name": "User Name",
  "email": "user@domain.com"
}
```

### Claims Mapping

Azure AD claims are mapped to internal JWT claims as follows:

| Azure AD Claim | Internal Claim | Description |
|----------------|----------------|--------------|
| `oid` | `userID` | Azure AD Object ID |
| `preferred_username` | `username` | Primary username (fallback to email/name) |
| `roles` | `roles` | User roles from Azure AD |
| `exp` | `exp` | Token expiration |
| `iss` | `iss` | Token issuer |

## API Usage Examples

### 1. Making Authenticated Requests

```bash
# Using Azure AD token
curl -H "Authorization: Bearer <azure-ad-token>" \
     http://localhost:8080/api/protected-endpoint
```

### 2. Accessing User Information

In your handlers, you can access user information from the context:

```go
func MyHandler(w http.ResponseWriter, r *http.Request) {
    // Get user information (works for both JWT types)
    userID, _ := middleware.GetUserIDFromContext(r)
    username, _ := middleware.GetUsernameFromContext(r)
    roles, _ := middleware.GetUserRolesFromContext(r)
    
    // Get authentication type
    authType, _ := middleware.GetAuthTypeFromContext(r)
    
    // Get Azure AD specific information (if Azure AD token)
    if authType == "azure_ad" {
        tenantID, _ := middleware.GetAzureTenantIDFromContext(r)
        appID, _ := middleware.GetAzureAppIDFromContext(r)
        objectID, _ := middleware.GetAzureObjectIDFromContext(r)
    }
}
```

## Security Considerations

### Token Validation

1. **Signature Verification**: Tokens are verified using RSA public keys from Azure AD
2. **Issuer Validation**: Ensures tokens are issued by the correct Azure AD tenant
3. **Audience Validation**: Verifies tokens are intended for your application
4. **Expiration Check**: Validates token expiration time

### Key Management

- Public keys are fetched from Azure AD JWKS endpoint in real-time
- No need to manage or rotate keys manually
- Keys are validated against the `kid` (Key ID) in the token header

## Error Handling

The system handles various error scenarios:

- **Missing Configuration**: Returns error if Azure AD config is not set
- **Invalid Token Format**: Handles malformed tokens
- **Expired Tokens**: Specific error for expired tokens
- **Invalid Signature**: Handles signature validation failures
- **Network Issues**: Handles JWKS endpoint connectivity issues

## Testing

### Test Script

Create a test script to verify Azure AD authentication:

```bash
#!/bin/bash

# Test Azure AD authentication
echo "Testing Azure AD JWT Authentication..."

# Replace with actual Azure AD token
AZURE_TOKEN="your-azure-ad-token-here"

# Test protected endpoint
echo "Testing protected endpoint..."
curl -H "Authorization: Bearer $AZURE_TOKEN" \
     -H "Content-Type: application/json" \
     http://localhost:8080/api/users

echo "\nTesting role-based access..."
curl -H "Authorization: Bearer $AZURE_TOKEN" \
     -H "Content-Type: application/json" \
     http://localhost:8080/api/admin/users
```

### Getting Test Tokens

To get Azure AD tokens for testing:

1. Use Azure CLI:
   ```bash
   az account get-access-token --resource your-client-id
   ```

2. Use Postman with Azure AD OAuth 2.0 flow

3. Use your application's login flow

## Integration Examples

### Router Setup

```go
package main

import (
    "main/internal/middleware"
    "github.com/gorilla/mux"
)

func setupRoutes() *mux.Router {
    router := mux.NewRouter()
    
    // Public routes
    router.PathPrefix("/auth").Handler(authRouter)
    router.PathPrefix("/swagger").Handler(swaggerHandler)
    
    // Protected API routes with hybrid authentication
    api := router.PathPrefix("/api").Subrouter()
    api.Use(middleware.HybridAuth) // Accepts both JWT types
    
    // Azure AD only routes
    azureAPI := router.PathPrefix("/azure").Subrouter()
    azureAPI.Use(middleware.AzureADAuth) // Azure AD tokens only
    
    // Role-based routes
    adminAPI := api.PathPrefix("/admin").Subrouter()
    adminAPI.Use(middleware.RequireRole("Admin"))
    
    return router
}
```

## Troubleshooting

### Common Issues

1. **"Azure AD configuration not found"**
   - Ensure `AZURE_AD_TENANT_ID` and `AZURE_AD_CLIENT_ID` are set in `.env`

2. **"Invalid issuer"**
   - Verify the tenant ID is correct
   - Check that the token is from the expected Azure AD tenant

3. **"Invalid audience"**
   - Ensure the client ID matches your Azure AD application

4. **"Key with kid not found"**
   - Token may be from a different tenant or application
   - Check network connectivity to Azure AD JWKS endpoint

5. **"Failed to fetch JWKS"**
   - Network connectivity issues
   - Azure AD service availability

### Debug Mode

For debugging, you can add logging to see token validation details:

```go
// Add debug logging in azure_ad.go
fmt.Printf("Validating token with kid: %s\n", kid)
fmt.Printf("Expected issuer: %s\n", config.Issuer)
fmt.Printf("Expected audience: %s\n", config.ClientID)
```

## Best Practices

1. **Environment Configuration**: Always use environment variables for sensitive configuration
2. **Error Handling**: Implement proper error handling and logging
3. **Token Caching**: Consider caching JWKS keys to reduce API calls
4. **Role Mapping**: Map Azure AD roles to your application's permission system
5. **Monitoring**: Monitor authentication failures and token validation errors
6. **Testing**: Test with both valid and invalid tokens
7. **Documentation**: Keep API documentation updated with authentication requirements

## Migration Guide

### From Regular JWT to Hybrid

1. Update middleware from `JWTAuth` to `HybridAuth`
2. Add Azure AD configuration to environment
3. Test with both token types
4. Update client applications gradually

### Adding Azure AD to Existing Routes

```go
// Before
router.Use(middleware.JWTAuth)

// After (supports both)
router.Use(middleware.HybridAuth)

// Or Azure AD only
router.Use(middleware.AzureADAuth)
```