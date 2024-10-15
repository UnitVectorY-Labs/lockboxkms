package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"lockboxkms/internal/config"
	"lockboxkms/internal/kms"
)

type PageData struct {
	GCPProject string
	KeyRing    string
}

var tpl = template.Must(template.ParseFiles("templates/index.html"))

var cfg config.Config

func main() {
	// Load configuration
	cfg = config.LoadConfig()

	// Initialize KMS client
	ctx := context.Background()
	kmsClient, err := kms.NewClient(ctx, kms.Config{
		ProjectID: cfg.ProjectID,
		Location:  cfg.Location,
		KeyRing:   cfg.KeyRing,
	})
	if err != nil {
		log.Fatalf("Failed to initialize KMS client: %v", err)
	}
	defer kmsClient.Close()

	// Verify essential configurations
	if cfg.ProjectID == "" || cfg.KeyRing == "" {
		log.Fatal("GCP_PROJECT and KMS_KEY_RING environment variables must be set")
	}

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))

	// Set up HTTP handlers
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/keys", getKeysHandler(kmsClient))
	http.HandleFunc("/encrypt", encryptHandler(kmsClient))

	// Start the server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}

// serveHome renders the main HTML page
func serveHome(w http.ResponseWriter, r *http.Request) {
	err := tpl.Execute(w, cfg)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
	}
}

// getKeysHandler returns an HTTP handler function for listing keys
func getKeysHandler(client *kms.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, err := client.ListKeys(context.Background())
		if err != nil {
			http.Error(w, "Failed to list keys", http.StatusInternalServerError)
			log.Printf("ListKeys error: %v", err)
			return
		}

		// Generate HTML options for the dropdown
		var options strings.Builder
		options.WriteString("<option value=\"\" disabled selected>Select key</option>")
		for name, shortName := range keys {
			options.WriteString(fmt.Sprintf("<option value=\"%s\">%s</option>", name, shortName))
		}

		// Return the options as HTML fragment
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(options.String()))
	}
}

// encryptHandler returns an HTTP handler function for encrypting text
func encryptHandler(client *kms.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keyName := r.FormValue("key")
		plaintext := r.FormValue("text")

		if keyName == "" || plaintext == "" {
			http.Error(w, "Key and text are required", http.StatusBadRequest)
			return
		}

		encryptedText, err := client.Encrypt(context.Background(), keyName, plaintext)
		if err != nil {
			http.Error(w, "Encryption failed", http.StatusInternalServerError)
			log.Printf("Encryption error: %v", err)
			return
		}

		// Return the encrypted text
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encryptedText))
	}
}
