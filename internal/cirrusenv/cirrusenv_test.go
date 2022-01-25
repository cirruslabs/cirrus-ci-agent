package cirrusenv_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/internal/cirrusenv"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestCirrusEnvNormal(t *testing.T) {
	ce, err := cirrusenv.New(42)
	if err != nil {
		t.Fatal(err)
	}
	defer ce.Close()

	if err := ioutil.WriteFile(ce.Path(), []byte("A=B\nA==B"), 0600); err != nil {
		t.Fatal(err)
	}

	env, err := ce.Consume()
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"A": "=B",
	}

	assert.Equal(t, expected, env)
}
