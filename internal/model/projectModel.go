package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Project struct {
	BaseModel    `bson:",inline"`
	Name         string               `json:"name" bson:"name"`
	Description  string               `json:"description" bson:"description"`
	CreateBy     primitive.ObjectID   `json:"createBy" bson:"createBy"`
	Participants []primitive.ObjectID `json:"participants" bson:"participants"` // list of user id
	Forms        []primitive.ObjectID `json:"forms" bson:"forms"`               // list of form id
}

type ProjectResponse struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	Name        string                 `json:"name" bson:"name"`
	Description string                 `json:"description" bson:"description"`
	CreateBy    UserResponseWithoutAcc `json:"createBy" bson:"createBy"`
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" bson:"updatedAt"`
	// Participants []primitive.ObjectID `json:"participants" bson:"participants"` // list of user id
	// Forms        []primitive.ObjectID `json:"forms" bson:"forms"`               // list of form id
}
