package internal

import (
	"os"

	"github.com/sanLimbu/todo-api/internal"
	"github.com/sanLimbu/todo-api/internal/envar/vault"
)

//NewVaultProvider instantiate the Vault client using configuration defined in environment variables.
func NewVaultProvider() (*vault.Provider, error) {
	vaultPath := os.Getenv("VAULT_PATH")
	vaultToken := os.Getenv("VAULT_TOKEN")
	vaultAddress := os.Getenv("VAULT_ADDRESS")

	provider, err := vault.New(vaultToken, vaultAddress, vaultPath)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "vault.New ")

	}

	return provider, nil
}
