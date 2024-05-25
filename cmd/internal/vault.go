package internal

import (
	"fmt"
	"os"

	"github.com/sanLimbu/todo-api/internal/envar/vault"
)

//NewVaultProvider instantiate the Vault client using configuration defined in environment variables.
func NewVaultProvider() (*vault.Provider, error) {
	vaultPath := os.Getenv("VAULT_PATH")
	vaultToken := os.Getenv("VAULT_TOKEN")
	vaultAddress := os.Getenv("VAULT_ADDRESS")

	provider, err := vault.New(vaultToken, vaultAddress, vaultPath)
	if err != nil {
		return nil, fmt.Errorf("vault.New %w", err)
	}

	return provider, nil
}
