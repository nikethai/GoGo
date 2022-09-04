package service

import (
	"context"
	"fmt"
	"main/db"
	"main/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserService struct {
	userCollection    *mongo.Collection
	accountCollection *mongo.Collection
}

func NewUserService() *UserService {
	return &UserService{
		userCollection:    db.MongoDatabase.Collection("user"),
		accountCollection: db.MongoDatabase.Collection("account"),
	}
}

func (us *UserService) GetUserByID(uid string, isAccountId bool) (*model.UserResponse, error) {
	var user model.UserResponse
	var aggSearch bson.M

	id, err := primitive.ObjectIDFromHex(uid)

	if err != nil {
		return nil, err
	}

	aggSearch = bson.M{"$match": bson.M{
		"_id": id,
	}}
	if isAccountId {
		aggSearch = bson.M{"$match": bson.M{
			"accountId": id,
		}}
	}

	aggLookup := bson.M{"$lookup": bson.M{
		"from":         "account",   // collection name in db
		"localField":   "accountId", // field name of children document
		"foreignField": "_id",       // field name of parent document
		"as":           "account",   // new field name in result
	}}

	// to remove the array of account field
	aggUnwind := bson.M{"$unwind": "$account"}

	cursor, err := us.userCollection.Aggregate(context.TODO(), []bson.M{aggSearch, aggLookup, aggUnwind})

	if err != nil {
		return nil, err
	}

	if cursor.Next(context.TODO()) {
		fmt.Println(cursor.Current.Elements())
		err := cursor.Decode(&user)
		if err != nil {
			return nil, err
		}
		return &user, nil
	}

	return nil, mongo.ErrNoDocuments
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
	rs, err := us.userCollection.InsertOne(context.TODO(), newusr)

	accErr := us.accountCollection.FindOneAndUpdate(context.TODO(), bson.M{"_id": accountId}, bson.M{"$set": bson.M{"userId": rs.InsertedID}}).Err()

	if accErr != nil {
		return nil, accErr
	}

	return rs, err
}
