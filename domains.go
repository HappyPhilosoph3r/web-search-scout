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

type Domain struct {
	ID           *primitive.ObjectID `bson:"_id,omitempty"`
	Name         string
	Address      string
	RobotsFile   bool
	Restrictions bool
	Allowed      []string
	Disallowed   []string
}

func extractDomain(url string) (string, string) {
	// extractDomain takes a url string and splits it to get the domain name.
	// Return domain name amd address respectively
	split := strings.Split(url, "/")

	return split[2], split[0] + "//" + split[2]
}

func findDomain(db *mongo.Client, name string) (*primitive.ObjectID, error) {

	collection := db.Database("web-search").Collection("domains")
	filter := bson.D{{Key: "name", Value: name}}

	var domain Domain

	err := collection.FindOne(context.TODO(), filter).Decode(&domain)

	return domain.ID, err

}

func readRobotTxt(text []byte) (bool, []string, []string) {
	// readRobotTxt finds the relevant data for processing the domain.
	// Returns restrictions, allowed paths and disallowed paths

	var restrictions bool = false
	var allowed []string
	var disallowed []string

	textString := string(text)

	// Find if relevant agent exists
	re := regexp.MustCompile(`User-agent: \*`)
	relevant_agent := re.FindAllStringIndex(textString, -1)

	if len(relevant_agent) == 0 {
		return restrictions, allowed, disallowed
	}

	// Find all agents
	reAgents := regexp.MustCompile(`User-agent`)
	agents := reAgents.FindAllStringIndex(textString, -1)

	for i, agent := range agents {
		next := len(textString)
		if i != len(agents)-1 {
			next = agents[i+1][0]
		}
		// Focus on specific agent
		sample := textString[agent[0]:next]
		if len(re.FindAllStringIndex(sample, -1)) == 0 {
			continue
		}

		// For relevant agent find allowed and disallowed paths
		elements := strings.Split(sample, "\n")
		for _, e := range elements {
			split := strings.Split(e, ": ")
			if split[0] == "Allow" && split[1] == "/" {
				return restrictions, allowed, disallowed
			}
			if split[0] == "Allow" {
				allowed = append(allowed, strings.Trim(split[1], " "))
			}
			if split[0] == "Disallow" {
				disallowed = append(disallowed, strings.Trim(split[1], " "))
			}
		}
	}

	restrictions = true

	return restrictions, allowed, disallowed
}

func addNewDomain(db *mongo.Client, domain Domain) bool {
	// addNewDoc checks if a document already exists. If it does not this function adds a document to the database.
	// Returns a boolean value, true means success occurred.

	if countDocuments(db, "domains", bson.D{{Key: "name", Value: domain.Name}}) == 0 {
		collection := db.Database("web-search").Collection("domains")
		result, err := collection.InsertOne(context.TODO(), domain)
		if err != nil {
			fmt.Printf("Error occurred creating domain %v \n", err)
			return false
		}
		fmt.Printf("Inserted domain with _id: %v\n", result.InsertedID)
		return true
	}
	fmt.Printf("Domain with name %v already exists\n", domain.Name)
	return false

}

func createDomain(db *mongo.Client, url string) {

	var domain Domain

	name, address := extractDomain(url)

	domain.Name = name
	domain.Address = address

	robotsTxt := address + "/robots.txt"

	resp, err := http.Get(robotsTxt)
	if err != nil {
		fmt.Printf("Error Occurred retrieving robot.txt file%v\n", err)
	}

	if resp.StatusCode == 200 {
		domain.RobotsFile = true
		// Analyse robots.txt
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error Occurred reading robot.txt file %v\n", err)
		}

		restrictions, allowed, disallowed := readRobotTxt(body)

		domain.Restrictions = restrictions
		domain.Allowed = allowed
		domain.Disallowed = disallowed

	} else {
		domain.RobotsFile = false
		domain.Restrictions = false
	}

	addNewDomain(db, domain)

}

func authoriseScoutExpedition(db *mongo.Client, domainID *primitive.ObjectID, url string) bool {

	collection := db.Database("web-search").Collection("domains")
	filter := bson.D{{Key: "_id", Value: domainID}}

	var domain Domain

	err := collection.FindOne(context.TODO(), filter).Decode(&domain)

	if err != nil {
		return false
	}

	// If no restrictions allow expedition
	if domain.Restrictions {
		return true
	}

	// If url contains disallowed pattern do not allow expedition
	fmt.Println("Disallowed Patterns")
	for _, pattern := range domain.Disallowed {
		if string(pattern[len(pattern)-1]) == "*" {
			pattern = pattern[0 : len(pattern)-1]
		}
		re := regexp.MustCompile(pattern)
		if len(re.FindAllString(url, -1)) > 0 {
			return false
		}
	}

	// If url contains allowed pattern allow expedition
	fmt.Println("Allowed Patterns")
	for _, pattern := range domain.Allowed {
		re := regexp.MustCompile(pattern)
		if len(re.FindAllString(url, -1)) > 0 {
			return true
		}
	}

	return false
}
