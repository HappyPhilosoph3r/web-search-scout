package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Doc struct {
	ID      *primitive.ObjectID `bson:"_id,omitempty"`
	Name    string
	Url     string
	Scouted bool
	Alive   bool
	Allowed bool

	URLCount    int
	URLParent   string
	ContentType string
	StatusCode  int
	Data        []byte
	Domain      *primitive.ObjectID `bson:"domain"`
}

var notScoutedFilter = bson.D{{Key: "scouted", Value: false}}

func DocsToScout(db *mongo.Client) []Doc {
	// DocsToScout finds all documents in the database that have not been scouted (examined for urls).
	// Returns an array of type Doc.

	collection := db.Database("web-search").Collection("docs")

	filter := bson.D{{Key: "scouted", Value: false}}

	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		panic(err)
	}
	var docs []Doc
	if err = cursor.All(context.TODO(), &docs); err != nil {
		panic(err)
	}

	return docs
}

func formURL(prefix, s string) string {
	// formURL checks whether a url is complete or requires a prefix to be able to navigate to the page.
	//  Returns a full url as a string.

	re := regexp.MustCompile(`^http`)

	if len(re.FindAllString(s, -1)) > 0 {
		return strings.Split(s, "\"")[1]
	}
	return fmt.Sprintf("%v%v", prefix, strings.Split(s, "\"")[1])
}

func addNewDoc(db *mongo.Client, doc Doc) bool {
	// addNewDoc checks if a document already exists. If it does not this function adds a document to the database.
	// Returns a boolean value, true means success occurred.

	if countDocuments(db, "docs", bson.D{{Key: "url", Value: doc.Url}}) == 0 {
		collection := db.Database("web-search").Collection("docs")
		result, err := collection.InsertOne(context.TODO(), doc)
		if err != nil {
			return false
		}
		fmt.Printf("Inserted document with _id: %v\n", result.InsertedID)
		return true
	}
	fmt.Printf("Document with url %v already exists \n", doc.Url)
	return false

}

func createDoc(url string, parent string, db *mongo.Client) Doc {
	// createDoc defines the relevant elements of a document and attempts to add the doument to the database.
	// Returns doc.

	var doc Doc

	doc.Url = url
	doc.URLParent = parent
	doc.Scouted = false

	name, _ := extractDomain(doc.Url)

	filter := bson.D{{Key: "name", Value: name}}

	if countDocuments(db, "domains", filter) == 0 {
		createDomain(db, doc.Url)
	}

	id, err := findDomain(db, name)

	if err != nil {
		fmt.Printf("Error occcured finding domain: %v\n", err)
		panic(nil)
	}

	doc.Domain = id

	addNewDoc(db, doc)

	return doc
}

func updateDoc(db *mongo.Client, doc Doc) {
	// updateDoc takes a dociment and updates the relevant fields in the database.

	filter := bson.D{{Key: "_id", Value: doc.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "scouted", Value: doc.Scouted},
		{Key: "alive", Value: doc.Alive},
		{Key: "contentype", Value: doc.ContentType},
		{Key: "statuscode", Value: doc.StatusCode},
		{Key: "urlcount", Value: doc.URLCount},
		{Key: "data", Value: doc.Data},
	}}}

	collection := db.Database("web-search").Collection("docs")
	result, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		fmt.Printf("Error occurred modifying document: %v, Error: %v\n", doc.ID, err)
	}
	fmt.Printf("Documents matched: %v, Documents updated: %v\n", result.MatchedCount, result.ModifiedCount)
}

func scoutDoc(doc Doc, db *mongo.Client) {
	// scoutDoc takes a url and creates a full profile for the database.
	doc.Scouted = true
	doc.Alive = false
	doc.Allowed = false

	fmt.Printf("Document name = %v\n", doc.Name)

	if !authoriseScoutExpedition(db, doc.Domain, doc.Url) {
		updateDoc(db, doc)
		return
	}

	doc.Allowed = true

	resp, err := http.Get(doc.Url)

	if err != nil {
		fmt.Printf("Error Occurred %v \n", err)
		updateDoc(db, doc)
		return
	}

	doc.ContentType = resp.Header.Get("Content-Type")
	doc.StatusCode = resp.StatusCode

	if doc.StatusCode > 299 {
		updateDoc(db, doc)
		fmt.Printf("status code err: %v\n", doc.StatusCode)
		return
	}

	doc.Alive = true

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error Occurred %v\n", err)
		updateDoc(db, doc)
		return
	}

	doc.Data = body

	if strings.Contains(resp.Header.Get("Content-Type"), "html") {

		dataString := string(doc.Data)

		// search text for urls
		re := regexp.MustCompile(`href="(\S+)"`)
		matches := re.FindAllString(dataString, -1)
		doc.URLCount = len(matches)

		// For each match add to database
		for _, match := range matches {
			var newUrl string = formURL(doc.Url, match)
			var newDoc = createDoc(newUrl, doc.Url, db)
			fmt.Sprintln(newUrl, newDoc)
		}
	}
	updateDoc(db, doc)
}
