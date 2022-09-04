package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Role struct {
	Id   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
}
