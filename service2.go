package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"net/http"
	"regexp"
)

type SaltResponse struct {
	Salt string `json:"salt"`
}

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Email    string             `bson:"email"`
	Salt     string             `bson:"salt"`
	Password string             `bson:"password"`
}

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/create-user", createUserHandler)
	r.Get("/get-user/{email}", getUserHandler)

	err := http.ListenAndServe(":8081", r)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var user User
	err := decoder.Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !isValidEmail(user.Email) {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}

	if isDuplicateEmail(user.Email) {
		http.Error(w, "Email address already exists", http.StatusBadRequest)
		return
	}

	salt, err := getSaltFromService1()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.Salt = salt

	hash := md5.Sum([]byte(salt + user.Password))
	user.Password = hex.EncodeToString(hash[:])

	err = saveUserToMongo(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")

	user, err := getUserByEmail(email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.NotFound(w, r)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(user)
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func isDuplicateEmail(email string) bool {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("строка для подключения к монго"))
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.Background())

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		panic(err)
	}

	collection := client.Database("test").Collection("users")

	filter := bson.D{{"email", email}}

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		panic(err)
	}

	return count > 0
}

func getSaltFromService1() (string, error) {
	requestBody, err := json.Marshal(map[string]string{})
	if err != nil {
		return "", err
	}

	resp, err := http.Post("http://localhost:3000/generate-salt", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var saltResponse struct {
		Salt string `json:"salt"`
	}
	err = json.NewDecoder(resp.Body).Decode(&saltResponse)
	if err != nil {
		return "", err
	}

	return saltResponse.Salt, nil
}

func saveUserToMongo(user User) error {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("строка для подключения к монго"))
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		return err
	}

	collection := client.Database("test").Collection("users")

	_, err = collection.InsertOne(context.Background(), user)
	if err != nil {
		return err
	}

	return nil
}

func getUserByEmail(email string) (User, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("строка для подключения к монго"))
	if err != nil {
		return User{}, err
	}
	defer client.Disconnect(context.Background())

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		return User{}, err
	}

	collection := client.Database("test").Collection("users")

	filter := bson.D{{"email", email}}

	var user User
	err = collection.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}
