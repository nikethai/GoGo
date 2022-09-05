package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Project struct {
	ID          string             `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	CreateBy    primitive.ObjectID `json:"createBy" bson:"createBy"`
	CreateAt    time.Time          `json:"createAt" bson:"createAt"`
	UpdateAt    time.Time          `json:"updateAt" bson:"updateAt"`
	// Participants []Participant      `json:"participants" bson:"participants"`
}
