package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Form struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	CreateAt    time.Time          `json:"createAt" bson:"createAt"`
	UpdateAt    time.Time          `json:"updateAt" bson:"updateAt"`
	Questions   []Question         `json:"questions" bson:"questions"`
}

func (f *Form) MarshalBSON() ([]byte, error) {
	if f.CreateAt.IsZero() {
		f.CreateAt = time.Now()
	}
	f.UpdateAt = time.Now()
	type my Form
	return bson.Marshal((*my)(f))
}
