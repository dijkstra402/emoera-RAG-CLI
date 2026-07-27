package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestResolvePriorityFlagEnvironmentProfileDefault(t *testing.T) {
	t.Setenv("EMOERA_PROFILE", "environment")
	t.Setenv("EMOERA_ENDPOINT", "https://env.example.com/")
	t.Setenv("EMOERA_API_TOKEN", "env-token")
	t.Setenv("EMOERA_OUTPUT", "json")
	file := File{
		CurrentProfile: "file",
		Profiles: map[string]Profile{
			"file":        {Endpoint: "https://file.example.com", DefaultOutput: "table"},
			"environment": {Endpoint: "https://profile.example.com", DefaultOutput: "table"},
			"flag":        {Endpoint: "https://flag-profile.example.com", DefaultOutput: "table"},
		},
	}

	runtime, err := Resolve(file, "/tmp/config", Overrides{
		Profile: "flag", Endpoint: "https://flag.example.com/", Token: "flag-token",
		Output: "jsonl", Timeout: 7 * time.Second,
	}, "keychain-token")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Profile != "flag" || runtime.Endpoint != "https://flag.example.com" {
		t.Fatalf("flag values did not win: %#v", runtime)
	}
	if runtime.Token != "flag-token" || runtime.TokenSource != "flag" {
		t.Fatalf("flag token did not win: %#v", runtime)
	}
	if runtime.Output != "jsonl" || runtime.Timeout != 7*time.Second {
		t.Fatalf("flag output/timeout did not win: %#v", runtime)
	}
}

func TestResolveEnvironmentBeforeProfileAndKeychain(t *testing.T) {
	t.Setenv("EMOERA_ENDPOINT", "https://env.example.com")
	t.Setenv("EMOERA_API_TOKEN", "env-token")
	t.Setenv("EMOERA_OUTPUT", "json")
	file := DefaultFile()
	file.Profiles[DefaultProfile] = Profile{
		Endpoint: "https://profile.example.com", DefaultOutput: "table",
	}

	runtime, err := Resolve(file, "/tmp/config", Overrides{}, "keychain-token")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Endpoint != "https://env.example.com" || runtime.Output != "json" {
		t.Fatalf("environment values did not win: %#v", runtime)
	}
	if runtime.Token != "env-token" || runtime.TokenSource != "environment" {
		t.Fatalf("environment token did not win: %#v", runtime)
	}
}

func TestResolveProfileAndKeychainFallback(t *testing.T) {
	t.Setenv("EMOERA_ENDPOINT", "")
	t.Setenv("EMOERA_API_TOKEN", "")
	t.Setenv("EMOERA_OUTPUT", "")
	t.Setenv("EMOERA_PROFILE", "")
	file := DefaultFile()
	file.Profiles[DefaultProfile] = Profile{
		Endpoint: "https://profile.example.com", DefaultOutput: "json",
	}

	runtime, err := Resolve(file, "/tmp/config", Overrides{}, "keychain-token")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Endpoint != "https://profile.example.com" || runtime.Output != "json" {
		t.Fatalf("profile values were not used: %#v", runtime)
	}
	if runtime.Token != "keychain-token" || runtime.TokenSource != "keychain" {
		t.Fatalf("keychain token was not used: %#v", runtime)
	}
}

func TestSaveDoesNotPersistToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	file := DefaultFile()
	if err := Save(path, file); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentProfile != DefaultProfile || len(loaded.Profiles) != 1 {
		t.Fatalf("unexpected config round trip: %#v", loaded)
	}
}
