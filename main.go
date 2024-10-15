package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/iterator"
)

var tpl = template.Must(template.ParseFiles("templates/index.html"))

type Config struct {
	ProjectID string
	Location  string
	KeyRing   string
}

func main() {
	// Load configuration from environment variables
	config := Config{
		ProjectID: getEnv("GCP_PROJECT", "example-project"),
		Location:  getEnv("KMS_LOCATION", "us"),
		KeyRing:   getEnv("KMS_KEY_RING", "lockboxkms"),
	}

	// Initialize KMS client
	ctx := context.Background()
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create KMS client: %v", err)
	}
	defer client.Close()

	// Verify essential configurations
	if config.ProjectID == "" || config.KeyRing == "" {
		log.Fatal("GCP_PROJECT and KMS_KEY_RING environment variables must be set")
	}

	// Set up HTTP handlers
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		getKeys(w, r, client, config)
	})
	http.HandleFunc("/encrypt", encryptHandler)

	// Start the server on port specified by PORT environment variable or default to 8080
	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}

// getEnv retrieves environment variables or returns a default value
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// serveHome renders the main HTML page
func serveHome(w http.ResponseWriter, r *http.Request) {
	err := tpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Template execution error: %v", err)
	}
}

// getKeys handles the /keys endpoint and returns the list of KMS keys
func getKeys(w http.ResponseWriter, _ *http.Request, client *kms.KeyManagementClient, config Config) {

	fmt.Println("Getting keys")

	keys, err := listKMSKeys(context.Background(), client, config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list keys: %v", err), http.StatusInternalServerError)
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

// encryptHandler handles the /encrypt endpoint and returns the Base64 encoded text
func encryptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	keyName := r.FormValue("key")
	plaintext := r.FormValue("text")

	if keyName == "" || plaintext == "" {
		http.Error(w, "Key and text are required", http.StatusBadRequest)
		return
	}

	// Initialize KMS client (ensure it's properly set up)
	ctx := context.Background()
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		http.Error(w, "Failed to create KMS client", http.StatusInternalServerError)
		log.Printf("KMS client error: %v", err)
		return
	}
	defer client.Close()

	req := &kmspb.EncryptRequest{
		Name:      keyName,
		Plaintext: []byte(plaintext),
	}

	resp, err := client.Encrypt(ctx, req)
	if err != nil {
		http.Error(w, "Encryption failed", http.StatusInternalServerError)
		log.Printf("Encryption error: %v", err)
		return
	}

	encryptedText := base64.StdEncoding.EncodeToString(resp.Ciphertext)

	// Return the encrypted text
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(encryptedText))
}

// listKMSKeys retrieves all symmetric encryption keys from the specified key ring
func listKMSKeys(ctx context.Context, client *kms.KeyManagementClient, config Config) (map[string]string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", config.ProjectID, config.Location, config.KeyRing)
	it := client.ListCryptoKeys(ctx, &kmspb.ListCryptoKeysRequest{
		Parent: parent,
		Filter: "purpose:ENCRYPT_DECRYPT",
	})

	keys := make(map[string]string)
	for {
		cryptoKey, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		// Extract the short key name from the full resource name
		shortName := cryptoKey.Name[strings.LastIndex(cryptoKey.Name, "/")+1:]
		keys[cryptoKey.Name] = shortName
	}

	return keys, nil
}
