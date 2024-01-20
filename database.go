package main

// This file contains all relevant MongoDB conections

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const uri = "mongodb://web_search_db"

func databaseConnection() (*mongo.Client, error) {
	// Use the SetServerAPIOptions() method to set the Stable API version to 1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)
	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, err
	}
	// Send a ping to confirm a successful connection
	var result bson.M
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result); err != nil {
		return nil, err
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
	return client, err
}

func countDocuments(db *mongo.Client, collectionName string, filter primitive.D) int64 {

	collection := db.Database("web-search").Collection(collectionName)

	count, err := collection.CountDocuments(context.TODO(), filter)

	if err != nil {
		fmt.Printf("Error Occurred counting documents %v \n", err)
		return 0
	}
	return count
}
