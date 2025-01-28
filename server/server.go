// server.go
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	// Use the xeodou fork of go-sqlcipher
	_ "github.com/xeodou/go-sqlcipher"
)

type Client struct {
	conn     net.Conn
	username string
}

var (
	clients       = make(map[net.Conn]*Client)
	clientsMutex  sync.Mutex
	db            *sql.DB
	encryptionKey string
	masterRegKey  string // single registration code for new signups
)

// generateEncryptionKey returns a random 256-bit encryption key in hex format
func generateEncryptionKey() string {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate encryption key: %v", err)
	}
	return hex.EncodeToString(key)
}

// generateRegistrationKey returns a random 20-hex-character code
func generateRegistrationKey() string {
	key := make([]byte, 20)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate registration key: %v", err)
	}
	return hex.EncodeToString(key)[:20]
}

// hashPassword returns the SHA-256 hex digest of a password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// initDatabase initializes an in-memory, SQLCipher-encrypted SQLite DB.
func initDatabase() {
	var err error
	// Open the SQLite database in memory using sqlcipher driver.
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}

	// Set the encryption key for SQLCipher
	_, err = db.Exec(fmt.Sprintf("PRAGMA key = '%s';", encryptionKey))
	if err != nil {
		log.Fatalf("Failed to set encryption key: %v", err)
	}

	// Create the users table
	_, err = db.Exec(`
        CREATE TABLE users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT UNIQUE NOT NULL,
            password TEXT NOT NULL
        );
    `)
	if err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	fmt.Fprintln(conn, "Welcome to the secure chat server!")
	fmt.Fprintln(conn, "Enter 'login' or 'register': ")

	userChoice, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Error reading choice: %v", err)
		return
	}

	userChoice = strings.TrimSpace(userChoice)

	if strings.ToLower(userChoice) == "register" {
		fmt.Fprintln(conn, "Enter the server's registration code: ")
		regAttempt, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading registration code: %v", err)
			return
		}

		// Trim whitespace and remove square brackets
		regAttempt = strings.TrimSpace(regAttempt)

		// Replace '[' and ']' characters
		regAttempt = strings.ReplaceAll(regAttempt, "[", "")
		regAttempt = strings.ReplaceAll(regAttempt, "]", "")

		// If code doesn't match, disconnect
		if regAttempt != masterRegKey {
			fmt.Fprintln(conn, "Invalid registration code. Closing connection.")
			return
		}

		fmt.Fprintln(conn, "Enter your desired username: ")
		usr, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading username: %v", err)
			return
		}
		usr = strings.TrimSpace(usr)

		// Note: actual password hiding is a client-side feature
		fmt.Fprintln(conn, "Enter your desired password (typing not hidden): ")
		pwd, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading password: %v", err)
			return
		}
		pwd = strings.TrimSpace(pwd)

		hashed := hashPassword(pwd)
		// Insert into DB
		_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", usr, hashed)
		if err != nil {
			fmt.Fprintln(conn, "Failed to register: %v\n", err)
			return
		}
		fmt.Fprintln(conn, "Registration successful! You can now login.")
		return

	} else if strings.ToLower(userChoice) == "login" {
		fmt.Fprintln(conn, "Username: ")
		usr, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading username: %v", err)
			return
		}
		usr = strings.TrimSpace(usr)

		fmt.Fprintln(conn, "Password (typing not hidden): ")
		pwd, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading password: %v", err)
			return
		}
		pwd = strings.TrimSpace(pwd)

		var storedPassword string
		row := db.QueryRow("SELECT password FROM users WHERE username = ?", usr)
		err = row.Scan(&storedPassword)
		if err != nil {
			fmt.Fprintln(conn, "Invalid username or password.")
			return
		}

		if hashPassword(pwd) != storedPassword {
			fmt.Fprintln(conn, "Invalid username or password.")
			return
		}

		fmt.Fprintf(conn, "Welcome back, %s!\n", usr)

		// Add client
		clientsMutex.Lock()
		clients[conn] = &Client{conn: conn, username: usr}
		clientsMutex.Unlock()

		broadcast(fmt.Sprintf("%s has joined the chat", usr), conn)

		// Read messages in a loop
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				clientsMutex.Lock()
				delete(clients, conn)
				clientsMutex.Unlock()
				broadcast(fmt.Sprintf("%s has left the chat", usr), conn)
				return
			}
			message := string(buf[:n])
			broadcast(fmt.Sprintf("%s: %s", usr, message), conn)
		}
	} else {
		fmt.Fprintln(conn, "Invalid choice. Closing.")
		return
	}
}

// broadcast sends the message to all connected clients except the sender
func broadcast(message string, sender net.Conn) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for c, client := range clients {
		if c != sender {
			fmt.Fprintln(c, message)
		}
		_ = client // avoid unused variable warning
	}
}

func main() {
	// Generate ephemeral encryption key
	encryptionKey = generateEncryptionKey()
	initDatabase()

	// Also generate a master registration key on startup
	masterRegKey = generateRegistrationKey()

	log.Println("Secure (SQLCipher) chat server started on port 9000...")
	log.Println("Encryption Key generated on startup. Database is ephemeral.")
	log.Printf("Registration Key for new signups: %s\n", masterRegKey)

	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleClient(conn)
	}
}
