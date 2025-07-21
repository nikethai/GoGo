package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	BaseModel `bson:",inline"`
	AccountId primitive.ObjectID `json:"accountId," bson:"accountId,omitempty"`
	Fullname  string             `json:"fullName" bson:"fullName"`
	DOB       string             `json:"dob" bson:"dob"`
	Email     string             `json:"email" bson:"email"`
	Phone     string             `json:"phone" bson:"phone"`
	Address   string             `json:"address" bson:"address,omitempty"`
	Avatar    string             `json:"avatar" bson:"avatar,omitempty"`
	Status    string             `json:"status" bson:"status"`
}

type UserResponse struct {
	ID primitive.ObjectID `json:"id," bson:"_id,omitempty"`
	// AccountId primitive.ObjectID `json:"accountId," bson:"accountId,omitempty"`
	Fullname string          `json:"fullName" bson:"fullName"`
	DOB      string          `json:"dob" bson:"dob"`
	Email    string          `json:"email" bson:"email"`
	Phone    string          `json:"phone" bson:"phone"`
	Address  string          `json:"address" bson:"address,omitempty"`
	Avatar   string          `json:"avatar" bson:"avatar,omitempty"`
	Status   string          `json:"status" bson:"status"`
	Account  AccountResponse `json:"account" bson:"account"`
}

type UserResponseWithoutAcc struct {
	ID       primitive.ObjectID `json:"id," bson:"_id,omitempty"`
	Fullname string             `json:"fullName" bson:"fullName"`
	DOB      string             `json:"dob" bson:"dob"`
	Email    string             `json:"email" bson:"email"`
	Phone    string             `json:"phone" bson:"phone"`
	Address  string             `json:"address" bson:"address,omitempty"`
	Avatar   string             `json:"avatar" bson:"avatar,omitempty"`
	Status   string             `json:"status" bson:"status"`
}

type UserRequest struct {
	AccountId primitive.ObjectID `json:"accountId"`
	Fullname  string             `json:"fullName"`
	DOB       string             `json:"dob"`
	Email     string             `json:"email"`
	Phone     string             `json:"phone"`
	Address   string             `json:"address"`
	Avatar    string             `json:"avatar"`
	Status    string             `json:"status"`
}
