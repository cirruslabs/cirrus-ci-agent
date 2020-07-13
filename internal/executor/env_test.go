package executor

import (
	"reflect"
	"testing"
)

func Test_DefaultValue(t *testing.T) {
	result := ExpandText("${CIRRUS_TAG:latest}", make(map[string]string))
	if result == "latest" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}

func Test_Simple(t *testing.T) {
	result := ExpandText("${CIRRUS_TAG:latest}", map[string]string{"CIRRUS_TAG": "foo"})
	if result == "foo" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}

func Test_Simple_Windows_Style(t *testing.T) {
	result := ExpandText("%CIRRUS_TAG%", map[string]string{"CIRRUS_TAG": "foo"})
	if result == "foo" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}

func Test_Environment(t *testing.T) {
	original := map[string]string{
		"GOPATH2":            "/root/go",
		"GOSRC2":             "$GOPATH2/src/github.com/some/thing",
		"CIRRUS_WORKING_DIR": "$GOSRC2",
		"SCRIPT_BASE":        "$GOSRC2/contrib/cirrus",
		"PACKER_BASE":        "${SCRIPT_BASE}/packer",
	}

	expected := map[string]string{
		"GOPATH2":            "/root/go",
		"GOSRC2":             "/root/go/src/github.com/some/thing",
		"CIRRUS_WORKING_DIR": "/root/go/src/github.com/some/thing",
		"SCRIPT_BASE":        "/root/go/src/github.com/some/thing/contrib/cirrus",
		"PACKER_BASE":        "/root/go/src/github.com/some/thing/contrib/cirrus/packer",
	}

	result := expandEnvironmentRecursively(original)

	if reflect.DeepEqual(result, expected) {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}
