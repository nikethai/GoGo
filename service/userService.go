package service

import (
	"context"
	"main/db"
	"main/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserService struct {
	userCollection *mongo.Collection
}

func NewUserService() *UserService {
	return &UserService{
		userCollection: db.MongoDatabase.Collection("user"),
	}
}

func (us *UserService) GetUserByID(uuid string) (*model.User, error) {
	var user model.User
	err := us.userCollection.FindOne(context.TODO(), bson.D{{"uuid", uuid}}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
