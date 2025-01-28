```markdown
# Secure Ephemeral Chat Application

This tool provides a simple **terminal-based chat application** using **Go** and **Bubble Tea**. It features:

- **In-Memory SQLite database** encrypted with **SQLCipher** (no data is persisted after server shutdown).
- **Ephemeral encryption key** generated on server start.
- **Random registration code** required for new user sign-up.
- **Hashed passwords** stored in the in-memory database (SHA-256).
- **Terminal UI (TUI) client** built with [Charm’s Bubble Tea](https://github.com/charmbracelet/bubbletea) for interactive text-based usage.
- **Hidden password input** using a “password state,” so typed characters are replaced with asterisks during login/registration.

## Table of Contents

- [Overview](#overview)
- [Security Features](#security-features)
- [Dependencies](#dependencies)
- [Server Usage](#server-usage)
- [Client Usage](#client-usage)
- [How It Works](#how-it-works)
  - [Ephemeral Encryption Key](#ephemeral-encryption-key)
  - [Registration Code](#registration-code)
  - [Login/Registration Flow](#loginregistration-flow)
  - [No Data Persistence](#no-data-persistence)
- [Possible Enhancements](#possible-enhancements)
- [License](#license)

---

## Overview

This repository contains:
1. **`server.go`** – Starts an in-memory SQLCipher-encrypted chat server on port `9000`.
2. **`client.go`** – A Bubble Tea client that connects to the server, shows prompts, and allows interactive chat.

**Goal**: Provide a simple, secure, ephemeral chat environment where no messages or user data persist beyond the server’s uptime.

---

## Security Features

1. **Ephemeral, Encrypted Database**  
   - Uses [xeodou’s go-sqlcipher](https://github.com/xeodou/go-sqlcipher) to open an SQLite database in memory, setting a random **256-bit** encryption key on each server start.
   - **No disk writes**; data disappears when the server stops.

2. **User Credential Security**  
   - On registration, passwords are hashed with **SHA-256** before storing in the in-memory database.
   - On login, the server verifies hashed credentials.

3. **Registration Code**  
   - A single random 20-character code is generated at server startup.
   - Required for new user registration—prevents unauthorized sign-ups.

4. **Hidden Password Entry** (in the client)  
   - The Bubble Tea client uses a separate password “state” to mask typed characters with asterisks, ensuring passwords are not displayed in the TUI.

5. **No Persistent Logs**  
   - All messages and user data are stored only in memory. Once the server is shut down, **everything** is lost.

---

## Dependencies

1. **Go 1.18+** (or newer)  
2. **xeodou/go-sqlcipher**  
   - Add with:  
     ```bash
     go get github.com/xeodou/go-sqlcipher
     ```
3. **Bubble Tea**  
   - Add with:  
     ```bash
     go get github.com/charmbracelet/bubbletea
     ```
4. (Optional) **golang.org/x/term** for cross-platform password hiding, if you want to avoid terminal-state commands. Current client uses a password “state” approach.

---

## Server Usage

1. **Install Dependencies**:
   ```bash
   go get github.com/xeodou/go-sqlcipher
   go get github.com/charmbracelet/bubbletea
   ```
2. **Compile** the server:
   ```bash
   go build -o server ./server/server.go
   ```
3. **Run** the server:
   ```bash
   ./server
   ```
   - The server prints:
     - A random **encryption key** (for internal DB use).
     - A **registration key** for new sign-ups.
   - Example:
     ```
     Secure (SQLCipher) chat server started on port 9000...
     Encryption Key generated on startup. Database is ephemeral.
     Registration Key for new signups: 9f6074d23c35bda3b83e
     ```
4. **Keep** the server running; any data is ephemeral and in-memory only.

---

## Client Usage

1. **Compile** the client:
   ```bash
   go build -o client ./client/client.go
   ```
2. **Run** the client:
   ```bash
   ./client
   ```
3. **Enter Server Address**:
   - For local testing: `localhost:9000`
4. **Follow Prompts**:
   - Choose `register` or `login`.
   - If registering, provide the server’s **registration code**.
   - Enter **username** and **password**.
   - Once logged in, type messages to chat. Type `/exit` to quit the client.

---

## How It Works

### Ephemeral Encryption Key

- **On startup**, the server calls `generateEncryptionKey()` to produce a random **32-byte** (256-bit) key in hex.
- This key is used via `PRAGMA key` in SQLite (SQLCipher). Data is **never** stored on disk.

### Registration Code

- The server also generates a **20-character** hex code (`masterRegKey`) shown in the console.
- Anyone wanting to **register** must supply that code. If the code is wrong, the server rejects them.

### Login/Registration Flow

1. **Register**:
   - The user chooses “register” in the client’s prompt.
   - The server requests the **registration code**, username, and password.
   - On success, the new user is stored in the ephemeral database with a hashed password.
2. **Login**:
   - The user chooses “login,” enters username/password.
   - The server checks credentials against the ephemeral DB.

### No Data Persistence

- The database is purely **in-memory**. A server reboot destroys all user data.
- No logs or messages remain once the server exits.

---

## Possible Enhancements

1. **TLS Encryption**  
   - Wrap connections in TLS to protect messages in transit.  
2. **Argon2/Bcrypt**  
   - Use a more advanced password-hashing scheme than SHA-256.  
3. **Cross-Platform Password Hiding**  
   - Use `golang.org/x/term` for Windows compatibility.  
4. **Configurable Ports / CLI Flags**  
   - Expose server flags for port, encryption key length, etc.  
5. **Improved Logging**  
   - Possibly store ephemeral logs or hide them entirely.

---

## License

This project is provided “as is,” without warranty. It’s intended as an example of ephemeral, encrypted chat in Go. Check the repository or relevant packages (Bubble Tea, go-sqlcipher) for their respective licenses.