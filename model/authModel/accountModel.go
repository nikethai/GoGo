package authModel

import "go.mongodb.org/mongo-driver/bson/primitive"

type Account struct {
	ID       primitive.ObjectID `json:"id," bson:"_id,omitempty"`
	Username string             `json:"username" bson:"username"`
	Password string             `json:"password" bson:"password"`
	Role     []Role             `json:"roles" bson:"roles"`
}

type AccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AccountRegister struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Roles    []Role `json:"roles"`
}

type AccountResponse struct {
	ID       string `json:"id" `
	Uuid     string `json:"uuid"`
	Username string `json:"username"`
	Roles    []Role `json:"roles"`
}
