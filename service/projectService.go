package service

import (
	"context"
	"main/db"
	"main/db/builder"
	"main/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProjectService struct {
	projectCollection: *mongo.Collection
}

func NewProjectService() *ProjectService {
	return &ProjectService{}
}
