package cmd

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func encrypt(pw string) {
	fmt.Println("Password: ")
	fmt.Println(pw)
	password := []byte(pw)
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	fmt.Println("\nYour encrypted password:\n")
	fmt.Println(string(hashedPassword))
	fmt.Println("\nPlease enter this string in the field 'access_key' into the domains table.\n")

	// test checking:
	err = bcrypt.CompareHashAndPassword(hashedPassword, password)
	fmt.Println(err) // nil means it is a match

}
