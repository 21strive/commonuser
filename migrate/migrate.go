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
		fmt.Fprintf(os.Stderr, "  %s -entity=admin -user=postgres -password=mypass -db=myapp -tables=account,reset,session\n", os.Args[0])
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

	// Start transaction for atomic table creation
	fmt.Println("Starting database transaction...")
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Fatalf("Transaction rolled back due to panic: %v", r)
		}
	}()

	// Create tables within transaction
	tables := []struct {
		name       string
		createFunc func(*sql.Tx, string) (bool, error)
	}{
		{"account", CreateAccountTableSQL},
		{"reset password", CreateResetPasswordTableSQL},
		{"update email", CreateUpdateEmailTableSQL},
		{"session", CreateSessionTableSQL},
		{"verification", CreateVerificationTableSQL},
		{"provider", CreateProviderTableSQL},
	}

	for _, table := range tables {
		fmt.Printf("Processing %s table for entity: %s\n", table.name, *entityName)
		created, err := table.createFunc(tx, *entityName)
		if err != nil {
			tx.Rollback()
			log.Fatalf("Failed to create %s table: %v", table.name, err)
		}

		if created {
			fmt.Printf("✓ %s table created successfully\n", capitalizeWords(table.name))
		} else {
			fmt.Printf("⚠ %s table already exists - skipping creation\n", capitalizeWords(table.name))
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Println("\nAll tables created successfully!")
}

func capitalizeWords(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.TrimSpace(s) == item {
			return true
		}
	}
	return false
}

func CreateResetPasswordTableSQL(tx *sql.Tx, entityName string) (bool, error) {
	tableName := entityName + "_reset_password"
	query := `CREATE TABLE IF NOT EXISTS ` + tableName + ` (
		uuid VARCHAR(255) PRIMARY KEY,
		randid VARCHAR(255) UNIQUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		account_uuid VARCHAR(255) NOT NULL,
		token VARCHAR(255) UNIQUE NOT NULL,
		expired_at TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_token ON ` + tableName + `(token);`

	result, err := tx.Exec(query)
	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func CreateAccountTableSQL(tx *sql.Tx, entityName string) (bool, error) {
	query := `CREATE TABLE IF NOT EXISTS ` + entityName + ` (
		uuid VARCHAR(255) PRIMARY KEY,
		randid VARCHAR(255) UNIQUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		name VARCHAR(255),
		username VARCHAR(255) UNIQUE,
		password VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		avatar VARCHAR(255),
		email_verified BOOLEAN DEFAULT FALSE
    );
    CREATE INDEX IF NOT EXISTS idx_` + entityName + `_email ON ` + entityName + `(email);
	CREATE INDEX IF NOT EXISTS idx_` + entityName + `_randid ON ` + entityName + `(randid);
	CREATE INDEX IF NOT EXISTS idx_` + entityName + `_uuid ON ` + entityName + `(uuid);
    CREATE INDEX IF NOT EXISTS idx_` + entityName + `_username ON ` + entityName + `(username);`

	result, err := tx.Exec(query)
	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func CreateUpdateEmailTableSQL(tx *sql.Tx, entityName string) (bool, error) {
	tableName := entityName + "_update_email"
	query := `CREATE TABLE IF NOT EXISTS ` + tableName + ` (
		uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		randid VARCHAR(255) UNIQUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		account_uuid UUID NOT NULL,
		previous_email_address VARCHAR(255),
		new_email_address VARCHAR(255) UNIQUE NOT NULL,
		reset_token VARCHAR(255) NOT NULL,
		revoke_token VARCHAR(255) NOT NULL,
		processed BOOLEAN DEFAULT FALSE,
		expired_at TIMESTAMP
    );
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_account_uuid ON ` + tableName + `(account_uuid);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_new_email_address ON ` + tableName + `(new_email_address);`

	result, err := tx.Exec(query)
	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func CreateSessionTableSQL(tx *sql.Tx, entityName string) (bool, error) {
	tableName := entityName + "_session"
	query := `CREATE TABLE IF NOT EXISTS ` + tableName + ` (
       uuid VARCHAR(255) PRIMARY KEY,
       randid VARCHAR(255) UNIQUE,
       created_at TIMESTAMP DEFAULT NOW(),
       updated_at TIMESTAMP DEFAULT NOW(),
       last_active_at TIMESTAMP DEFAULT NOW(),
       account_uuid VARCHAR(255) NOT NULL,
       device_id VARCHAR(255),
       device_type TEXT,
       user_agent TEXT,
       refresh_token VARCHAR(255) UNIQUE NOT NULL,
       expired_at TIMESTAMP NOT NULL,
       revoked BOOLEAN DEFAULT TRUE
    );
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_account_uuid ON ` + tableName + `(account_uuid);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_refresh_token ON ` + tableName + `(refresh_token);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_randid ON ` + tableName + `(randid);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_expired_at ON ` + tableName + `(expired_at);`

	result, err := tx.Exec(query)
	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func CreateVerificationTableSQL(tx *sql.Tx, entityName string) (bool, error) {
	tableName := entityName + "_verification"
	query := `CREATE TABLE IF NOT EXISTS ` + tableName + ` (
		uuid VARCHAR(255) PRIMARY KEY,
		randid VARCHAR(255) UNIQUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		account_uuid VARCHAR(255) NOT NULL,
		code VARCHAR(255) NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_account_uuid ON ` + tableName + `(account_uuid);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_code ON ` + tableName + `(code);`

	result, err := tx.Exec(query)
	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

func CreateProviderTableSQL(tx *sql.Tx, entityName string) (bool, error) {
	tableName := entityName + "_provider"
	query := `CREATE TABLE IF NOT EXISTS ` + tableName + ` (
		uuid VARCHAR(255) PRIMARY KEY,
		randid VARCHAR(255) UNIQUE,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW(),
		name VARCHAR(255),
		email VARCHAR(255),
		sub VARCHAR(255),
		issuer VARCHAR(255),
		account_uuid VARCHAR(255)
    );
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_email ON ` + tableName + `(email);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_sub ON ` + tableName + `(sub);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_issuer ON ` + tableName + `(issuer);
    CREATE INDEX IF NOT EXISTS idx_` + tableName + `_account_uuid ON ` + tableName + `(account_uuid);`

	result, err := tx.Exec(query)
	if err != nil {
		return false, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}
