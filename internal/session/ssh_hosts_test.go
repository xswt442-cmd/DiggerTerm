package session

import (
	"reflect"
	"strings"
	"testing"
)

func TestSSHHostAliasParsing(t *testing.T) {
	content := []byte(`# Comments only
Host prod dev-box
  HostName internal.example.com
  IdentityFile ~/.ssh/private_key
Host *.example.com !secrets
Host=staging
host TestServer
Host prod
Host "-oProxyCommand=id"
Host alice@machine
Host foo_bar build-01
Include ~/.ssh/config.d/*
Host different # no more aliases
`)
	got := parseSSHHostAliases(content)
	want := []string{"build-01", "dev-box", "different", "foo_bar", "prod", "staging", "TestServer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("aliases=%q, want=%q", got, want)
	}
	for _, name := range got {
		if strings.Contains(strings.ToLower(name), "private_key") {
			t.Fatal("sensitive SSH config directive leaked")
		}
	}
}

func TestSSHHostSuggestionsLimited(t *testing.T) {
	content := []byte(strings.Repeat("Host prod\n", 1000))
	result := parseSSHHostAliases(content)
	if len(result) != 1 || result[0] != "prod" {
		t.Fatalf("duplicate host suggestions: %#v", result)
	}
}
