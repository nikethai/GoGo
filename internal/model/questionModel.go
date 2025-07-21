package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Question struct {
	BaseModel   `bson:",inline"`
	Uuid        string             `json:"uuid" bson:"uuid"`
	Content     string             `json:"content"`
	Description string             `json:"description" bson:"description,omitempty"`
	Type        string             `json:"type" bson:"type"`
	CreateBy    primitive.ObjectID `json:"createBy" bson:"createBy"` // user id
	Trait       primitive.M        `json:"trait" bson:",inline"`
}

/*Example for Trait*/
// 	Trait: bson.M{
// 		"options": []string{"John", "Jane", "Joe"}, // Copilot is so gud
// 	},
// }
// 	Trait: bson.M{
// 		"col": []string{"Ảnh hưởng tới gia đình", "Ảnh hưởng tới sức khoẻ"},
// 		"row": []string{"Nhiều", "Trung bình", "Ít"},
// 	},
