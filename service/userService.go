package service

import (
	"context"
	"main/db"
	"main/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func (us *UserService) GetUserByID(uid string) (*model.User, error) {
	var user model.User
	id, err := primitive.ObjectIDFromHex(uid)

	if err != nil {
		return nil, err
	}

	aggSearch := bson.M{"$match": bson.M{
		"_id": id,
	}}
	aggLookup := bson.M{"$lookup": bson.M{
		"from":         "account", // collection name in db
		"localField":   "_id",     // field name of children document
		"foreignField": "_id",     // field name of parent document
		"as":           "account", // new field name in result
	}}

	cursor, err := us.userCollection.Aggregate(context.TODO(), []bson.M{aggSearch, aggLookup})

	if err != nil {
		return nil, err
	}

	if cursor.Next(context.TODO()) {
		err := cursor.Decode(&user)
		if err != nil {
			return nil, err
		}
	}

	return &user, nil
}

func (us *UserService) NewUser(reqUser *model.UserRequest, accountId primitive.ObjectID) (*mongo.InsertOneResult, error) {
	newusr := model.User{
		AccountId: accountId,
		Fullname:  reqUser.Fullname,
		DOB:       reqUser.DOB,
		Email:     reqUser.Email,
		Phone:     reqUser.Phone,
		Address:   reqUser.Address,
		Avatar:    reqUser.Avatar,
		Status:    reqUser.Status,
	}
	return us.userCollection.InsertOne(context.TODO(), newusr)
}
