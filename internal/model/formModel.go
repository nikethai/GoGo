package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Form struct {
	BaseModel   `bson:",inline"`
	Name        string               `json:"name" bson:"name"`
	Description string               `json:"description" bson:"description"`
	Questions   []primitive.ObjectID `json:"questions" bson:"questions"` // list of question id (new id for each form)
}
