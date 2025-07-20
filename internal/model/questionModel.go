package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Question struct {
	Id          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Uuid        string             `json:"uuid" bson:"uuid"`
	Content     string             `json:"content"`
	Description string             `json:"description" bson:"description,omitempty"`
	Type        string             `json:"type" bson:"type"`
	CreateBy    primitive.ObjectID `json:"createBy" bson:"createBy"` // user id
	CreateAt    time.Time          `json:"createAt" bson:"createAt"`
	UpdateAt    time.Time          `json:"updateAt" bson:"updateAt"`
	Trait       primitive.M        `json:"trait" bson:",inline"`
}

// GetID implements the Entity interface
func (q *Question) GetID() primitive.ObjectID {
	return q.Id
}

// SetID implements the Entity interface
func (q *Question) SetID(id primitive.ObjectID) {
	q.Id = id
}

func (q *Question) MarshalBSON() ([]byte, error) {
	if q.CreateAt.IsZero() {
		q.CreateAt = time.Now()
	}
	q.UpdateAt = time.Now()
	type my Question
	return bson.Marshal((*my)(q))
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
