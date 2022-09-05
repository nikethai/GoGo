package model

type Form struct {
	ID          string `json:"id" bson:"_id,omitempty"`
	Name        string `json:"name" bson:"name"`
	Description string `json:"description" bson:"description"`
	// ProjectID   string             `json:"projectId" bson:"projectId"`
	// CreateBy    primitive.ObjectID `json:"createBy" bson:"createBy"`
	// CreateAt    time.Time          `json:"createAt" bson:"createAt"`
	// UpdateAt    time.Time          `json:"updateAt" bson:"updateAt"`
	// Participants []Participant      `json:"participants" bson:"participants"`
}
