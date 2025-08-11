package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Connection parameters from the screenshot
	host := "localhost"
	port := 5433
	user := "postgres"
	password := "admin123" // Password provided by user
	dbname := "sunat"      // Maintenance database

	// Build connection string
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	fmt.Println("Attempting to connect to PostgreSQL...")
	fmt.Printf("Connection string: host=%s port=%d user=%s dbname=%s\n", host, port, user, dbname)

	// Open database connection
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")

	// List all databases
	fmt.Println("\nListing all databases:")
	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		log.Fatal("Error querying databases:", err)
	}
	defer rows.Close()

	databases := []string{}
	for rows.Next() {
		var dbName string
		err := rows.Scan(&dbName)
		if err != nil {
			log.Fatal("Error scanning row:", err)
		}
		databases = append(databases, dbName)
		fmt.Printf("- %s\n", dbName)
	}

	// Check if 'wirai' database exists
	wiraiExists := false
	for _, db := range databases {
		if db == "sunat" {
			wiraiExists = true
			break
		}
	}

	if wiraiExists {
		fmt.Println("\n✅ Database 'sunat' exists!")

		// Try to connect to sunat database
		wiraiConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, "sunat")

		wiraiDB, err := sql.Open("postgres", wiraiConnStr)
		if err == nil {
			defer wiraiDB.Close()
			if err = wiraiDB.Ping(); err == nil {
				fmt.Println("Successfully connected to 'sunat' database!")

				// List tables in sunat database
				fmt.Println("\nTables in 'sunat' database:")
				tableRows, err := wiraiDB.Query(`
					SELECT table_name 
					FROM information_schema.tables 
					WHERE table_schema = 'public' 
					ORDER BY table_name
				`)
				if err == nil {
					defer tableRows.Close()
					for tableRows.Next() {
						var tableName string
						tableRows.Scan(&tableName)
						fmt.Printf("- %s\n", tableName)
					}
				}
			}
		}
	} else {
		fmt.Println("\n❌ Database 'sunat' does not exist")
	}
}
