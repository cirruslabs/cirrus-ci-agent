package vaultunboxer

import (
	"context"
	"fmt"
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
	cache  map[string]*CachedSecret
}

type CachedSecret struct {
	Secret *vault.Secret
	Err    error
}

func New(client *vault.Client) *VaultUnboxer {
	return &VaultUnboxer{
		client: client,
		cache:  map[string]*CachedSecret{},
	}
}

func NewFromEnvironment(ctx context.Context, env *environment.Environment) (*VaultUnboxer, error) {
	config := vault.DefaultConfig()

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

func (unboxer *VaultUnboxer) Unbox(ctx context.Context, value *BoxedValue) (string, error) {
	secret, err := unboxer.retrieveSecret(ctx, value)
	if err != nil {
		return "", err
	}

	if secret == nil {
		return "", fmt.Errorf("associated Vault secret doesn't exist")
	}

	if secret.Data == nil {
		return "", fmt.Errorf("associated Vault secret contains no data")
	}

	return value.Select(secret.Data)
}

func (unboxer *VaultUnboxer) retrieveSecret(ctx context.Context, value *BoxedValue) (*vault.Secret, error) {
	if value.UseCache() {
		// Try the cache first, and fall back to poking the Vault
		// if no entry exists in the cache
		cachedSecret, ok := unboxer.cache[value.vaultPath]
		if ok {
			return cachedSecret.Secret, cachedSecret.Err
		}
	}

	secret, err := unboxer.client.Logical().ReadWithContext(ctx, value.vaultPath)

	// Cache the result, even a negative one (with err != nil)
	unboxer.cache[value.vaultPath] = &CachedSecret{
		Secret: secret,
		Err:    err,
	}

	return secret, err
}
