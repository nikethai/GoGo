# Azure AD JWT Authentication for Gogo API

This document provides a comprehensive guide for implementing and using Azure AD JWT authentication in the Gogo API.

## Overview

The Gogo API now supports dual authentication modes:
1. **Regular JWT Authentication** - Traditional JWT tokens signed with a secret key
2. **Azure AD JWT Authentication** - JWT tokens issued by Azure Active Directory
3. **Hybrid Authentication** - Supports both authentication types simultaneously

## Features

✅ **Azure AD Token Validation** - Validates JWT tokens issued by Azure AD
✅ **Public Key Verification** - Retrieves and caches Azure AD public keys from JWKS endpoint
✅ **Claims Mapping** - Maps Azure AD claims to internal user structure
✅ **Role-Based Access Control** - Supports Azure AD roles and groups
✅ **Hybrid Authentication** - Backward compatibility with existing JWT tokens
✅ **Middleware Integration** - Easy integration with existing router structure
✅ **Environment Configuration** - Configurable via environment variables
✅ **Comprehensive Testing** - Includes test scripts and examples

## Quick Start

### 1. Environment Setup

Add the following environment variables to your `.env` file:

```bash
# Existing variables
MONGODB_URI=mongodb://localhost:27017/gogo
JWT_SECRET_KEY=your-secret-key-here

# New Azure AD variables
AZURE_AD_TENANT_ID=your-tenant-id-here
AZURE_AD_CLIENT_ID=your-client-id-here
```

### 2. Basic Usage

```go
package main

import (
    "main/internal/middleware"
    "github.com/gorilla/mux"
)

func main() {
    router := mux.NewRouter()
    
    // Use hybrid authentication (supports both JWT types)
    api := router.PathPrefix("/api").Subrouter()
    api.Use(middleware.HybridAuth)
    
    // Your routes here
    api.HandleFunc("/users", getUsersHandler).Methods("GET")
    
    // Start server
    http.ListenAndServe(":8080", router)
}
```

### 3. Testing

Run the test script to verify the implementation:

```bash
./scripts/test_azure_auth.sh
```

## Authentication Modes

### 1. Regular JWT Authentication

```go
// Only accepts regular JWT tokens
api.Use(middleware.JWTAuth)
```

**Token Format:**
```
Authorization: Bearer <regular_jwt_token>
```

### 2. Azure AD Authentication

```go
// Only accepts Azure AD JWT tokens
api.Use(middleware.AzureADAuth)
```

**Token Format:**
```
Authorization: Bearer <azure_ad_jwt_token>
```

### 3. Hybrid Authentication (Recommended)

```go
// Accepts both regular JWT and Azure AD tokens
api.Use(middleware.HybridAuth)
```

**Supports both token formats above**

## Role-Based Access Control

### Regular JWT Roles

```go
// Requires specific role from regular JWT
api.Use(middleware.RequireRole("Admin"))
```

### Azure AD Roles

```go
// Requires specific role from Azure AD token
api.Use(middleware.RequireAzureADRole("Admin"))
```

### Universal Role Check

```go
// Works with both authentication types
api.Use(middleware.RequireRole("Admin"))
```

## Context Information

After authentication, you can access user information from the request context:

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    // Standard information (available for both auth types)
    userID, _ := middleware.GetUserIDFromContext(r)
    username, _ := middleware.GetUsernameFromContext(r)
    roles, _ := middleware.GetUserRolesFromContext(r)
    authType, _ := middleware.GetAuthTypeFromContext(r)
    
    // Azure AD specific information (only when authType == "azure_ad")
    if authType == "azure_ad" {
        tenantID, _ := middleware.GetAzureTenantIDFromContext(r)
        appID, _ := middleware.GetAzureAppIDFromContext(r)
        objectID, _ := middleware.GetAzureObjectIDFromContext(r)
    }
}
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `AZURE_AD_TENANT_ID` | Yes | Your Azure AD tenant ID |
| `AZURE_AD_CLIENT_ID` | Yes | Your Azure AD application (client) ID |
| `JWT_SECRET_KEY` | Yes* | Secret key for regular JWT tokens |

*Required only if using regular JWT authentication

### Azure AD Setup

1. **Register Application in Azure AD**
   - Go to Azure Portal → Azure Active Directory → App registrations
   - Click "New registration"
   - Configure redirect URIs if needed
   - Note the Application (client) ID and Directory (tenant) ID

2. **Configure Token Settings**
   - Go to Token configuration
   - Add optional claims if needed
   - Configure group claims for role-based access

3. **API Permissions**
   - Add necessary API permissions
   - Grant admin consent if required

## Migration Guide

### From Regular JWT to Hybrid

1. **Add Environment Variables**
   ```bash
   AZURE_AD_TENANT_ID=your-tenant-id
   AZURE_AD_CLIENT_ID=your-client-id
   ```

2. **Update Middleware**
   ```go
   // Before
   api.Use(middleware.JWTAuth)
   
   // After
   api.Use(middleware.HybridAuth)
   ```

3. **Test Both Authentication Types**
   - Existing JWT tokens should continue working

## OAuth 2.0 Authorization Code Flow

The Gogo API now fully supports the OAuth 2.0 Authorization Code Flow with PKCE (Proof Key for Code Exchange) for enhanced security when integrating with Azure AD.

### Endpoints

- **`/auth/azure/login`**
  - Initiates the OAuth 2.0 Authorization Code Flow.
  - Redirects the user to the Azure AD login page.
  - Automatically generates and includes PKCE `code_challenge` and `state` parameters for security.

- **`/auth/azure/callback`**
  - Handles the callback from Azure AD after successful user authentication.
  - Exchanges the authorization `code` for `access_token`, `refresh_token`, and `id_token`.
  - Validates the `state` parameter to prevent CSRF attacks.
  - Stores session information and tokens securely.

### Security Features

- **PKCE Support**: Protects against authorization code interception attacks by verifying the client application during the token exchange process.
- **State Parameter Validation**: Ensures that the authorization response from Azure AD is legitimate and prevents Cross-Site Request Forgery (CSRF) attacks.

### Usage Example (Conceptual Flow)

1. **Initiate Login**: Your client application redirects the user to `/auth/azure/login`.
2. **Azure AD Authentication**: The user authenticates with Azure AD.
3. **Callback Handling**: Azure AD redirects back to `/auth/azure/callback` with an authorization code.
4. **Token Exchange**: The Gogo API backend exchanges the code for tokens.
5. **Session Establishment**: A secure session is established for the user.
   - Azure AD tokens should now be accepted

### Gradual Migration Strategy

```go
// Phase 1: Legacy routes (existing functionality)
v1 := router.PathPrefix("/api/v1").Subrouter()
v1.Use(middleware.JWTAuth) // Regular JWT only

// Phase 2: Hybrid routes (supports both)
v2 := router.PathPrefix("/api/v2").Subrouter()
v2.Use(middleware.HybridAuth) // Both JWT types

// Phase 3: Azure AD only routes (new features)
v3 := router.PathPrefix("/api/v3").Subrouter()
v3.Use(middleware.AzureADAuth) // Azure AD only
```

## Examples

See `examples/azure_router_integration.go` for comprehensive integration examples including:

- Basic hybrid authentication setup
- Role-based routing
- Conditional authentication
- Migration strategies
- Handler examples

## Testing

### Automated Testing

```bash
# Run the test script
./scripts/test_azure_auth.sh

# Test specific endpoints
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/users
```

### Manual Testing with Real Azure AD Tokens

1. **Get Azure AD Token**
   ```bash
   # Using Azure CLI
   az account get-access-token --resource <your-client-id>
   ```

2. **Test API Endpoints**
   ```bash
   curl -H "Authorization: Bearer <azure_ad_token>" \
        http://localhost:8080/api/users
   ```

### Token Validation Testing

```bash
# Test with invalid token
curl -H "Authorization: Bearer invalid_token" \
     http://localhost:8080/api/users

# Test without token
curl http://localhost:8080/api/users

# Test with expired token
curl -H "Authorization: Bearer <expired_token>" \
     http://localhost:8080/api/users
```

## Security Considerations

### Token Validation
- ✅ Signature verification using Azure AD public keys
- ✅ Issuer validation (Azure AD tenant)
- ✅ Audience validation (your application)
- ✅ Expiration time validation
- ✅ Not before time validation

### Best Practices
- Use HTTPS in production
- Implement token refresh mechanisms
- Set appropriate token expiration times
- Monitor authentication failures
- Implement rate limiting
- Log security events

### JWKS Caching
- Public keys are cached for 1 hour
- Automatic refresh on cache miss
- Fallback to fresh fetch on validation failure

## Troubleshooting

### Common Issues

1. **"Invalid token" errors**
   - Check token format and encoding
   - Verify token hasn't expired
   - Ensure correct audience and issuer

2. **"Failed to get public key" errors**
   - Check internet connectivity
   - Verify Azure AD tenant ID
   - Check JWKS endpoint accessibility

3. **"Invalid audience" errors**
   - Verify AZURE_AD_CLIENT_ID matches token audience
   - Check token was issued for correct application

4. **"Invalid issuer" errors**
   - Verify AZURE_AD_TENANT_ID is correct
   - Check token issuer matches expected format

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
// Add debug logging in your main function
log.SetLevel(log.DebugLevel)
```

### Health Check

Test Azure AD configuration:

```bash
curl http://localhost:8080/health/azure-ad
```

## API Reference

### Middleware Functions

- `middleware.JWTAuth` - Regular JWT authentication only
- `middleware.AzureADAuth` - Azure AD JWT authentication only
- `middleware.HybridAuth` - Both authentication types
- `middleware.RequireRole(role)` - Role-based access control
- `middleware.RequireAzureADRole(role)` - Azure AD specific role check

### Context Helper Functions

- `GetUserIDFromContext(r)` - Get user ID
- `GetUsernameFromContext(r)` - Get username
- `GetUserRolesFromContext(r)` - Get user roles
- `GetAuthTypeFromContext(r)` - Get authentication type
- `GetAzureTenantIDFromContext(r)` - Get Azure AD tenant ID
- `GetAzureAppIDFromContext(r)` - Get Azure AD app ID
- `GetAzureObjectIDFromContext(r)` - Get Azure AD object ID

### Core Functions

- `auth.ValidateAzureADToken(tokenString)` - Validate Azure AD token
- `auth.GetAzureADConfig()` - Get Azure AD configuration
- `auth.ConvertAzureADClaimsToJWTClaims(claims)` - Convert claims

## Performance Considerations

- **JWKS Caching**: Public keys are cached to reduce external API calls
- **Token Parsing**: Efficient JWT parsing and validation
- **Memory Usage**: Minimal memory footprint for token validation
- **Concurrent Requests**: Thread-safe implementation

## Monitoring and Logging

### Metrics to Monitor
- Authentication success/failure rates
- Token validation latency
- JWKS fetch frequency
- Role-based access patterns

### Log Events
- Authentication attempts
- Token validation failures
- JWKS refresh events
- Role-based access denials

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review the examples in `examples/azure_router_integration.go`
3. Run the test script: `./scripts/test_azure_auth.sh`
4. Check the detailed documentation in `docs/AZURE_AD_JWT.md`

## License

This implementation is part of the Gogo API project and follows the same license terms.