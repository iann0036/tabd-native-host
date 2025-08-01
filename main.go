package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ClipboardData represents the simplified data structure received from the browser extension
type ClipboardData struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
	URL       string `json:"url"`
	Title     string `json:"title"`
}

// Response represents the response sent back to the browser extension
type Response struct {
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// TabdNativeHost handles native messaging communication
type TabdNativeHost struct {
	tabdDir       string
	logFile       *os.File
	secureStorage SecureStorage
}

// NewTabdNativeHost creates a new native host instance
func NewTabdNativeHost() (*TabdNativeHost, error) {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// Create ~/.tabd directory
	tabdDir := filepath.Join(homeDir, ".tabd")
	if err := os.MkdirAll(tabdDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .tabd directory: %v", err)
	}

	var logFile *os.File

	// Only set up logging if debug environment variable is set
	if os.Getenv("TABD_DEBUG") != "" {
		// Open log file
		logPath := filepath.Join(tabdDir, "native-host.log")
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}

		// Set up logging
		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		// Disable logging by sending to a discard writer
		log.SetOutput(io.Discard)
	}

	return &TabdNativeHost{
		tabdDir:       tabdDir,
		logFile:       logFile,
		secureStorage: NewSecureStorage(tabdDir),
	}, nil
}

// Close closes the native host resources
func (t *TabdNativeHost) Close() {
	if t.logFile != nil {
		t.logFile.Close()
	}
}

// readMessage reads a message from stdin using Chrome's native messaging format
func (t *TabdNativeHost) readMessage() ([]byte, error) {
	// Read the message length (4 bytes, little-endian)
	var length uint32
	if err := binary.Read(os.Stdin, binary.LittleEndian, &length); err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read message length: %v", err)
	}

	// Validate message length
	if length == 0 || length > 1024*1024 { // Max 1MB message
		return nil, fmt.Errorf("invalid message length: %d", length)
	}

	// Read the message data
	message := make([]byte, length)
	if _, err := io.ReadFull(os.Stdin, message); err != nil {
		return nil, fmt.Errorf("failed to read message data: %v", err)
	}

	return message, nil
}

// sendMessage sends a message to stdout using Chrome's native messaging format
func (t *TabdNativeHost) sendMessage(message []byte) error {
	// Write message length (4 bytes, little-endian)
	length := uint32(len(message))
	if err := binary.Write(os.Stdout, binary.LittleEndian, length); err != nil {
		return fmt.Errorf("failed to write message length: %v", err)
	}

	// Write message data
	if _, err := os.Stdout.Write(message); err != nil {
		return fmt.Errorf("failed to write message data: %v", err)
	}

	return nil
}

// saveClipboardData saves clipboard data to secure storage
func (t *TabdNativeHost) saveClipboardData(data *ClipboardData) error {
	// Convert to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal clipboard data: %v", err)
	}

	// Store in secure storage
	return t.secureStorage.Store("latest_clipboard", jsonData)
}

// getClipboardData retrieves clipboard data from secure storage
func (t *TabdNativeHost) getClipboardData() (*ClipboardData, error) {
	// Retrieve from secure storage
	jsonData, err := t.secureStorage.Retrieve("latest_clipboard")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve clipboard data: %v", err)
	}

	// Parse JSON
	var data ClipboardData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clipboard data: %v", err)
	}

	return &data, nil
}

// handleMessage processes incoming messages from the browser extension
func (t *TabdNativeHost) handleMessage(messageData []byte) error {
	// Parse the message
	var data ClipboardData
	if err := json.Unmarshal(messageData, &data); err != nil {
		return fmt.Errorf("failed to parse message: %v", err)
	}

	// Save to secure storage
	if err := t.saveClipboardData(&data); err != nil {
		log.Printf("Error saving clipboard data: %v", err)

		// Send error response
		response := Response{
			Status:    "error",
			Message:   fmt.Sprintf("Failed to save clipboard data: %v", err),
			Timestamp: time.Now().Unix(),
		}

		responseData, _ := json.Marshal(response)
		return t.sendMessage(responseData)
	}

	// Send success response
	response := Response{
		Status:    "success",
		Message:   "Clipboard data saved successfully",
		Timestamp: time.Now().Unix(),
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %v", err)
	}

	return t.sendMessage(responseData)
}

// run starts the native messaging loop
func (t *TabdNativeHost) run() error {
	log.Println("Tab'd Native Host started")

	for {
		// Read message from browser extension
		messageData, err := t.readMessage()
		if err != nil {
			if err == io.EOF {
				log.Println("Browser extension disconnected")
				break
			}
			log.Printf("Error reading message: %v", err)
			continue
		}

		// Handle the message
		if err := t.handleMessage(messageData); err != nil {
			log.Printf("Error handling message: %v", err)
		}
	}

	return nil
}

func main() {
	// Check if this is a getclipboard command
	if len(os.Args) > 1 && os.Args[1] == "getclipboard" {
		// Handle getclipboard command
		host, err := NewTabdNativeHost()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create native host: %v\n", err)
			os.Exit(1)
		}
		defer host.Close()

		// Retrieve clipboard data
		data, err := host.getClipboardData()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to retrieve clipboard data: %v\n", err)
			os.Exit(1)
		}

		// Output as JSON
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to encode clipboard data: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create native host for native messaging
	host, err := NewTabdNativeHost()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create native host: %v\n", err)
		os.Exit(1)
	}
	defer host.Close()

	// Run the native messaging loop
	if err := host.run(); err != nil {
		log.Printf("Native host error: %v", err)
		os.Exit(1)
	}

	log.Println("Tab'd Native Host shutdown")
}
