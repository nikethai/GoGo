package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	MongoClient   *mongo.Client
	MongoDatabase *mongo.Database
)

func InitConnection() {
	MongoClient = GetMongoEnv()
	MongoDatabase = MongoClient.Database("surveyDB")
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
