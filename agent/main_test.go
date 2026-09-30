package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidUser(t *testing.T) {
	for _, u := range []string{"a", "alice", "henry1", "web-user", "a1-b2"} {
		if !validUser.MatchString(u) { t.Fatalf("expected valid %q", u) }
	}
	for _, u := range []string{"Alice", "a_b", "-alice", "alice-", "../../x", "a very long username that should fail"} {
		if validUser.MatchString(u) { t.Fatalf("expected invalid %q", u) }
	}
}
func TestWithin(t *testing.T) { if !within("/home", "/home/alice") { t.Fatal("child path rejected") }; if within("/home", "/home2/alice") { t.Fatal("prefix escape accepted") } }
func TestValidKey(t *testing.T) { if !validKey.MatchString("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAexample laptop") { t.Fatal("expected key") } }
func TestSuspendedConfigRebuild(t *testing.T) {
	root:=t.TempDir(); susp:=filepath.Join(root,"suspended"); if err:=os.MkdirAll(susp,0755);err!=nil{t.Fatal(err)}
	_ = os.WriteFile(filepath.Join(susp,"henry"),nil,0600); _ = os.WriteFile(filepath.Join(susp,"alice"),nil,0600)
	cfg:=Config{SuspendedRoot:susp,SuspendedSSHConfig:filepath.Join(root,"sshd.conf")}; s:=&Server{cfg:cfg}
	if err:=s.rebuildSSH();err!=nil{t.Fatal(err)};data,err:=os.ReadFile(cfg.SuspendedSSHConfig);if err!=nil{t.Fatal(err)};if !strings.Contains(string(data),"DenyUsers alice henry"){t.Fatalf("unexpected config: %q",string(data))}
}
