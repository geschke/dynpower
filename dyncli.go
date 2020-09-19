package main

import (
	"flag"
	"fmt"
	"os"

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

func main() {
	fmt.Println("dyncli is a small helper tool to manage the dynpower database.")

	/*dbPasswordPtr := flag.String("password", "", "Database password")
	dbHostPtr := flag.String("host", "", "Database server")
	dbNamePtr := flag.String("dbname")
	*/
	//numbPtr := flag.Int("numb", 42, "an int")
	//boolPtr := flag.Bool("fork", false, "a bool")

	//var svar string
	//flag.StringVar(&svar, "svar", "bar", "a string var")

	// todo maybe: use flag subcommands
	flag.Parse()

	switch flag.Arg(0) {
	case "encrypt":
		password := flag.Arg(1)
		if len(password) < 1 {
			fmt.Println("\nPassword parameter missing. \n")
			os.Exit(1)
			//panic(err.Error()) // proper error handling instead of panic in your app
		}
		encrypt(password)
	default:
		fmt.Println("Unknown or undefined command, please use the following commands:\n")
		fmt.Println("\tencrypt <password> :\t Encrypt password string to enter into database table.")
		fmt.Println("\n\n")
		os.Exit(1)
		return
	}

	//fmt.Println("word:", *wordPtr)
	//fmt.Println("numb:", *numbPtr)
	//fmt.Println("fork:", *boolPtr)
	//fmt.Println("svar:", svar)
	//fmt.Println("tail:", flag.Args())
}
