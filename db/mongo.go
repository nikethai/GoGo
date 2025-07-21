package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"main/internal/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	MongoClient   *mongo.Client
	MongoDatabase *mongo.Database
	UserCollection    = config.UserCollection
	AccountCollection = config.AccountCollection
	RoleCollection    = config.RoleCollection
	FormCollection    = config.FormCollection
	ProjectCollection = config.ProjectCollection
	QuestionCollection = config.QuestionCollection
)

func InitConnection() {
	MongoClient = GetMongoEnv()
	MongoDatabase = MongoClient.Database("surveyDB")
	
	// Initialize default roles if they don't exist
	initializeDefaultRoles()
}

func GetMongoEnv() *mongo.Client {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found")
	}

	uri := os.Getenv("MONGODB_URI")

	if uri == "" {
		log.Fatal("MONGODB_URI is not set")
	}

	// The hell is TODO? Why TODO? WTF GO????
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalln("Cannot connect to mongodb")
		log.Fatal(err)
	}

	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		// Can't connect to Mongo
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	return client
}

// initializeDefaultRoles creates default roles if they don't exist
func initializeDefaultRoles() {
	roleCollection := MongoDatabase.Collection(RoleCollection)
	
	// Check if the default user role exists
	var userRole struct {
		Name string `bson:"name"`
	}
	
	err := roleCollection.FindOne(context.TODO(), bson.M{"name": "user"}).Decode(&userRole)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create the default user role
			defaultRole := struct {
				Name      string    `bson:"name"`
				CreatedAt time.Time `bson:"createdAt"`
				UpdatedAt time.Time `bson:"updatedAt"`
			}{
				Name:      "user",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			
			_, err := roleCollection.InsertOne(context.TODO(), defaultRole)
			if err != nil {
				log.Printf("Failed to create default user role: %v", err)
			} else {
				log.Println("Created default user role")
			}
		} else {
			log.Printf("Error checking for default user role: %v", err)
		}
	}
	
	// Check if admin role exists
	var adminRole struct {
		Name string `bson:"name"`
	}
	
	err = roleCollection.FindOne(context.TODO(), bson.M{"name": "admin"}).Decode(&adminRole)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create the admin role
			adminRoleDoc := struct {
				Name      string    `bson:"name"`
				CreatedAt time.Time `bson:"createdAt"`
				UpdatedAt time.Time `bson:"updatedAt"`
			}{
				Name:      "admin",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			
			_, err := roleCollection.InsertOne(context.TODO(), adminRoleDoc)
			if err != nil {
				log.Printf("Failed to create admin role: %v", err)
			} else {
				log.Println("Created admin role")
			}
		}
	}
}
