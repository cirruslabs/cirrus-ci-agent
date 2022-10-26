package vaultunboxer

import (
	"context"
	"fmt"
	"net/http"

	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-ci-agent/internal/environment"
	vault "github.com/hashicorp/vault/api"
)

const (
	EnvCirrusVaultURL       = "CIRRUS_VAULT_URL"
	EnvCirrusVaultAuthPath  = "CIRRUS_VAULT_AUTH_PATH"
	EnvCirrusVaultNamespace = "CIRRUS_VAULT_NAMESPACE"
	EnvCirrusVaultRole      = "CIRRUS_VAULT_ROLE"
)

type VaultUnboxer struct {
	client *vault.Client
}

func New(client *vault.Client) *VaultUnboxer {
	return &VaultUnboxer{
		client: client,
	}
}

func NewFromEnvironment(ctx context.Context, env *environment.Environment) (*VaultUnboxer, error) {
	config := vault.DefaultConfig()

	tlsConfig := config.HttpClient.Transport.(*http.Transport).TLSClientConfig
	pool, _ := gocertifi.CACerts()
	tlsConfig.RootCAs = pool

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, err
	}

	url, ok := env.Lookup(EnvCirrusVaultURL)
	if !ok {
		return nil, fmt.Errorf("found Vault-protected environment variables, "+
			"but no %s variable was provided", EnvCirrusVaultURL)
	}

	if err := client.SetAddress(url); err != nil {
		return nil, err
	}

	if namespace, ok := env.Lookup(EnvCirrusVaultNamespace); ok {
		client.SetNamespace(namespace)
	}

	if jwtToken, ok := env.Lookup("CIRRUS_OIDC_TOKEN"); ok {
		auth := &JWTAuth{
			Token: jwtToken,
			Role:  env.Get(EnvCirrusVaultRole),
			Path:  env.Get(EnvCirrusVaultAuthPath),
		}

		_, err := client.Auth().Login(ctx, auth)
		if err != nil {
			return nil, err
		}
	}

	return New(client), nil
}

func (unboxer *VaultUnboxer) Unbox(ctx context.Context, selector *BoxedValue) (string, error) {
	secret, err := unboxer.client.Logical().ReadWithContext(ctx, selector.vaultPath)
	if err != nil {
		return "", err
	}

	return selector.Select(secret.Data)
}
