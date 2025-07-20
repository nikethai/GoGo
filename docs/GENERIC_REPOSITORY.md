# Generic MongoDB Repository Implementation

This document provides a comprehensive guide to the generic MongoDB repository pattern implemented for the Gogo project, upgraded to Go 1.24 with full generic type support.

## Overview

The generic repository pattern provides a unified interface for data access operations across all MongoDB collections, eliminating code duplication and ensuring consistency in data operations.

## Architecture

### Core Components

1. **Entity Interface** (`internal/repository/repository.go`)
   - Defines the contract that all models must implement
   - Provides `GetID()` and `SetID()` methods for consistent ID handling

2. **Repository Interface** (`internal/repository/repository.go`)
   - Generic interface for standard CRUD operations
   - Type-safe operations using Go 1.24 generics
   - Support for filtering, pagination, and aggregation

3. **MongoDB Implementation** (`internal/repository/mongo/mongo_repository.go`)
   - Concrete implementation of the repository interface
   - MongoDB-specific optimizations and error handling
   - Transaction support for complex operations

## Key Features

### 1. Type Safety
```go
// Type-safe repository creation
userRepo := mongo.NewMongoRepository[*model.User](database, "users")
projectRepo := mongo.NewMongoRepository[*model.Project](database, "projects")

// Compile-time type checking
user, err := userRepo.GetByID(ctx, userID) // Returns *model.User
project, err := projectRepo.GetByID(ctx, projectID) // Returns *model.Project
```

### 2. Consistent Interface
All repositories provide the same set of operations:
- `Create(ctx, entity)` - Create new document
- `GetByID(ctx, id)` - Retrieve by ObjectID
- `GetByField(ctx, field, value)` - Retrieve by any field
- `Update(ctx, id, updates)` - Update document
- `Delete(ctx, id)` - Delete document
- `List(ctx, filter, options)` - Paginated listing
- `GetAll(ctx, filter, options)` - Get all matching documents
- `Count(ctx, filter)` - Count documents
- `Exists(ctx, id)` - Check existence
- `Aggregate(ctx, pipeline)` - MongoDB aggregation

### 3. Advanced Features
- **Pagination Support**: Built-in pagination with `ListOptions`
- **Flexible Filtering**: Support for complex MongoDB queries
- **Aggregation Pipeline**: Full MongoDB aggregation support
- **Transaction Support**: Optional transactional operations
- **Error Handling**: Consistent error handling across all operations

## Implementation Guide

### Step 1: Model Preparation

All models must implement the `Entity` interface:

```go
type User struct {
    ID        primitive.ObjectID `json:"id," bson:"_id,omitempty"`
    AccountId primitive.ObjectID `json:"accountId," bson:"accountId,omitempty"`
    Fullname  string             `json:"fullname" bson:"fullname"`
    Email     string             `json:"email" bson:"email"`
    Status    string             `json:"status" bson:"status"`
}

// GetID implements the Entity interface
func (u *User) GetID() primitive.ObjectID {
    return u.ID
}

// SetID implements the Entity interface
func (u *User) SetID(id primitive.ObjectID) {
    u.ID = id
}
```

### Step 2: Repository Creation

```go
// Initialize MongoDB connection
db.InitConnection()
client := db.GetClient()
database := client.Database("gogo")

// Create type-safe repositories
userRepo := mongo.NewMongoRepository[*model.User](database, "users")
accountRepo := mongo.NewMongoRepository[*model.Account](database, "accounts")
projectRepo := mongo.NewMongoRepository[*model.Project](database, "projects")
```

### Step 3: Basic Operations

```go
ctx := context.Background()

// Create
user := &model.User{
    ID:       primitive.NewObjectID(),
    Fullname: "John Doe",
    Email:    "john@example.com",
    Status:   "active",
}
createdUser, err := userRepo.Create(ctx, user)

// Read
user, err := userRepo.GetByID(ctx, userID)
user, err := userRepo.GetByField(ctx, "email", "john@example.com")

// Update
updates := bson.M{"fullname": "John Smith"}
updatedUser, err := userRepo.Update(ctx, userID, updates)

// Delete
err := userRepo.Delete(ctx, userID)
```

### Step 4: Advanced Operations

```go
// Pagination
listResult, err := userRepo.List(ctx, bson.M{"status": "active"}, &repository.ListOptions{
    Page:  1,
    Limit: 10,
    Sort:  bson.M{"fullname": 1},
})

// Aggregation
pipeline := []bson.M{
    {"$match": bson.M{"status": "active"}},
    {"$group": bson.M{
        "_id":   "$status",
        "count": bson.M{"$sum": 1},
    }},
}
results, err := userRepo.Aggregate(ctx, pipeline)

// Count and Existence
count, err := userRepo.Count(ctx, bson.M{"status": "active"})
exists, err := userRepo.Exists(ctx, userID)
```

### Step 5: Service Layer Integration

```go
type UserService struct {
    userRepo    repository.Repository[*model.User]
    accountRepo repository.Repository[*model.Account]
}

func NewUserService(userRepo repository.Repository[*model.User], accountRepo repository.Repository[*model.Account]) *UserService {
    return &UserService{
        userRepo:    userRepo,
        accountRepo: accountRepo,
    }
}

func (s *UserService) CreateUser(ctx context.Context, userReq *model.UserRequest, accountReq *model.AccountRequest) (*model.User, error) {
    // Business logic using repositories
    // ...
}
```

## Migration from Existing Code

### Before (Old Pattern)
```go
// Old service with direct MongoDB operations
type UserService struct {
    userCollection    *mongo.Collection
    accountCollection *mongo.Collection
}

func (s *UserService) GetUserByID(userID primitive.ObjectID) (*model.User, error) {
    var user model.User
    err := s.userCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

### After (Generic Repository Pattern)
```go
// New service with generic repository
type UserService struct {
    userRepo repository.Repository[*model.User]
}

func (s *UserService) GetUserByID(ctx context.Context, userID primitive.ObjectID) (*model.User, error) {
    return s.userRepo.GetByID(ctx, userID)
}
```

### Migration Steps

1. **Update Go Version**: Upgrade `go.mod` to Go 1.24
2. **Implement Entity Interface**: Add `GetID()` and `SetID()` methods to all models
3. **Create Repositories**: Replace direct collection access with repository instances
4. **Update Services**: Inject repositories instead of collections
5. **Refactor Operations**: Replace manual MongoDB operations with repository methods
6. **Add Tests**: Create comprehensive tests for the new repository pattern

## Transaction Support

For operations requiring transactions, use the `TransactionalRepository`:

```go
txUserRepo := mongo.NewTransactionalMongoRepository[*model.User](database, "users")
txAccountRepo := mongo.NewTransactionalMongoRepository[*model.Account](database, "accounts")

err := txUserRepo.WithTransaction(ctx, func(ctx context.Context) error {
    // Create account
    account := &model.Account{...}
    _, err := txAccountRepo.Create(ctx, account)
    if err != nil {
        return err
    }
    
    // Create user
    user := &model.User{...}
    _, err = txUserRepo.Create(ctx, user)
    return err
})
```

## Best Practices

### 1. Repository Initialization
- Initialize repositories once at application startup
- Use dependency injection to provide repositories to services
- Consider using a repository factory for complex setups

### 2. Error Handling
- Always handle repository errors appropriately
- Use context for timeout and cancellation
- Log errors with sufficient context

### 3. Performance Considerations
- Use appropriate indexes for frequently queried fields
- Implement pagination for large result sets
- Use aggregation pipelines for complex queries
- Consider caching for frequently accessed data

### 4. Testing
- Mock repositories for unit testing services
- Use integration tests for repository implementations
- Test error scenarios and edge cases

## Examples

See `examples/repository_usage.go` for comprehensive usage examples including:
- Basic CRUD operations
- Pagination and filtering
- Aggregation queries
- Service layer integration
- Transaction handling

## Benefits

1. **Code Reusability**: Common operations implemented once
2. **Type Safety**: Compile-time type checking with generics
3. **Consistency**: Uniform interface across all data access
4. **Maintainability**: Centralized data access logic
5. **Testability**: Easy to mock and test
6. **Performance**: Optimized MongoDB operations
7. **Scalability**: Easy to extend with new operations

## Conclusion

The generic repository pattern provides a robust, type-safe, and maintainable approach to data access in the Gogo project. By leveraging Go 1.24's generics, we achieve both performance and developer experience improvements while maintaining clean architecture principles.