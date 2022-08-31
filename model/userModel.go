package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `json:"id," bson:"_id,omitempty"`
	Fullname string             `json:"fullname" bson:"fullname"`
	DOB      string             `json:"dob" bson:"dob"`
	Email    string             `json:"email" bson:"email"`
	Phone    string             `json:"phone" bson:"phone"`
	Address  string             `json:"address" bson:"address"`
	Avatar   string             `json:"avatar" bson:"avatar"`
	Status   string             `json:"status" bson:"status"`
}
