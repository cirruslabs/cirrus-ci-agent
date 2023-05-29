package vaultunboxer_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/internal/executor/vaultunboxer"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNonBoxedValues(t *testing.T) {
	// Empty value
	_, err := vaultunboxer.NewBoxedValue("")
	require.ErrorIs(t, err, vaultunboxer.ErrNotABoxedValue)

	// Unterminated Vault-boxed value
	_, err = vaultunboxer.NewBoxedValue("VAULT[")
	require.ErrorIs(t, err, vaultunboxer.ErrNotABoxedValue)
}

func TestInvalidBoxedValues(t *testing.T) {
	// Empty value
	_, err := vaultunboxer.NewBoxedValue("VAULT[]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value with not enough arguments
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value with too much arguments
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path some.path extraneous]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value that contains a selector with empty elements
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path some.]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)
}

func TestSelectorInvalidCombinations(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"first_key": "first secret key value",
		},
	}

	trials := []struct {
		Name          string
		RawBoxedValue string
	}{
		{
			Name:          "querying elements in a scalar element",
			RawBoxedValue: "VAULT[secret/data/keys data.first_key.not_in_dict]",
		},
		{
			Name:          "querying a non-existent element",
			RawBoxedValue: "VAULT[secret/data/keys data.nonexistent]",
		},
		{
			Name:          "when querying terminating element is not a string",
			RawBoxedValue: "VAULT[secret/data/keys data]",
		},
	}

	for _, trial := range trials {
		t.Run(trial.Name, func(t *testing.T) {
			selector, err := vaultunboxer.NewBoxedValue(trial.RawBoxedValue)
			require.NoError(t, err)

			_, err = selector.Select(data)
			require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)
		})
	}
}

func TestSelector(t *testing.T) {
	const (
		firstSecretKeyValue  = "first secret key value"
		secondSecretKeyValue = "second secret key value"
	)

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"first_key": firstSecretKeyValue,
			"extra": map[string]interface{}{
				"second_key": secondSecretKeyValue,
			},
		},
	}

	trials := []struct {
		Name          string
		RawBoxedValue string
		Expected      string
	}{
		{
			Name:          "first key",
			RawBoxedValue: "VAULT[secret/data/keys data.first_key]",
			Expected:      firstSecretKeyValue,
		},
		{
			Name:          "second key",
			RawBoxedValue: "VAULT[secret/data/keys data.extra.second_key]",
			Expected:      secondSecretKeyValue,
		},
	}

	for _, trial := range trials {
		t.Run(trial.Name, func(t *testing.T) {
			selector, err := vaultunboxer.NewBoxedValue(trial.RawBoxedValue)
			require.NoError(t, err)

			result, err := selector.Select(data)
			require.NoError(t, err)
			require.Equal(t, trial.Expected, result)
		})
	}
}

func TestUseCache(t *testing.T) {
	valueThatDoesNotUseCache, err := vaultunboxer.NewBoxedValue("VAULT[path key]")
	require.NoError(t, err)
	require.False(t, valueThatDoesNotUseCache.UseCache())

	valueThatUsesCache, err := vaultunboxer.NewBoxedValue("VAULT_CACHED[path key]")
	require.NoError(t, err)
	require.True(t, valueThatUsesCache.UseCache())
}
