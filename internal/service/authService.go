package service

import (
	"context"
	"main/db"
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
		return nil, err
	}

	// Check if the password matches
	err = account.CheckPassword(password)
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

func (as *AuthService) Register(username string, password string, roles []model.Role) (*mongo.InsertOneResult, error) {
	var rolesList []model.Role

	for _, role := range roles {
		role, err := as.roleService.GetRoleByName(role.Name)
		if err != nil {
			return nil, err
		}
		rolesList = append(rolesList, *role)
	}

	account := model.Account{
		Username: username,
		Password: password,
		Roles:    rolesList,
	}

	// Hash the password before saving
	err := account.HashPassword()
	if err != nil {
		return nil, err
	}

	// Set timestamps
	account.SetTimestamps()

	rs, err := as.accountCollection.InsertOne(context.TODO(), account)

	if err != nil {
		return nil, err
	}

	return rs, nil
}
