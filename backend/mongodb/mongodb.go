package mongodb

import (
	"context"
	"log"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var once sync.Once
var mongoClient *mongo.Client

// GetMongoDBClient get the client
func GetMongoDBClient() (*mongo.Client, error) {
	once.Do(func() {
		// Set client options
		clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

		// Connect to MongoDB
		client, err := mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			log.Fatal(err)
			return
		}

		mongoClient = client
	})

	// Check the connection
	err := mongoClient.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
		return mongoClient, err
	}

	return mongoClient, nil
}
