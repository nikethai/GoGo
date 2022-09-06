package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Project struct {
	ID           primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name         string               `json:"name" bson:"name"`
	Description  string               `json:"description" bson:"description"`
	CreateBy     primitive.ObjectID   `json:"createBy" bson:"createBy"`
	CreateAt     time.Time            `json:"createAt" bson:"createAt"`
	UpdateAt     time.Time            `json:"updateAt" bson:"updateAt"`
	Participants []primitive.ObjectID `json:"participants" bson:"participants"` // list of user id
	Forms        []primitive.ObjectID `json:"forms" bson:"forms"`               // list of form id
}

type ProjectResponse struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Name        string                 `json:"name" bson:"name"`
	Description string                 `json:"description" bson:"description"`
	CreateBy    UserResponseWithoutAcc `json:"createBy" bson:"createBy"`
	CreateAt    time.Time              `json:"createAt" bson:"createAt"`
	UpdateAt    time.Time              `json:"updateAt" bson:"updateAt"`
	// Participants []primitive.ObjectID `json:"participants" bson:"participants"` // list of user id
	// Forms        []primitive.ObjectID `json:"forms" bson:"forms"`               // list of form id
}

func (p *Project) MarshalBSON() ([]byte, error) {
	if p.CreateAt.IsZero() {
		p.CreateAt = time.Now()
	}
	p.UpdateAt = time.Now()
	type my Project
	return bson.Marshal((*my)(p))
}
