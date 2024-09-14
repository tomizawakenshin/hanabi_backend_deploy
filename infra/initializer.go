// package infra

// import (
// 	"log"

// 	"github.com/joho/godotenv"
// )

// func Initialize() {
// 	err := godotenv.Load()
// 	if err != nil {
// 		log.Fatal("Error loading .env file")
// 	}
// }

package infra

import (
	"fmt"
)

func Initialize() {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }
	fmt.Println("In Initializer.go")
}
