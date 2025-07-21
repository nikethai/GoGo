package service

import (
	"context"
	"main/db"
	"main/internal/model"

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

func (as *RoleService) GetRole(roleId string) (*model.Role, error) {
	var role model.Role
	objId, err := primitive.ObjectIDFromHex(roleId)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"_id": objId}
	er := as.roleCollection.FindOne(context.TODO(), filter).Decode(&role)
	return &role, er
}

func (as *RoleService) GetRoleByName(roleName string) (*model.Role, error) {
	var role model.Role
	filter := bson.M{"name": roleName}
	er := as.roleCollection.FindOne(context.TODO(), filter).Decode(&role)
	return &role, er
}

func (as *RoleService) NewRole(roleName string) (*mongo.InsertOneResult, error) {
	role := model.Role{
		Name: roleName,
	}
	
	// Set timestamps
	role.SetTimestamps()
	
	return as.roleCollection.InsertOne(context.TODO(), role)
}
