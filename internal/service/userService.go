package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"main/internal/model"
	"main/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserService demonstrates the new generic repository pattern
type UserService struct {
	userRepo    repository.Repository[*model.User]
	accountRepo repository.Repository[*model.Account]
}

// NewUserService creates a new user service using the generic repository
func NewUserService(userRepo repository.Repository[*model.User], accountRepo repository.Repository[*model.Account]) *UserService {
	return &UserService{
		userRepo:    userRepo,
		accountRepo: accountRepo,
	}
}

// CreateUser creates a new user with account
func (s *UserService) CreateUser(ctx context.Context, userReq *model.UserRequest, accountReq *model.AccountRequest) (*model.User, error) {
	// Check if account already exists
	existingAccount, err := s.accountRepo.GetByField(ctx, "username", accountReq.Username)
	if err == nil && existingAccount != nil {
		return nil, errors.New("username already exists")
	}

	// Create account first
	account := &model.Account{
		Username: accountReq.Username,
		Password: accountReq.Password,
		Roles:    []model.Role{}, // Default empty roles
	}

	// Hash password and set timestamps
	err = account.HashPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	account.SetTimestamps()

	createdAccount, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Create user
	user := &model.User{
		AccountId: createdAccount.GetID(),
		Fullname:  userReq.Fullname,
		Email:     userReq.Email,
		Status:    "active",
	}

	// Set timestamps for user
	user.SetTimestamps()

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		// Rollback account creation if user creation fails
		_ = s.accountRepo.Delete(ctx, createdAccount.GetID())
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return createdUser, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID primitive.ObjectID) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// GetUserWithAccount retrieves a user with their account information using aggregation
func (s *UserService) GetUserWithAccount(ctx context.Context, userID primitive.ObjectID) (bson.M, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"_id": userID}},
		{"$lookup": bson.M{
			"from":         "accounts",
			"localField":   "accountId",
			"foreignField": "_id",
			"as":           "account",
		}},
		{"$unwind": "$account"},
		// Project only the fields we want, excluding password for security
		{"$project": bson.M{
			"_id":        1,
			"accountId":  1,
			"fullname":   1,
			"dob":        1,
			"email":      1,
			"phone":      1,
			"address":    1,
			"avatar":     1,
			"status":     1,
			"createdAt":  1,
			"updatedAt":  1,
			"account": bson.M{
				"_id":       "$account._id",
				"username":  "$account.username",
				"roles":     "$account.roles",
				"createdAt": "$account.createdAt",
				"updatedAt": "$account.updatedAt",
				// Note: password field is intentionally omitted for security
			},
		}},
	}

	results, err := s.userRepo.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.New("user not found")
	}

	return results[0], nil
}

// ListUsers retrieves users with pagination
func (s *UserService) ListUsers(ctx context.Context, page, limit int) (*repository.ListResult[*model.User], error) {
	opts := repository.ListOptions{
		Page:   page,
		Limit:  limit,
		Sort:   map[string]int{"fullname": 1}, // Sort by fullname ascending
		Filter: bson.M{},
	}

	result, err := s.userRepo.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(ctx context.Context, userID primitive.ObjectID, updates bson.M) (*model.User, error) {
	// Add updated timestamp
	updates["updatedAt"] = time.Now()
	return s.userRepo.Update(ctx, userID, updates)
}

// DeleteUser deletes a user and their account
func (s *UserService) DeleteUser(ctx context.Context, userID primitive.ObjectID) error {
	// Get user first to find account ID
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Delete user
	err = s.userRepo.Delete(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Delete associated account
	err = s.accountRepo.Delete(ctx, user.AccountId)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to delete account %s: %v\n", user.AccountId.Hex(), err)
	}

	return nil
}

// SearchUsersByEmail searches users by email pattern
func (s *UserService) SearchUsersByEmail(ctx context.Context, emailPattern string) ([]*model.User, error) {
	filter := bson.M{
		"email": bson.M{"$regex": emailPattern, "$options": "i"},
	}

	return s.userRepo.GetAll(ctx, filter, &options.FindOptions{
		Sort: bson.M{"email": 1},
	})
}

// CountActiveUsers counts active users
func (s *UserService) CountActiveUsers(ctx context.Context) (int64, error) {
	filter := bson.M{"status": "active"}
	return s.userRepo.Count(ctx, filter)
}

// UserExists checks if a user exists
func (s *UserService) UserExists(ctx context.Context, userID primitive.ObjectID) (bool, error) {
	return s.userRepo.Exists(ctx, userID)
}

// CreateUserForExistingAccount creates a user for an existing account (migration compatibility)
func (s *UserService) CreateUserForExistingAccount(ctx context.Context, userReq *model.UserRequest) (*model.User, error) {
	// Verify the account exists
	exists, err := s.accountRepo.Exists(ctx, userReq.AccountId)
	if err != nil {
		return nil, fmt.Errorf("failed to check account existence: %w", err)
	}
	if !exists {
		return nil, errors.New("account not found")
	}

	// Create user
	user := &model.User{
		AccountId: userReq.AccountId,
		Fullname:  userReq.Fullname,
		DOB:       userReq.DOB,
		Email:     userReq.Email,
		Phone:     userReq.Phone,
		Address:   userReq.Address,
		Avatar:    userReq.Avatar,
		Status:    userReq.Status,
	}

	// Set timestamps
	user.SetTimestamps()

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Update account with user ID (maintaining old behavior)
	updates := bson.M{
		"userId":    createdUser.GetID(),
		"updatedAt": time.Now(),
	}
	_, err = s.accountRepo.Update(ctx, userReq.AccountId, updates)
	if err != nil {
		// Log error but don't fail the operation since user was created
		fmt.Printf("Warning: failed to update account with user ID: %v\n", err)
	}

	return createdUser, nil
}

// GetUserByAccountID retrieves a user by account ID (migration compatibility)
func (s *UserService) GetUserByAccountID(ctx context.Context, accountID primitive.ObjectID) (bson.M, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"accountId": accountID}},
		{"$lookup": bson.M{
			"from":         "account",
			"localField":   "accountId",
			"foreignField": "_id",
			"as":           "account",
		}},
		{"$unwind": "$account"},
		// Project only the fields we want, excluding password for security
		{"$project": bson.M{
			"_id":        1,
			"accountId":  1,
			"fullname":   1,
			"dob":        1,
			"email":      1,
			"phone":      1,
			"address":    1,
			"avatar":     1,
			"status":     1,
			"createdAt":  1,
			"updatedAt":  1,
			"account": bson.M{
				"_id":       "$account._id",
				"username":  "$account.username",
				"roles":     "$account.roles",
				"createdAt": "$account.createdAt",
				"updatedAt": "$account.updatedAt",
				// Note: password field is intentionally omitted for security
			},
		}},
	}

	results, err := s.userRepo.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.New("user not found")
	}

	return results[0], nil
}
