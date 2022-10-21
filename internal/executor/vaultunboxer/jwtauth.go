package vaultunboxer

import (
	"context"
	vault "github.com/hashicorp/vault/api"
)

type JWTAuth struct {
	Token string
	Role  string
}

func (jwtAuth *JWTAuth) Login(ctx context.Context, client *vault.Client) (*vault.Secret, error) {
	data := map[string]interface{}{
		"jwt": jwtAuth.Token,
	}

	if jwtAuth.Role != "" {
		data["role"] = jwtAuth.Role
	}

	return client.Logical().WriteWithContext(ctx, "auth/jwt/login", data)
}
