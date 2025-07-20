package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Form struct {
	ID          primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name        string               `json:"name" bson:"name"`
	Description string               `json:"description" bson:"description"`
	CreateAt    time.Time            `json:"createAt" bson:"createAt"`
	UpdateAt    time.Time            `json:"updateAt" bson:"updateAt"`
	Questions   []primitive.ObjectID `json:"questions" bson:"questions"` // list of question id (new id for each form)
}

// GetID implements the Entity interface
func (f *Form) GetID() primitive.ObjectID {
	return f.ID
}

// SetID implements the Entity interface
func (f *Form) SetID(id primitive.ObjectID) {
	f.ID = id
}

func (f *Form) MarshalBSON() ([]byte, error) {
	if f.CreateAt.IsZero() {
		f.CreateAt = time.Now()
	}
	f.UpdateAt = time.Now()
	type my Form
	return bson.Marshal((*my)(f))
}
