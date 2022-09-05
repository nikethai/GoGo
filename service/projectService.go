package service

type ProjectService struct {
	projectCollection: *mongo.Collection
}

func NewProjectService() *ProjectService {
	return &ProjectService{}
}
