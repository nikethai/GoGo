package service

import (
	"context"
	"main/db"
	"main/model"

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
	var account model.AccountResponse
	err := as.accountCollection.FindOne(context.TODO(),
		bson.D{{"username", username}, {"password", password}}).Decode(&account)
	if err != nil {
		return nil, err
	}
	return &account, nil
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

	rs, err := as.accountCollection.InsertOne(context.TODO(), account)

	if err != nil {
		return nil, err
	}

	return rs, nil
}
