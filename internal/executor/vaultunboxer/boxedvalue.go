package vaultunboxer

import (
	"errors"
	"fmt"
	"strings"
)

type BoxedValue struct {
	vaultPath string
	dataPath  []string
}

const (
	prefix = "VAULT["
	suffix = "]"
)

var (
	ErrNotABoxedValue    = errors.New("doesn't look like a Vault-boxed value")
	ErrInvalidBoxedValue = errors.New("Vault-boxed value has an invalid format")
)

func NewBoxedValue(rawBoxedValue string) (*BoxedValue, error) {
	if !strings.HasPrefix(rawBoxedValue, prefix) || !strings.HasSuffix(rawBoxedValue, suffix) {
		return nil, ErrNotABoxedValue
	}

	rawBoxedValue = strings.TrimPrefix(rawBoxedValue, prefix)
	rawBoxedValue = strings.TrimSuffix(rawBoxedValue, suffix)

	parts := strings.Split(rawBoxedValue, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: there should be 2 parameters (path and a selector), found %d",
			ErrInvalidBoxedValue, len(parts))
	}

	dataPath := strings.Split(parts[1], ".")
	if len(dataPath) == 0 {
		return nil, fmt.Errorf("%w: selector should at least one", ErrInvalidBoxedValue)
	}

	for _, element := range dataPath {
		if element == "" {
			return nil, fmt.Errorf("%w: found an empty selector element ", ErrInvalidBoxedValue)
		}
	}

	return &BoxedValue{
		vaultPath: parts[0],
		dataPath:  dataPath,
	}, nil
}

func (selector *BoxedValue) Select(data interface{}) (string, error) {
	for _, element := range selector.dataPath {
		dataAsMap, ok := data.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("%w: selector's element %q should always "+
				"query in a dictionary/map-like structures", ErrInvalidBoxedValue, element)
		}

		newData, ok := dataAsMap[element]
		if !ok {
			return "", fmt.Errorf("%w: selector's element %q not found in a dictionary/map-like structure",
				ErrInvalidBoxedValue, element)
		}

		data = newData
	}

	s, ok := data.(string)
	if !ok {
		return "", fmt.Errorf("%w: selector's element %q should point to a string",
			ErrInvalidBoxedValue, selector.dataPath[len(selector.dataPath)-1])
	}

	return s, nil
}
