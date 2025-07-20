package repository

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Entity represents any entity that can be stored in the repository
type Entity interface {
	// GetID returns the entity's ID
	GetID() primitive.ObjectID
	// SetID sets the entity's ID
	SetID(primitive.ObjectID)
}

// ListOptions provides options for listing entities
type ListOptions struct {
	Page   int
	Limit  int
	Sort   map[string]int // field -> order (1 for asc, -1 for desc)
	Filter interface{}
}

// ListResult contains the result of a list operation
type ListResult[T Entity] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"totalPages"`
}

// Repository defines the generic repository interface
type Repository[T Entity] interface {
	// Create inserts a new entity
	Create(ctx context.Context, entity T) (T, error)
	
	// GetByID retrieves an entity by its ID
	GetByID(ctx context.Context, id primitive.ObjectID) (T, error)
	
	// GetByField retrieves an entity by a specific field
	GetByField(ctx context.Context, field string, value interface{}) (T, error)
	
	// Update updates an existing entity
	Update(ctx context.Context, id primitive.ObjectID, updates interface{}) (T, error)
	
	// Delete removes an entity by its ID
	Delete(ctx context.Context, id primitive.ObjectID) error
	
	// List retrieves entities with pagination and filtering
	List(ctx context.Context, opts ListOptions) (ListResult[T], error)
	
	// Count returns the total number of entities matching the filter
	Count(ctx context.Context, filter interface{}) (int64, error)
	
	// Exists checks if an entity exists by ID
	Exists(ctx context.Context, id primitive.ObjectID) (bool, error)
	
	// GetAll retrieves all entities (use with caution)
	GetAll(ctx context.Context, filter interface{}, opts *options.FindOptions) ([]T, error)
	
	// Aggregate performs aggregation operations
	Aggregate(ctx context.Context, pipeline interface{}) ([]bson.M, error)
}

// TransactionalRepository extends Repository with transaction support
type TransactionalRepository[T Entity] interface {
	Repository[T]
	
	// WithTransaction executes operations within a transaction
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}