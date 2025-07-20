package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Role struct {
	Id   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
}

// GetID implements the Entity interface
func (r *Role) GetID() primitive.ObjectID {
	return r.Id
}

// SetID implements the Entity interface
func (r *Role) SetID(id primitive.ObjectID) {
	r.Id = id
}
