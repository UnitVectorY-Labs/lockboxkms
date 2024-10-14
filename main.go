package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
)

var tpl = template.Must(template.ParseFiles("templates/index.html"))

func main() {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/foo", getKeys)

	// Start the server on port 8080
	http.ListenAndServe(":8080", nil)
}

// Handler to serve the home page
func serveHome(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w, nil)
}

func getKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := listKMSKeys("uvy-personal", "us", "lockboxkms")
	if err != nil {
		w.Write([]byte(fmt.Sprintf("failed to list keys: %v", err)))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<ul hx-trigger='load' hx-swap='outerHTML'>"))
	for _, key := range keys {
		w.Write([]byte(fmt.Sprintf("<li>%s</li>", key)))
	}
	w.Write([]byte("</ul>"))
}

func listKMSKeys(projectID, locationID, keyRingID string) (map[string]string, error) {
	// Create a context
	ctx := context.Background()

	// Create a KMS client
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client: %v", err)
	}
	defer client.Close()

	// Define the parent key ring path
	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", projectID, locationID, keyRingID)

	// Call the API to list keys
	req := &kmspb.ListCryptoKeysRequest{
		Parent: parent,
	}
	it := client.ListCryptoKeys(ctx, req)

	// Create a map to store the keys
	keys := make(map[string]string)

	// Iterate over the keys
	for {
		cryptoKey, err := it.Next()
		if err != nil {
			break
		}
		keys[cryptoKey.Name] = cryptoKey.Name[strings.LastIndex(cryptoKey.Name, "/")+1:]
	}

	return keys, nil
}
