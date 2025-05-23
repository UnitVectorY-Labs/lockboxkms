package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/UnitVectorY-Labs/lockboxkms/internal/config"
	"github.com/UnitVectorY-Labs/lockboxkms/internal/kms"
)

const (
	maxKeyNameLength = 63
	maxPlaintextSize = 64 * 1024 // 64 KiB
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	if cfg.ProjectID == "" {
		log.Fatal("GCP_PROJECT environment variable must be set")
	}

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

	// Regular expression for validating key names
	keyNameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// Load the HTML template
	tpl := template.Must(template.ParseFiles("templates/index.html"))

	// Set up HTTP handlers
	http.HandleFunc("/", getHomeHandler(cfg, tpl))
	http.HandleFunc("/keys", getKeysHandler(kmsClient))
	http.HandleFunc("/encrypt", encryptHandler(cfg, kmsClient, keyNameRegex))

	// Start the server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}

// serveHome renders the main HTML page
func getHomeHandler(cfg config.Config, tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		err := tpl.Execute(w, cfg)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Template execution error: %v", err)
		}
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
func encryptHandler(cfg config.Config, client *kms.Client, keyNameRegex *regexp.Regexp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keyName := r.FormValue("key")
		plaintext := r.FormValue("text")

		if keyName == "" || plaintext == "" {
			http.Error(w, "Key and text are required", http.StatusBadRequest)
			return
		}

		expectedPrefix := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/", cfg.ProjectID, cfg.Location, cfg.KeyRing)
		if !strings.HasPrefix(keyName, expectedPrefix) {
			http.Error(w, "Invalid keyName format", http.StatusBadRequest)
			return
		}

		shortName := keyName[strings.LastIndex(keyName, "/")+1:]

		if !keyNameRegex.MatchString(shortName) {
			http.Error(w, "Invalid keyName format", http.StatusBadRequest)
			return
		}

		if len(shortName) > maxKeyNameLength {
			http.Error(w, fmt.Sprintf("keyName exceeds maximum length of %d characters", maxKeyNameLength), http.StatusBadRequest)
			return
		}

		if len(plaintext) >= maxPlaintextSize {
			http.Error(w, fmt.Sprintf("plaintext exceeds maximum size of %d bytes", maxPlaintextSize), http.StatusBadRequest)
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
