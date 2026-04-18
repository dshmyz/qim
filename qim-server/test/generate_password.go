package test

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func test() {
	password := "123456"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error generating hash:", err)
		return
	}
	fmt.Println("Password:", password)
	fmt.Println("Hash:", string(hash))
}
