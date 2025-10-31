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
	fmt.Printf("Creating account table for entity: %s\n", *entityName)
	if err := CreateAccountTableSQL(tx, *entityName); err != nil {
		tx.Rollback()
		log.Fatalf("Failed to create account table: %v", err)
	}
	fmt.Println("✓ Account table created successfully")

	fmt.Printf("Creating reset password table for entity: %s\n", *entityName)
	if err := CreateResetPasswordTableSQL(tx, *entityName); err != nil {
		tx.Rollback()
		log.Fatalf("Failed to create reset password table: %v", err)
	}
	fmt.Println("✓ Reset password table created successfully")

	fmt.Printf("Creating update email table for entity: %s\n", *entityName)
	if err := CreateUpdateEmailTableSQL(tx, *entityName); err != nil {
		tx.Rollback()
		log.Fatalf("Failed to create update email table: %v", err)
	}
	fmt.Println("✓ Update email table created successfully")

	fmt.Printf("Creating session table for entity: %s\n", *entityName)
	if err := CreateSessionTableSQL(tx, *entityName); err != nil {
		tx.Rollback()
		log.Fatalf("Failed to create session table: %v", err)
	}
	fmt.Println("✓ Session table created successfully")

	fmt.Printf("Creating verification table for entity: %s\n", *entityName)
	if err := CreateVerificationTableSQL(tx, *entityName); err != nil {
		tx.Rollback()
		log.Fatalf("Failed to create verification table: %v", err)
	}
	fmt.Println("✓ Verification table created successfully")

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
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

func CreateResetPasswordTableSQL(tx *sql.Tx, entityName string) error {
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

	_, err := tx.Exec(query)
	return err
}

func CreateAccountTableSQL(tx *sql.Tx, entityName string) error {
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

	_, err := tx.Exec(query)
	return err
}

func CreateUpdateEmailTableSQL(tx *sql.Tx, entityName string) error {
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

	_, err := tx.Exec(query)
	return err
}

func CreateSessionTableSQL(tx *sql.Tx, entityName string) error {
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

	_, err := tx.Exec(query)
	return err
}

func CreateVerificationTableSQL(tx *sql.Tx, entityName string) error {
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

	_, err := tx.Exec(query)
	return err
}
