package model

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BaseModel contains common fields for all models
type BaseModel struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
	DeletedAt *time.Time         `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
}

// GetID implements the Entity interface
func (b *BaseModel) GetID() primitive.ObjectID {
	return b.ID
}

// SetID implements the Entity interface
func (b *BaseModel) SetID(id primitive.ObjectID) {
	b.ID = id
}

// SetTimestamps sets the created and updated timestamps
func (b *BaseModel) SetTimestamps() {
	now := time.Now()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now
}

// SoftDelete sets the deleted timestamp
func (b *BaseModel) SoftDelete() {
	now := time.Now()
	b.DeletedAt = &now
}

// IsDeleted checks if the model is soft deleted
func (b *BaseModel) IsDeleted() bool {
	return b.DeletedAt != nil
}