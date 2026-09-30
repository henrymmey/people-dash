package main

import "testing"

func TestValidUser(t *testing.T){for _,u:=range []string{"alice","henry_1","web-user"}{if !validUser.MatchString(u){t.Fatalf("expected valid username %q",u)}};for _,u:=range []string{"Alice","../../x","a very long username that should fail because it is too long"}{if validUser.MatchString(u){t.Fatalf("expected invalid username %q",u)}}}
func TestWithin(t *testing.T){if !within("/home","/home/alice"){t.Fatal("expected child path")};if within("/home","/home2/alice"){t.Fatal("prefix escape")}}
func TestValidKey(t *testing.T){if !validKey.MatchString("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAexample laptop"){t.Fatal("expected key")}}
