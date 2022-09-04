package service

import (
	"context"
	"main/db"
	"main/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type QuestionService struct {
	questionCollection *mongo.Collection
}

func NewQuestionService() *QuestionService {
	return &QuestionService{
		questionCollection: db.MongoDatabase.Collection("question"),
	}
}

func (qs *QuestionService) GetQuestionById(id string) (*model.Question, error) {
	var question model.Question
	err := qs.questionCollection.FindOne(context.TODO(), bson.D{{"id", id}}).Decode(&question)
	if err != nil {
		return nil, err
	}
	return &question, nil
}

func (qs *QuestionService) GetAllQuestions() (*[]model.Question, error) {
	var questions []model.Question
	cursor, err := qs.questionCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &questions); err != nil {
		return nil, err
	}
	return &questions, nil
}

func (qs *QuestionService) CreateQuestion(question *model.Question) (*mongo.InsertOneResult, error) {
	newUuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	question.Uuid = newUuid.String()

	rs, err := qs.questionCollection.InsertOne(context.TODO(), question)
	if err != nil {
		return nil, err
	}
	return rs, nil
}
