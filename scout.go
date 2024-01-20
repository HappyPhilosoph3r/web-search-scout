package main

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

var startUrl string = "https://pkg.go.dev/std"

func main() {

	// 1. Connect to database to get a list of urls that require assesment.

	db, err := databaseConnection()

	if err != nil {
		fmt.Printf("Database Connection Error Occurred %v \n", err)
		return
	}

	// 2. Find the docs collection and return the number of documents.

	// Check there is a seed document (First document in the database)

	if countDocuments(db, "docs", bson.D{{}}) == 0 {
		createDoc(startUrl, "", db)
	}

	documentCount := countDocuments(db, "docs", notScoutedFilter)

	fmt.Printf("Number of documents to scout: %v\n", documentCount)

	// Loop that only terminates when all documents are scouted (capped at 1 for development).
	// This will be replaced with a webserver so that new requests can be added easily / prioritised.

	for i := 0; i < 1; i++ {
		documentCount := countDocuments(db, "docs", notScoutedFilter)
		fmt.Printf("Number of documents to scout: %v\n", documentCount)

		if documentCount == 0 {
			return
		}

		// 3. For each url, look at it and collect any urls within the page.

		results := DocsToScout(db)

		for _, result := range results {
			scoutDoc(result, db)
		}
	}
}
