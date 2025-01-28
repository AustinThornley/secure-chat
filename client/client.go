// chat_client.go
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type clientState int

const (
	stateLogin clientState = iota
	statePassword
	stateChat
)

type model struct {
	messages  []string
	input     string
	conn      net.Conn
	exit      bool
	state     clientState
	prevState clientState
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ─────────────────────────────────────────────────────────────────────────────
	// KEYBOARD INPUT:
	// ─────────────────────────────────────────────────────────────────────────────
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.conn != nil && len(m.input) > 0 {
				if m.input == "/exit" {
					return m.exitProgram()
				}
				// Send typed input to the server
				fmt.Fprintln(m.conn, m.input)

				// If in chat mode, display local message
				if m.state == stateChat {
					m.messages = append(m.messages, "You: "+m.input)
				}

				// If we’re in hidden password mode, revert to previous state after sending
				if m.state == statePassword {
					m.state = m.prevState
				}
			}
			m.input = "" // Clear input on enter

		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}

		default:
			// If we’re in password mode, do not echo typed chars
			if m.state == statePassword {
				m.input += msg.String() // stored but not displayed
			} else {
				// Normal state => echo typed chars
				m.input += msg.String()
			}
		}

	// ─────────────────────────────────────────────────────────────────────────────
	// SERVER LINES (STRING):
	// ─────────────────────────────────────────────────────────────────────────────
	case string:
		serverLine := strings.TrimRight(msg, "\r\n")

		// If the server closed the connection, exit the program.
		if serverLine == "Connection closed by server." {
			// Display it for clarity, then quit
			m.messages = append(m.messages, serverLine)
			return m.exitProgram()
		}

		// 1) If server prompts for a password => switch to hidden input
		if strings.Contains(serverLine, "(typing not hidden):") {
			m.prevState = m.state
			m.state = statePassword
		}

		// 2) If we see “Welcome back” or “has joined the chat,” user is fully logged in
		if strings.Contains(serverLine, "Welcome back") ||
			strings.Contains(serverLine, "has joined the chat") {
			// Clear all old login lines so we start fresh for the chat
			m.messages = nil
			m.state = stateChat

			// Add the welcome line (so they can see it)
			// or comment this out if you don’t want to show it
			m.messages = append(m.messages, serverLine)
			return m, nil
		}

		// 3) For everything else, just display in TUI
		if trimmed := strings.TrimSpace(serverLine); trimmed != "" {
			m.messages = append(m.messages, trimmed)
		}
	}
	return m, nil
}

func (m model) View() string {
	var sb strings.Builder
	for _, line := range m.messages {
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\nType /exit to quit.\n> ")

	// If in password mode, hide typed input
	if m.state == statePassword {
		sb.WriteString(strings.Repeat("*", len(m.input)))
	} else {
		sb.WriteString(m.input)
	}
	return sb.String()
}

func (m model) exitProgram() (tea.Model, tea.Cmd) {
	m.exit = true
	return m, tea.Quit
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the server address (e.g., localhost:9000): ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	// Initial model is in login state
	m := model{conn: conn, state: stateLogin}

	p := tea.NewProgram(m)

	// Read server lines
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			p.Send(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			p.Send(fmt.Sprintf("Error reading from server: %v", err))
		} else {
			p.Send("Connection closed by server.")
		}
	}()

	if _, runErr := p.Run(); runErr != nil {
		fmt.Println("Error running program:", runErr)
		os.Exit(1)
	}

	fmt.Println("Exiting chat client. Goodbye!")
}
