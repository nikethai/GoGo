package model

type Role struct {
	BaseModel `bson:",inline"`
	Name      string `json:"name" bson:"name"`
}
