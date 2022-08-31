package service

import (
	"context"
	"main/db"
	"main/model/authModel"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RoleService struct {
	roleCollection *mongo.Collection
}

func NewRoleService() *RoleService {
	return &RoleService{
		roleCollection: db.MongoDatabase.Collection("role"),
	}
}

func (as *RoleService) GetRole(roleId string) (*authModel.Role, error) {
	var role authModel.Role
	objId, err := primitive.ObjectIDFromHex(roleId)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"_id": objId}
	er := as.roleCollection.FindOne(context.TODO(), filter).Decode(&role)
	return &role, er
}

func (as *RoleService) GetRoleByName(roleName string) (*authModel.Role, error) {
	var role authModel.Role
	filter := bson.M{"name": roleName}
	er := as.roleCollection.FindOne(context.TODO(), filter).Decode(&role)
	return &role, er
}

func (as *RoleService) NewRole(roleName string) (*mongo.InsertOneResult, error) {
	role := authModel.Role{
		Name: roleName,
	}
	return as.roleCollection.InsertOne(context.TODO(), role)
}
