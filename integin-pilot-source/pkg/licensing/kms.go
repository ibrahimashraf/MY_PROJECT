package licensing

import (
	"context"
	"errors"
	"fmt"

	vault "github.com/hashicorp/vault/api"
)

// OpenBaoKMSClient wraps the Vault/OpenBao API for L0 Root PKI key rotation and signing.
type OpenBaoKMSClient struct {
	client *vault.Client
}

// NewOpenBaoKMSClient initializes a client with the provided OpenBao/Vault address and token.
func NewOpenBaoKMSClient(address, token string) (*OpenBaoKMSClient, error) {
	if address == "" {
		return nil, errors.New("kms address cannot be empty")
	}
	cfg := vault.DefaultConfig()
	cfg.Address = address

	client, err := vault.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init openbao client: %w", err)
	}
	if token != "" {
		client.SetToken(token)
	}
	return &OpenBaoKMSClient{client: client}, nil
}

// Client returns the underlying Vault API client.
func (c *OpenBaoKMSClient) Client() *vault.Client {
	return c.client
}

// HealthCheck asserts connectivity to the OpenBao/Vault cluster.
func (c *OpenBaoKMSClient) HealthCheck(ctx context.Context) (bool, error) {
	health, err := c.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return false, err
	}
	return health.Initialized && !health.Sealed, nil
}
