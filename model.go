package main

import "go.mongodb.org/mongo-driver/bson/primitive"

type News struct {
	Date    string `json:"date"`
	Id      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content" bson:"content,omitempty"`
}

type Question struct {
	Id          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Content     string             `json:"content"`
	Description string             `json:"description" bson:"description,omitempty"`
	Type        string             `json:"type" bson:"type"`
	Trait       primitive.M        `json:"trait" bson:",inline"`
}
