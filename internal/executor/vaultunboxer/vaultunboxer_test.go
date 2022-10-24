//go:build linux
// +build linux

package vaultunboxer_test

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/vaultunboxer"
	"github.com/google/uuid"
	vault "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"testing"
)

func TestVault(t *testing.T) {
	ctx := context.Background()

	var vaultToken = uuid.New().String()

	// Create and start the HashiCorp's Vault container
	request := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "vault:latest",
			ExposedPorts: []string{"8200/tcp"},
			Env: map[string]string{
				"VAULT_DEV_ROOT_TOKEN_ID": vaultToken,
			},
		},
		Started: true,
	}
	container, err := testcontainers.GenericContainer(ctx, request)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	// Create demo data
	vaultURL, err := container.Endpoint(ctx, "http")
	require.NoError(t, err)

	client, err := vault.NewClient(vault.DefaultConfig())
	require.NoError(t, err)

	require.NoError(t, client.SetAddress(vaultURL))
	client.SetToken(vaultToken)

	const (
		secretKeyValue = "secret key value"
	)

	_, err = client.KVv2("secret").Put(ctx, "keys", map[string]interface{}{
		"admin": secretKeyValue,
	})
	require.NoError(t, err)

	// Unbox a Vault-boxed value
	selector, err := vaultunboxer.NewBoxedValue("VAULT[secret/data/keys data.admin]")
	require.NoError(t, err)

	secretValue, err := vaultunboxer.New(client).Unbox(ctx, selector)
	require.NoError(t, err)
	require.Equal(t, secretKeyValue, secretValue)
}
