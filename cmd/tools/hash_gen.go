package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := flag.String("password", "", "Password to hash")
	cost := flag.Int("cost", bcrypt.DefaultCost, "Bcrypt cost (4-31, default 10)")
	flag.Parse()

	if *password == "" {
		fmt.Fprintln(os.Stderr, "Usage: hash_gen -password <password> [-cost <cost>]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Generates a bcrypt hash of the provided password.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *cost < bcrypt.MinCost || *cost > bcrypt.MaxCost {
		fmt.Fprintf(os.Stderr, "Error: cost must be between %d and %d\n", bcrypt.MinCost, bcrypt.MaxCost)
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), *cost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(hash))
}
