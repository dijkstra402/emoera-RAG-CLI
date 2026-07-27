package cmd

import (
	"testing"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

func TestMissingPositionalArgumentUsesArgumentsExitCode(t *testing.T) {
	command := newRootCommand(memoryStore{})
	command.SetArgs([]string{"doc", "show"})

	err := command.Execute()
	if apperr.ExitCode(err) != apperr.ExitArguments {
		t.Fatalf("expected arguments exit code, got %d: %v", apperr.ExitCode(err), err)
	}
}

func TestInvalidFlagUsesArgumentsExitCode(t *testing.T) {
	command := newRootCommand(memoryStore{})
	command.SetArgs([]string{"--not-a-real-flag"})

	err := command.Execute()
	if apperr.ExitCode(err) != apperr.ExitArguments {
		t.Fatalf("expected arguments exit code, got %d: %v", apperr.ExitCode(err), err)
	}
}

func TestResolvedVersionPrefersInjectedReleaseVersion(t *testing.T) {
	original := version
	version = "v1.2.3"
	t.Cleanup(func() { version = original })

	if got := resolvedVersion(); got != "1.2.3" {
		t.Fatalf("expected normalized release version, got %q", got)
	}
}

type memoryStore struct{}

func (memoryStore) Get(string) (string, error) { return "", nil }
func (memoryStore) Set(string, string) error   { return nil }
func (memoryStore) Delete(string) error        { return nil }
