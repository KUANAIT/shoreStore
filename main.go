package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Shoe struct {
	ID    string  `json:"id" bson:"id"`
	Name  string  `json:"name" bson:"name"`
	Brand string  `json:"brand" bson:"brand"`
	Size  int     `json:"size" bson:"size"`
	Price float64 `json:"price" bson:"price"`
}

var (
	client  *mongo.Client
	shoeCol *mongo.Collection
	ctx     = context.Background()
)

func main() {
	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	shoeCol = client.Database("shoeStore").Collection("shoes")

	http.HandleFunc("/create", createShoeHandler)
	http.HandleFunc("/getall", getAllShoes)
	http.HandleFunc("/getbyid", getShoeByID)
	http.HandleFunc("/delete", deleteShoeByID)
	http.HandleFunc("/update", updateShoeByID)

	log.Println("Server is running on port 3000")
	http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func updateShoeByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Only PUT method is supported.", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID is required.", http.StatusBadRequest)
		return
	}

	var updatedShoe Shoe
	err := json.NewDecoder(r.Body).Decode(&updatedShoe)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filter := bson.M{"id": id}
	update := bson.M{
		"$set": bson.M{
			"name":  updatedShoe.Name,
			"brand": updatedShoe.Brand,
			"size":  updatedShoe.Size,
			"price": updatedShoe.Price,
		},
	}

	result, err := shoeCol.UpdateOne(ctx, filter, update)
	if err != nil {
		http.Error(w, "Error updating shoe: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Shoe not found.", http.StatusNotFound)
		return
	}

	var shoe Shoe
	err = shoeCol.FindOne(ctx, filter).Decode(&shoe)
	if err != nil {
		http.Error(w, "Error retrieving updated shoe: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shoe)
}

func createShoeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported.", http.StatusMethodNotAllowed)
		return
	}

	var shoe Shoe
	err := json.NewDecoder(r.Body).Decode(&shoe)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shoe.ID = generateID()

	_, err = shoeCol.InsertOne(ctx, shoe)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shoe)
}

func getAllShoes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is supported.", http.StatusMethodNotAllowed)
		return
	}

	cursor, err := shoeCol.Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var shoes []Shoe
	for cursor.Next(ctx) {
		var shoe Shoe
		if err := cursor.Decode(&shoe); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		shoes = append(shoes, shoe)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shoes)
}

func getShoeByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is supported.", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID is required.", http.StatusBadRequest)
		return
	}

	var shoe Shoe
	err := shoeCol.FindOne(ctx, bson.M{"id": id}).Decode(&shoe)
	if err != nil {
		http.Error(w, "Shoe not found.", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shoe)
}

func deleteShoeByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE method is supported.", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID is required.", http.StatusBadRequest)
		return
	}

	_, err := shoeCol.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		http.Error(w, "Shoe not found.", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func generateID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d", rand.Intn(1000000))
}
