package service

import (
	"context"
	"errors"
	"main/db"
	"main/model/authModel"

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

func (as *AuthService) Login(username string, password string) (*authModel.Account, error) {
	var account authModel.Account
	err := as.accountCollection.FindOne(context.TODO(), bson.D{{"username", username}}).Decode(&account)
	if err != nil {
		return nil, err
	}
	if account.Password != password {
		return nil, errors.New("incorrect password")
	}
	return &account, nil
}

func (as *AuthService) Register(username string, password string, roles []authModel.Role) (*mongo.InsertOneResult, error) {
	var rolesList []authModel.Role

	for _, role := range roles {
		role, err := as.roleService.GetRoleByName(role.Name)
		if err != nil {
			return nil, err
		}
		rolesList = append(rolesList, *role)
	}

	account := authModel.Account{
		Username: username,
		Password: password,
		Role:     rolesList,
	}

	rs, err := as.accountCollection.InsertOne(context.TODO(), account)

	if err != nil {
		return nil, err
	}

	return rs, nil
}
