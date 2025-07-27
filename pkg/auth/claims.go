package auth

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ClaimsManager handles Firebase custom claims operations
type ClaimsManager struct {
	firebaseService *FirebaseService
}

// UserRole represents a user role in the system
type UserRole struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UserClaims represents the custom claims structure for Firebase
type UserClaims struct {
	Roles       []string               `json:"roles,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	Department  string                 `json:"department,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
	UpdatedAt   int64                  `json:"updated_at"`
}

// Permission represents a system permission
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

// Standard system roles
var (
	RoleAdmin = UserRole{
		ID:          "admin",
		Name:        "Administrator",
		Description: "Full system access",
		Permissions: []string{"*"},
	}

	RoleUser = UserRole{
		ID:          "user",
		Name:        "User",
		Description: "Standard user access",
		Permissions: []string{"read:profile", "update:profile", "read:forms", "submit:forms"},
	}

	RoleManager = UserRole{
		ID:          "manager",
		Name:        "Manager",
		Description: "Management access",
		Permissions: []string{"read:*", "create:forms", "update:forms", "delete:forms", "manage:users"},
	}

	RoleViewer = UserRole{
		ID:          "viewer",
		Name:        "Viewer",
		Description: "Read-only access",
		Permissions: []string{"read:forms", "read:profile"},
	}
)

// Standard system permissions
var (
	PermissionReadProfile = Permission{
		ID:          "read:profile",
		Name:        "Read Profile",
		Description: "Read user profile information",
		Resource:    "profile",
		Action:      "read",
	}

	PermissionUpdateProfile = Permission{
		ID:          "update:profile",
		Name:        "Update Profile",
		Description: "Update user profile information",
		Resource:    "profile",
		Action:      "update",
	}

	PermissionReadForms = Permission{
		ID:          "read:forms",
		Name:        "Read Forms",
		Description: "Read form data",
		Resource:    "forms",
		Action:      "read",
	}

	PermissionCreateForms = Permission{
		ID:          "create:forms",
		Name:        "Create Forms",
		Description: "Create new forms",
		Resource:    "forms",
		Action:      "create",
	}

	PermissionUpdateForms = Permission{
		ID:          "update:forms",
		Name:        "Update Forms",
		Description: "Update existing forms",
		Resource:    "forms",
		Action:      "update",
	}

	PermissionDeleteForms = Permission{
		ID:          "delete:forms",
		Name:        "Delete Forms",
		Description: "Delete forms",
		Resource:    "forms",
		Action:      "delete",
	}

	PermissionSubmitForms = Permission{
		ID:          "submit:forms",
		Name:        "Submit Forms",
		Description: "Submit form responses",
		Resource:    "forms",
		Action:      "submit",
	}

	PermissionManageUsers = Permission{
		ID:          "manage:users",
		Name:        "Manage Users",
		Description: "Manage user accounts",
		Resource:    "users",
		Action:      "manage",
	}
)

// NewClaimsManager creates a new claims manager instance
func NewClaimsManager(firebaseService *FirebaseService) *ClaimsManager {
	return &ClaimsManager{
		firebaseService: firebaseService,
	}
}

// SetUserClaims sets custom claims for a user
func (cm *ClaimsManager) SetUserClaims(ctx context.Context, uid string, claims *UserClaims) error {
	if claims == nil {
		return fmt.Errorf("claims cannot be nil")
	}

	// Set updated timestamp
	claims.UpdatedAt = time.Now().Unix()

	// Convert to map for Firebase
	claimsMap := map[string]interface{}{
		"updated_at": claims.UpdatedAt,
	}

	if len(claims.Roles) > 0 {
		claimsMap["roles"] = claims.Roles
	}

	if len(claims.Permissions) > 0 {
		claimsMap["permissions"] = claims.Permissions
	}

	if claims.TenantID != "" {
		claimsMap["tenant_id"] = claims.TenantID
	}

	if claims.Department != "" {
		claimsMap["department"] = claims.Department
	}

	if claims.Custom != nil {
		for key, value := range claims.Custom {
			claimsMap[key] = value
		}
	}

	return cm.firebaseService.SetCustomClaims(ctx, uid, claimsMap)
}

// GetUserClaims retrieves custom claims for a user
func (cm *ClaimsManager) GetUserClaims(ctx context.Context, uid string) (*UserClaims, error) {
	user, err := cm.firebaseService.GetUser(ctx, uid)
	if err != nil {
		return nil, err
	}

	claims := &UserClaims{
		Custom: make(map[string]interface{}),
	}

	// Extract roles
	if roles, ok := user.CustomClaims["roles"].([]interface{}); ok {
		claims.Roles = make([]string, len(roles))
		for i, role := range roles {
			if roleStr, ok := role.(string); ok {
				claims.Roles[i] = roleStr
			}
		}
	}

	// Extract permissions
	if permissions, ok := user.CustomClaims["permissions"].([]interface{}); ok {
		claims.Permissions = make([]string, len(permissions))
		for i, perm := range permissions {
			if permStr, ok := perm.(string); ok {
				claims.Permissions[i] = permStr
			}
		}
	}

	// Extract other standard claims
	if tenantID, ok := user.CustomClaims["tenant_id"].(string); ok {
		claims.TenantID = tenantID
	}

	if department, ok := user.CustomClaims["department"].(string); ok {
		claims.Department = department
	}

	if updatedAt, ok := user.CustomClaims["updated_at"].(float64); ok {
		claims.UpdatedAt = int64(updatedAt)
	}

	// Extract custom claims
	for key, value := range user.CustomClaims {
		if key != "roles" && key != "permissions" && key != "tenant_id" && key != "department" && key != "updated_at" {
			claims.Custom[key] = value
		}
	}

	return claims, nil
}

// AddUserRole adds a role to a user
func (cm *ClaimsManager) AddUserRole(ctx context.Context, uid string, roleID string) error {
	claims, err := cm.GetUserClaims(ctx, uid)
	if err != nil {
		// If user doesn't exist or has no claims, create new claims
		claims = &UserClaims{
			Roles:  []string{},
			Custom: make(map[string]interface{}),
		}
	}

	// Check if role already exists
	for _, role := range claims.Roles {
		if role == roleID {
			return nil // Role already exists
		}
	}

	// Add the new role
	claims.Roles = append(claims.Roles, roleID)

	// Update permissions based on role
	if err := cm.updatePermissionsForRoles(claims); err != nil {
		return err
	}

	return cm.SetUserClaims(ctx, uid, claims)
}

// RemoveUserRole removes a role from a user
func (cm *ClaimsManager) RemoveUserRole(ctx context.Context, uid string, roleID string) error {
	claims, err := cm.GetUserClaims(ctx, uid)
	if err != nil {
		return err
	}

	// Remove the role
	newRoles := []string{}
	for _, role := range claims.Roles {
		if role != roleID {
			newRoles = append(newRoles, role)
		}
	}
	claims.Roles = newRoles

	// Update permissions based on remaining roles
	if err := cm.updatePermissionsForRoles(claims); err != nil {
		return err
	}

	return cm.SetUserClaims(ctx, uid, claims)
}

// SetUserRoles sets all roles for a user (replaces existing roles)
func (cm *ClaimsManager) SetUserRoles(ctx context.Context, uid string, roleIDs []string) error {
	claims, err := cm.GetUserClaims(ctx, uid)
	if err != nil {
		// If user doesn't exist or has no claims, create new claims
		claims = &UserClaims{
			Custom: make(map[string]interface{}),
		}
	}

	claims.Roles = roleIDs

	// Update permissions based on roles
	if err := cm.updatePermissionsForRoles(claims); err != nil {
		return err
	}

	return cm.SetUserClaims(ctx, uid, claims)
}

// HasRole checks if a user has a specific role
func (cm *ClaimsManager) HasRole(ctx context.Context, uid string, roleID string) (bool, error) {
	claims, err := cm.GetUserClaims(ctx, uid)
	if err != nil {
		return false, err
	}

	for _, role := range claims.Roles {
		if role == roleID {
			return true, nil
		}
	}

	return false, nil
}

// HasPermission checks if a user has a specific permission
func (cm *ClaimsManager) HasPermission(ctx context.Context, uid string, permission string) (bool, error) {
	claims, err := cm.GetUserClaims(ctx, uid)
	if err != nil {
		return false, err
	}

	// Check for wildcard permission (admin)
	for _, perm := range claims.Permissions {
		if perm == "*" {
			return true, nil
		}
		if perm == permission {
			return true, nil
		}
		// Check for wildcard resource permissions (e.g., "read:*")
		if strings.HasSuffix(perm, ":*") {
			resource := strings.TrimSuffix(perm, ":*")
			if strings.HasPrefix(permission, resource+":") {
				return true, nil
			}
		}
	}

	return false, nil
}

// updatePermissionsForRoles updates the permissions based on the user's roles
func (cm *ClaimsManager) updatePermissionsForRoles(claims *UserClaims) error {
	permissionsSet := make(map[string]bool)

	// Get permissions for each role
	for _, roleID := range claims.Roles {
		role := cm.getRoleByID(roleID)
		if role != nil {
			for _, permission := range role.Permissions {
				permissionsSet[permission] = true
			}
		}
	}

	// Convert set to slice
	claims.Permissions = make([]string, 0, len(permissionsSet))
	for permission := range permissionsSet {
		claims.Permissions = append(claims.Permissions, permission)
	}

	return nil
}

// getRoleByID returns a role by its ID
func (cm *ClaimsManager) getRoleByID(roleID string) *UserRole {
	switch roleID {
	case "admin":
		return &RoleAdmin
	case "user":
		return &RoleUser
	case "manager":
		return &RoleManager
	case "viewer":
		return &RoleViewer
	default:
		return nil
	}
}

// GetAvailableRoles returns all available system roles
func (cm *ClaimsManager) GetAvailableRoles() []UserRole {
	return []UserRole{
		RoleAdmin,
		RoleUser,
		RoleManager,
		RoleViewer,
	}
}

// GetAvailablePermissions returns all available system permissions
func (cm *ClaimsManager) GetAvailablePermissions() []Permission {
	return []Permission{
		PermissionReadProfile,
		PermissionUpdateProfile,
		PermissionReadForms,
		PermissionCreateForms,
		PermissionUpdateForms,
		PermissionDeleteForms,
		PermissionSubmitForms,
		PermissionManageUsers,
	}
}

// ValidateRole validates if a role ID is valid
func (cm *ClaimsManager) ValidateRole(roleID string) bool {
	return cm.getRoleByID(roleID) != nil
}

// ValidatePermission validates if a permission is valid
func (cm *ClaimsManager) ValidatePermission(permission string) bool {
	if permission == "*" {
		return true
	}

	for _, perm := range cm.GetAvailablePermissions() {
		if perm.ID == permission {
			return true
		}
	}

	// Check for wildcard permissions
	if strings.HasSuffix(permission, ":*") {
		resource := strings.TrimSuffix(permission, ":*")
		validResources := []string{"profile", "forms", "users", "projects", "questions", "roles"}
		for _, validResource := range validResources {
			if resource == validResource {
				return true
			}
		}
	}

	return false
}

// Global claims manager instance
var claimsManager *ClaimsManager

// InitClaimsManager initializes the global claims manager
func InitClaimsManager(firebaseService *FirebaseService) {
	claimsManager = NewClaimsManager(firebaseService)
}

// GetClaimsManager returns the global claims manager instance
func GetClaimsManager() *ClaimsManager {
	return claimsManager
}