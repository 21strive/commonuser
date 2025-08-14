package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	var (
		dbHost     = flag.String("host", "localhost", "Database host")
		dbPort     = flag.String("port", "5432", "Database port")
		dbUser     = flag.String("user", "", "Database user")
		dbPassword = flag.String("password", "", "Database password")
		dbName     = flag.String("db", "", "Database name")
		entityName = flag.String("entity", "", "Entity name (required)")
		tables     = flag.String("tables", "all", "Tables to create: all, account, reset, update (comma-separated)")
		sslMode    = flag.String("ssl", "disable", "SSL mode (disable, require)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Creates database tables for user management.\n\n")
		fmt.Fprintf(os.Stderr, "Required flags:\n")
		fmt.Fprintf(os.Stderr, "  -entity string    Entity name (e.g., 'user', 'admin')\n")
		fmt.Fprintf(os.Stderr, "  -user string      Database user\n")
		fmt.Fprintf(os.Stderr, "  -password string  Database password\n")
		fmt.Fprintf(os.Stderr, "  -db string        Database name\n\n")
		fmt.Fprintf(os.Stderr, "Optional flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -entity=user -user=postgres -password=mypass -db=myapp\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -entity=admin -user=postgres -password=mypass -db=myapp -tables=account,reset\n", os.Args[0])
	}

	flag.Parse()

	// Validate required flags
	if *entityName == "" || *dbUser == "" || *dbName == "" {
		fmt.Fprintf(os.Stderr, "Error: Missing required flags\n\n")
		flag.Usage()
		os.Exit(1)
	}

	var connStr string
	if *dbPassword != "" {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			*dbHost, *dbPort, *dbUser, *dbPassword, *dbName, *sslMode)
	} else {
		connStr = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
			*dbHost, *dbPort, *dbUser, *dbName, *sslMode)
	}
	fmt.Println(connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Printf("Connected to database: %s\n", *dbName)

	// Parse tables to create
	tablesToCreate := strings.Split(strings.ToLower(*tables), ",")
	createAll := contains(tablesToCreate, "all")

	// Create tables
	if createAll || contains(tablesToCreate, "account") {
		fmt.Printf("Creating account table for entity: %s\n", *entityName)
		if err := CreateAccountTableSQL(db, *entityName); err != nil {
			log.Fatalf("Failed to create account table: %v", err)
		}
		fmt.Println("✓ Account table created successfully")
	}

	if createAll || contains(tablesToCreate, "reset") {
		fmt.Printf("Creating reset password table for entity: %s\n", *entityName)
		if err := CreateResetPasswordTableSQL(db, *entityName); err != nil {
			log.Fatalf("Failed to create reset password table: %v", err)
		}
		fmt.Println("✓ Reset password table created successfully")
	}

	if createAll || contains(tablesToCreate, "update") {
		fmt.Printf("Creating update email table for entity: %s\n", *entityName)
		if err := CreateUpdateEmailTableSQL(db, *entityName); err != nil {
			log.Fatalf("Failed to create update email table: %v", err)
		}
		fmt.Println("✓ Update email table created successfully")
	}

	fmt.Println("\nAll tables created successfully!")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.TrimSpace(s) == item {
			return true
		}
	}
	return false
}

func CreateResetPasswordTableSQL(db *sql.DB, entityName string) error {
	tableName := entityName + "resetpassword"
	query := `CREATE TABLE IF NOT EXISTS ` + tableName + ` (
       uuid VARCHAR(255) PRIMARY KEY,
       email VARCHAR(255) UNIQUE NOT NULL,
       accountuuid VARCHAR(255) NOT NULL,
       createdat TIMESTAMP DEFAULT NOW(),
       updatedat TIMESTAMP DEFAULT NOW(),
       token VARCHAR(255) UNIQUE NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_email ON ` + tableName + `(email);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_token ON ` + tableName + `(token);`

	_, err := db.Exec(query)
	return err
}

func CreateAccountTableSQL(db *sql.DB, entityName string) error {
	query := `CREATE TABLE IF NOT EXISTS ` + entityName + ` (
       uuid VARCHAR(255) PRIMARY KEY,
       randId VARCHAR(255) UNIQUE,
       createdat TIMESTAMP DEFAULT NOW(),
       updatedat TIMESTAMP DEFAULT NOW(),
       name VARCHAR(255),
       username VARCHAR(255) UNIQUE,
       password VARCHAR(255) NOT NULL,
       email VARCHAR(255) UNIQUE NOT NULL,
       avatar VARCHAR(255),
       suspended BOOLEAN DEFAULT FALSE
    );
    CREATE INDEX IF NOT EXISTS idx_` + entityName + `_email ON ` + entityName + `(email);
	CREATE INDEX IF NOT EXISTS idx_` + entityName + `_randid ON ` + entityName + `(randid);
	CREATE INDEX IF NOT EXISTS idx_` + entityName + `_uuid ON ` + entityName + `(uuid);
    CREATE INDEX IF NOT EXISTS idx_` + entityName + `_username ON ` + entityName + `(username);`

	_, err := db.Exec(query)
	return err
}

func CreateUpdateEmailTableSQL(db *sql.DB, entityName string) error {
	tableName := entityName + "updateemail"
	query := `CREATE TABLE IF NOT EXISTS ` + tableName + ` (
       uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       randId VARCHAR(255) UNIQUE,
       createdat TIMESTAMP DEFAULT NOW(),
       updatedat TIMESTAMP DEFAULT NOW(),
       accountuuid UUID NOT NULL,
       previousemailaddress VARCHAR(255),
       newemailaddress VARCHAR(255) UNIQUE NOT NULL,
       resettoken VARCHAR(255) NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_accountuuid ON ` + tableName + `(accountuuid);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_newemail ON ` + tableName + `(newemailaddress);`

	_, err := db.Exec(query)
	return err
}
