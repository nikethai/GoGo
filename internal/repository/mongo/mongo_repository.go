package mongo

import (
	"context"
	"errors"
	"main/internal/repository"
	"math"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRepository implements the generic repository pattern for MongoDB
type MongoRepository[T repository.Entity] struct {
	collection *mongo.Collection
	client     *mongo.Client
}

// NewMongoRepository creates a new MongoDB repository instance
func NewMongoRepository[T repository.Entity](database *mongo.Database, collection string) repository.Repository[T] {
	return &MongoRepository[T]{
		collection: database.Collection(collection),
		client:     database.Client(),
	}
}

// NewTransactionalMongoRepository creates a new MongoDB repository with transaction support
func NewTransactionalMongoRepository[T repository.Entity](database *mongo.Database, collection string) repository.TransactionalRepository[T] {
	return &MongoRepository[T]{
		collection: database.Collection(collection),
		client:     database.Client(),
	}
}

// Create inserts a new entity
func (r *MongoRepository[T]) Create(ctx context.Context, entity T) (T, error) {
	// Generate new ID if not set
	if entity.GetID().IsZero() {
		entity.SetID(primitive.NewObjectID())
	}

	result, err := r.collection.InsertOne(ctx, entity)
	if err != nil {
		return entity, err
	}

	// Set the inserted ID
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		entity.SetID(oid)
	}

	return entity, nil
}

// GetByID retrieves an entity by its ID
func (r *MongoRepository[T]) GetByID(ctx context.Context, id primitive.ObjectID) (T, error) {
	var entity T
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&entity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return entity, errors.New("entity not found")
		}
		return entity, err
	}
	return entity, nil
}

// GetByField retrieves an entity by a specific field
func (r *MongoRepository[T]) GetByField(ctx context.Context, field string, value interface{}) (T, error) {
	var entity T
	filter := bson.M{field: value}
	err := r.collection.FindOne(ctx, filter).Decode(&entity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return entity, errors.New("entity not found")
		}
		return entity, err
	}
	return entity, nil
}

// Update updates an existing entity
func (r *MongoRepository[T]) Update(ctx context.Context, id primitive.ObjectID, updates interface{}) (T, error) {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": updates}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedEntity T
	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedEntity)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			var zero T
			return zero, errors.New("entity not found")
		}
		var zero T
		return zero, err
	}
	return updatedEntity, nil
}

// Delete removes an entity by its ID
func (r *MongoRepository[T]) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("entity not found")
	}
	return nil
}

// List retrieves entities with pagination and filtering
func (r *MongoRepository[T]) List(ctx context.Context, opts repository.ListOptions) (repository.ListResult[T], error) {
	var result repository.ListResult[T]

	// Set defaults
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	// Build filter
	filter := bson.M{}
	if opts.Filter != nil {
		filter = opts.Filter.(bson.M)
	}

	// Count total documents
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return result, err
	}

	// Build find options
	findOpts := options.Find()
	findOpts.SetSkip(int64((opts.Page - 1) * opts.Limit))
	findOpts.SetLimit(int64(opts.Limit))

	// Add sorting
	if len(opts.Sort) > 0 {
		findOpts.SetSort(opts.Sort)
	}

	// Execute query
	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return result, err
	}
	defer cursor.Close(ctx)

	// Decode results
	var entities []T
	if err = cursor.All(ctx, &entities); err != nil {
		return result, err
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(opts.Limit)))

	result = repository.ListResult[T]{
		Data:       entities,
		Total:      total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}

	return result, nil
}

// Count returns the total number of entities matching the filter
func (r *MongoRepository[T]) Count(ctx context.Context, filter interface{}) (int64, error) {
	if filter == nil {
		filter = bson.M{}
	}
	return r.collection.CountDocuments(ctx, filter)
}

// Exists checks if an entity exists by ID
func (r *MongoRepository[T]) Exists(ctx context.Context, id primitive.ObjectID) (bool, error) {
	filter := bson.M{"_id": id}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAll retrieves all entities (use with caution)
func (r *MongoRepository[T]) GetAll(ctx context.Context, filter interface{}, opts *options.FindOptions) ([]T, error) {
	var entities []T
	if filter == nil {
		filter = bson.M{}
	}
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &entities); err != nil {
		return nil, err
	}
	return entities, nil
}

// Aggregate performs aggregation operations
func (r *MongoRepository[T]) Aggregate(ctx context.Context, pipeline interface{}) ([]bson.M, error) {
	var results []bson.M
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// WithTransaction executes operations within a transaction
func (r *MongoRepository[T]) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := r.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		_, err := session.WithTransaction(sc, func(sc mongo.SessionContext) (interface{}, error) {
			return nil, fn(sc)
		})
		return err
	})
}