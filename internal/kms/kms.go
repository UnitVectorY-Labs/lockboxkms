package kms

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	kmsclient "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/iterator"
)

// Client wraps the KMS KeyManagementClient
type Client struct {
	KMSClient *kmsclient.KeyManagementClient
	Config    Config
}

// Config holds KMS configuration
type Config struct {
	ProjectID string
	Location  string
	KeyRing   string
}

// NewClient initializes a new KMS client
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	client, err := kmsclient.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client: %w", err)
	}
	return &Client{
		KMSClient: client,
		Config:    cfg,
	}, nil
}

// Close closes the KMS client
func (c *Client) Close() error {
	return c.KMSClient.Close()
}

// ListKeys retrieves all symmetric encryption keys from the specified key ring
func (c *Client) ListKeys(ctx context.Context) (map[string]string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", c.Config.ProjectID, c.Config.Location, c.Config.KeyRing)
	it := c.KMSClient.ListCryptoKeys(ctx, &kmspb.ListCryptoKeysRequest{
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

// Encrypt encrypts the plaintext using the specified key
func (c *Client) Encrypt(ctx context.Context, keyName, plaintext string) (string, error) {
	req := &kmspb.EncryptRequest{
		Name:      keyName,
		Plaintext: []byte(plaintext),
	}

	resp, err := c.KMSClient.Encrypt(ctx, req)
	if err != nil {
		return "", err
	}

	encryptedText := base64.StdEncoding.EncodeToString(resp.Ciphertext)
	return encryptedText, nil
}
