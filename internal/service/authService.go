package service

import (
	"context"
	"errors"
	"log"
	"main/db"
	customError "main/internal/error"
	"main/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthService struct {
	accountCollection *mongo.Collection
	roleService       *RoleService
}

func NewAuthService() *AuthService {
	return &AuthService{
		accountCollection: db.MongoDatabase.Collection("account"),
		roleService:       NewRoleService(),
	}
}

func (as *AuthService) Login(username string, password string) (*model.AccountResponse, error) {
	// First find the account by username
	var account model.Account
	err := as.accountCollection.FindOne(context.TODO(),
		bson.D{{Key: "username", Value: username}}).Decode(&account)
	if err != nil {
		log.Printf("Login error for username %s: %v", username, err)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, customError.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if the password matches
	err = account.CheckPassword(password)
	if err != nil {
		log.Printf("Password check error for username %s: %v", username, err)
		return nil, err
	}

	// Return account response without password
	accountResponse := &model.AccountResponse{
		ID:       account.ID,
		Username: account.Username,
		Roles:    account.Roles,
	}
	return accountResponse, nil
}

func (as *AuthService) Register(username string, password string, email string, roles []model.Role) (*model.AccountResponse, error) {
	// Check if username already exists
	var existingAccount model.Account
	err := as.accountCollection.FindOne(context.TODO(), bson.M{"username": username}).Decode(&existingAccount)
	if err == nil {
		// Username already exists
		return nil, customError.ErrDuplicateUsername
	} else if err != mongo.ErrNoDocuments {
		// Some other error occurred
		return nil, err
	}

	// Check if email already exists (in user collection)
	var existingUser model.User
	userCollection := db.MongoDatabase.Collection("user")
	err = userCollection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&existingUser)
	if err == nil {
		// Email already exists
		return nil, customError.ErrDuplicateEmail
	} else if err != mongo.ErrNoDocuments {
		// Some other error occurred
		return nil, err
	}

	var rolesList []model.Role

	// If no roles provided, assign default user role
	if len(roles) == 0 {
		userRole, err := as.roleService.GetRoleByName("user")
		if err != nil {
			return nil, err
		}
		rolesList = append(rolesList, *userRole)
	} else {
		// Process provided roles
		for _, role := range roles {
			role, err := as.roleService.GetRoleByName(role.Name)
			if err != nil {
				return nil, err
			}
			rolesList = append(rolesList, *role)
		}
	}

	account := model.Account{
		Username: username,
		Password: password,
		Roles:    rolesList,
	}

	// Hash the password before saving
	err = account.HashPassword()
	if err != nil {
		return nil, err
	}

	// Set timestamps
	account.SetTimestamps()

	_, err = as.accountCollection.InsertOne(context.TODO(), account)
	if err != nil {
		return nil, err
	}

	// Return account response without password
	accountResponse := &model.AccountResponse{
		ID:       account.ID,
		Username: account.Username,
		Roles:    account.Roles,
	}
	return accountResponse, nil
}
