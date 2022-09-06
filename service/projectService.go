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
	projectCollection *mongo.Collection
}

func NewProjectService() *ProjectService {
	return &ProjectService{
		projectCollection: db.MongoDatabase.Collection("project"),
	}
}

func (p *ProjectService) GetProjects() (*[]model.ProjectResponse, error) {
	var projects []model.ProjectResponse

	aggLookup := builder.Lookup("user", "createBy", "_id", "createBy")
	aggUnwind := builder.Unwind("createBy")

	cursor, err := p.projectCollection.Aggregate(context.TODO(), []bson.M{aggLookup, aggUnwind})

	if err != nil {
		return nil, err
	}

	if err = cursor.All(context.TODO(), &projects); err != nil {
		return nil, err
	}

	return &projects, nil
}

func (p *ProjectService) GetProjectById(pid string) (*model.Project, error) {
	return builder.GetById[model.Project](p.projectCollection, pid)
}

func (p *ProjectService) CreateProject(project *model.Project) (*mongo.InsertOneResult, error) {
	return p.projectCollection.InsertOne(context.TODO(), project)
}
