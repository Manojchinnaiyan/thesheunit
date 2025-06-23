package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run scripts/generate_password.go <password>")
	}

	password := os.Args[1]
	cost := 12

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		log.Fatal("Error generating hash:", err)
	}

	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Hash: %s\n", string(hash))

	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		log.Fatal("Hash verification failed:", err)
	}

	fmt.Println("✅ Hash verified successfully!")
}
