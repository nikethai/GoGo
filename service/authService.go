package service

import (
	"context"
	"errors"
	"main/db"
	"main/model/authModel"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthService struct {
	accountCollection *mongo.Collection
}

func NewAuthService() *AuthService {
	return &AuthService{
		accountCollection: db.MongoDatabase.Collection("account"),
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
	newUuid, err := uuid.NewRandom()

	if err != nil {
		return nil, err
	}

	account := authModel.Account{
		Uuid:     newUuid.String(),
		Username: username,
		Password: password,
		Role:     roles,
	}

	rs, err := as.accountCollection.InsertOne(context.TODO(), account)

	if err != nil {
		return nil, err
	}

	return rs, nil
}
