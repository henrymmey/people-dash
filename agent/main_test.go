package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidUser(t *testing.T) {
	for _, username := range []string{"a", "alice", "henry1", "web-user", "a1-b2"} {
		if !validUser.MatchString(username) {
			t.Fatalf("expected valid %q", username)
		}
	}

	for _, username := range []string{
		"Alice",
		"a_b",
		"-alice",
		"alice-",
		"../../x",
		"a very long username that should fail",
	} {
		if validUser.MatchString(username) {
			t.Fatalf("expected invalid %q", username)
		}
	}
}

func TestWithin(t *testing.T) {
	if !within("/home", "/home/alice") {
		t.Fatal("child path rejected")
	}
	if within("/home", "/home2/alice") {
		t.Fatal("prefix escape accepted")
	}
}

func TestValidKey(t *testing.T) {
	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAexample laptop"
	if !validKey.MatchString(key) {
		t.Fatal("expected key")
	}
}

func TestSuspendedConfigRebuild(t *testing.T) {
	root := t.TempDir()
	suspended := filepath.Join(root, "suspended")
	if err := os.MkdirAll(suspended, 0755); err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(filepath.Join(suspended, "henry"), nil, 0600)
	_ = os.WriteFile(filepath.Join(suspended, "alice"), nil, 0600)

	cfg := Config{
		SuspendedRoot:      suspended,
		SuspendedSSHConfig: filepath.Join(root, "sshd.conf"),
	}
	server := &Server{cfg: cfg}

	if err := server.rebuildSSH(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(cfg.SuspendedSSHConfig)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "DenyUsers alice henry") {
		t.Fatalf("unexpected config: %q", string(data))
	}
}
