package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type Account struct {
	BaseModel `bson:",inline"`
	UserId    primitive.ObjectID `json:"userId," bson:"userId,omitempty"`
	Username  string             `json:"username" bson:"username"`
	Password  string             `json:"password" bson:"password"`
	Roles     []Role             `json:"roles" bson:"roles"`
}

// HashPassword hashes the account password using bcrypt
func (a *Account) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(a.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.Password = string(hashedPassword)
	return nil
}

// CheckPassword compares the provided password with the hashed password
func (a *Account) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(password))
}

// AccountRequest represents the login request payload
type AccountRequest struct {
	Username string `json:"username" binding:"required" example:"john_doe"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// AccountRegister represents the registration request payload
type AccountRegister struct {
	Username string `json:"username" binding:"required" example:"john_doe"`
	Password string `json:"password" binding:"required" example:"password123"`
	Roles    []Role `json:"roles" example:"[{\"name\":\"user\"}]"`
}

// AccountResponse represents the account data in API responses
type AccountResponse struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	Username string             `json:"username" example:"john_doe"`
	Roles    []Role             `json:"roles"`
}
