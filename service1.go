package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/generate-salt", func(w http.ResponseWriter, r *http.Request) {
		salt := generateSalt()
		json.NewEncoder(w).Encode(map[string]string{"salt": salt})
	})

	http.ListenAndServe(":3000", nil)
}

func generateSalt() string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	salt := make([]rune, 12)
	for i := range salt {
		salt[i] = chars[rand.Intn(len(chars))]
	}
	return string(salt)
}
