package main

import (
	"context"
	"fmt"
	"log"

	"main/db"
	"main/internal/repository"
	"main/internal/repository/mongo"
	"main/internal/service"
	"main/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	RepositoryUsageExample()
}

// RepositoryUsageExample demonstrates how to use the generic MongoDB repository
func RepositoryUsageExample() {
	// Initialize MongoDB connection
	db.InitConnection()
	client := db.MongoClient
	database := client.Database("gogo")

	ctx := context.Background()

	// Create repositories for different models
	userRepo := mongo.NewMongoRepository[*model.User](database, "users")
	accountRepo := mongo.NewMongoRepository[*model.Account](database, "accounts")
	projectRepo := mongo.NewMongoRepository[*model.Project](database, "projects")
	roleRepo := mongo.NewMongoRepository[*model.Role](database, "roles")
	
	// Additional repositories that will be used later
	formRepo := mongo.NewMongoRepository[*model.Form](database, "forms")
	questionRepo := mongo.NewMongoRepository[*model.Question](database, "questions")

	// Example 1: Basic CRUD operations with User
	fmt.Println("=== Example 1: Basic CRUD Operations ===")
	
	// Create a new user
	user := &model.User{
		ID:        primitive.NewObjectID(),
		AccountId: primitive.NewObjectID(),
		Fullname:  "John Doe",
		Email:     "john.doe@example.com",
		Status:    "active",
	}

	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return
	}
	fmt.Printf("Created user: %+v\n", createdUser)

	// Get user by ID
	fetchedUser, err := userRepo.GetByID(ctx, createdUser.GetID())
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		return
	}
	fmt.Printf("Fetched user: %+v\n", fetchedUser)

	// Update user
	updates := bson.M{"fullname": "John Smith"}
	updatedUser, err := userRepo.Update(ctx, createdUser.GetID(), updates)
	if err != nil {
		log.Printf("Error updating user: %v", err)
		return
	}
	fmt.Printf("Updated user: %+v\n", updatedUser)

	// Example 2: List operations with pagination
	fmt.Println("\n=== Example 2: List Operations with Pagination ===")
	
	// Create multiple users for demonstration
	for i := 0; i < 5; i++ {
		testUser := &model.User{
			ID:        primitive.NewObjectID(),
			AccountId: primitive.NewObjectID(),
			Fullname:  fmt.Sprintf("Test User %d", i+1),
			Email:     fmt.Sprintf("test%d@example.com", i+1),
			Status:    "active",
		}
		_, _ = userRepo.Create(ctx, testUser)
	}

	// List users with pagination
	listResult, err := userRepo.List(ctx, repository.ListOptions{
		Page:   1,
		Limit:  3,
		Sort:   map[string]int{"fullname": 1},
		Filter: bson.M{},
	})
	if err != nil {
		log.Printf("Error listing users: %v", err)
		return
	}
	fmt.Printf("Listed users (page 1): %d items, total: %d\n", len(listResult.Data), listResult.Total)
	for _, user := range listResult.Data {
		fmt.Printf("  - %s (%s)\n", user.Fullname, user.Email)
	}

	// Example 3: Search and filtering
	fmt.Println("\n=== Example 3: Search and Filtering ===")
	
	// Get users by field
	activeUsers, err := userRepo.GetAll(ctx, bson.M{"status": "active"}, nil)
	if err != nil {
		log.Printf("Error getting active users: %v", err)
		return
	}
	fmt.Printf("Found %d active users\n", len(activeUsers))

	// Count documents
	count, err := userRepo.Count(ctx, bson.M{"status": "active"})
	if err != nil {
		log.Printf("Error counting users: %v", err)
		return
	}
	fmt.Printf("Total active users count: %d\n", count)

	// Check existence
	exists, err := userRepo.Exists(ctx, createdUser.GetID())
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		return
	}
	fmt.Printf("User exists: %t\n", exists)

	// Example 4: Aggregation operations
	fmt.Println("\n=== Example 4: Aggregation Operations ===")
	
	// Aggregate users by status
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
			"users": bson.M{"$push": "$fullname"},
		}},
		{"$sort": bson.M{"count": -1}},
	}

	aggResults, err := userRepo.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Error in aggregation: %v", err)
		return
	}
	fmt.Printf("Aggregation results: %+v\n", aggResults)

	// Example 5: Using the service layer
	fmt.Println("\n=== Example 5: Service Layer Usage ===")
	
	userService := service.NewUserService(userRepo, accountRepo)

	// Create user with account
	userReq := &model.UserRequest{
		Fullname: "Jane Doe",
		Email:    "jane.doe@example.com",
	}
	accountReq := &model.AccountRequest{
		Username: "janedoe",
		Password: "securepassword123",
	}

	serviceUser, err := userService.CreateUser(ctx, userReq, accountReq)
	if err != nil {
		log.Printf("Error creating user via service: %v", err)
		return
	}
	fmt.Printf("Created user via service: %+v\n", serviceUser)

	// Get user with account information
	userWithAccount, err := userService.GetUserWithAccount(ctx, serviceUser.GetID())
	if err != nil {
		log.Printf("Error getting user with account: %v", err)
		return
	}
	fmt.Printf("User with account: %+v\n", userWithAccount)

	// Example 6: Working with other models
	fmt.Println("\n=== Example 6: Working with Other Models ===")
	
	// Create a project
	project := &model.Project{
		ID:          primitive.NewObjectID(),
		Name:        "Sample Project",
		Description: "A sample project for testing",
		CreateBy:    createdUser.GetID(),
		Participants: []primitive.ObjectID{createdUser.GetID()},
		Forms:       []primitive.ObjectID{},
	}

	createdProject, err := projectRepo.Create(ctx, project)
	if err != nil {
		log.Printf("Error creating project: %v", err)
		return
	}
	fmt.Printf("Created project: %+v\n", createdProject)

	// Create a role
	role := &model.Role{
		Id:   primitive.NewObjectID(),
		Name: "Admin",
	}

	createdRole, err := roleRepo.Create(ctx, role)
	if err != nil {
		log.Printf("Error creating role: %v", err)
		return
	}
	fmt.Printf("Created role: %+v\n", createdRole)
	
	// Create a form using formRepo
	form := &model.Form{
		ID:          primitive.NewObjectID(),
		Name:        "Sample Form",
		Description: "A sample form for testing",
		Questions:   []primitive.ObjectID{},
	}
	_, err = formRepo.Create(ctx, form)
	if err != nil {
		log.Printf("Error creating form: %v", err)
	}
	
	// Create a question using questionRepo
	question := &model.Question{
		Id:          primitive.NewObjectID(),
		Uuid:        "sample-uuid",
		Content:     "Sample Question",
		Description: "A sample question",
		Type:        "text",
		CreateBy:    createdUser.GetID(),
	}
	_, err = questionRepo.Create(ctx, question)
	if err != nil {
		log.Printf("Error creating question: %v", err)
	}

	// Example 7: Transaction support (if using TransactionalRepository)
	fmt.Println("\n=== Example 7: Transaction Support ===")
	
	txUserRepo := mongo.NewTransactionalMongoRepository[*model.User](database, "users")
	txAccountRepo := mongo.NewTransactionalMongoRepository[*model.Account](database, "accounts")

	// Perform operations within a transaction
	err = txUserRepo.WithTransaction(ctx, func(ctx context.Context) error {
		// Create account
		txAccount := &model.Account{
			ID:       primitive.NewObjectID(),
			Username: "txuser",
			Password: "txpassword",
			Roles:    []model.Role{*createdRole},
		}
		_, err := txAccountRepo.Create(ctx, txAccount)
		if err != nil {
			return err
		}

		// Create user
		txUser := &model.User{
			ID:        primitive.NewObjectID(),
			AccountId: txAccount.GetID(),
			Fullname:  "Transaction User",
			Email:     "tx@example.com",
			Status:    "active",
		}
		_, err = txUserRepo.Create(ctx, txUser)
		return err
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
	} else {
		fmt.Println("Transaction completed successfully")
	}

	// Cleanup - Delete created user
	err = userRepo.Delete(ctx, createdUser.GetID())
	if err != nil {
		log.Printf("Error deleting user: %v", err)
	}

	fmt.Println("\n=== Repository Usage Example Completed ===")
}

// InitializeRepositories shows how to set up repositories in your application
func InitializeRepositories() map[string]interface{} {
	// Initialize MongoDB connection
	db.InitConnection()
	client := db.MongoClient
	database := client.Database("gogo")

	// Create all repositories
	repos := map[string]interface{}{
		"user":     mongo.NewMongoRepository[*model.User](database, "users"),
		"account":  mongo.NewMongoRepository[*model.Account](database, "accounts"),
		"project":  mongo.NewMongoRepository[*model.Project](database, "projects"),
		"form":     mongo.NewMongoRepository[*model.Form](database, "forms"),
		"question": mongo.NewMongoRepository[*model.Question](database, "questions"),
		"role":     mongo.NewMongoRepository[*model.Role](database, "roles"),
	}

	return repos
}