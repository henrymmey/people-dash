package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Listen, Token, HomeRoot, PublicDomain, SSHHost string
	StateRoot, SuspendedRoot, SuspendedSSHConfig   string
	DefaultQuotaBytes                              int64
}

type Server struct {
	cfg Config
	mux *http.ServeMux
	mu  sync.Mutex
}

type userRequest struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
}

type keyRequest struct {
	PublicKey string `json:"public_key"`
}

var (
	validUser = regexp.MustCompile(`^(?:[a-z]|[a-z][a-z0-9-]{0,30}[a-z0-9])$`)
	validKey  = regexp.MustCompile(`^(ssh-(ed25519|rsa)|ecdsa-sha2-nistp256|sk-(ssh-ed25519|ecdsa-sha2-nistp256)@openssh\.com)\s+[^\s]+(?:\s+.*)?$`)
)

func main() {
	cfg := Config{
		Listen:             env("LISTEN", "192.168.176.118:8080"),
		Token:              mustEnv("PEOPLE_AGENT_TOKEN"),
		HomeRoot:           env("HOME_ROOT", "/home"),
		PublicDomain:       env("PUBLIC_DOMAIN", "p.meyerbrief.de"),
		SSHHost:            env("SSH_HOST", "ssh.p.meyerbrief.de"),
		StateRoot:          env("STATE_ROOT", "/var/lib/people-agent/users"),
		SuspendedRoot:      env("SUSPENDED_ROOT", "/var/lib/people-agent/suspended"),
		SuspendedSSHConfig: env("SUSPENDED_SSH_CONFIG", "/etc/ssh/sshd_config.d/90-people-suspended.conf"),
		DefaultQuotaBytes:  envInt64("DEFAULT_QUOTA_BYTES", 0),
	}

	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	if err := s.ensureRuntime(); err != nil {
		log.Fatal(err)
	}
	s.routes()

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           s.auth(s.mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("people-agent listening on %s", cfg.Listen)
	log.Fatal(srv.ListenAndServe())
}

func (s *Server) ensureRuntime() error {
	for _, path := range []string{s.cfg.StateRoot, s.cfg.SuspendedRoot} {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}
	}

	for _, command := range []string{
		"getent",
		"groupadd",
		"useradd",
		"userdel",
		"setfacl",
		"sshd",
		"systemctl",
		"pkill",
		"loginctl",
		"du",
	} {
		if _, err := exec.LookPath(command); err != nil {
			return fmt.Errorf("%s is required: %w", command, err)
		}
	}

	if err := ensureGroup("people"); err != nil {
		return err
	}

	return s.rebuildSSH()
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	s.mux.HandleFunc("POST /v1/users", s.createUser)
	s.mux.HandleFunc("DELETE /v1/users/{username}", s.deleteUser)
	s.mux.HandleFunc("POST /v1/users/{username}/suspend", s.suspendUser)
	s.mux.HandleFunc("DELETE /v1/users/{username}/suspend", s.resumeUser)
	s.mux.HandleFunc("GET /v1/users/{username}/status", s.status)
	s.mux.HandleFunc("POST /v1/users/{username}/ssh-keys", s.addKey)
	s.mux.HandleFunc("DELETE /v1/users/{username}/ssh-keys", s.removeKey)
	s.mux.HandleFunc("GET /v1/users/{username}/storage", s.storage)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		expected := "Bearer " + s.cfg.Token
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var in userRequest
	if !decodeJSON(r, &in) || !validUser.MatchString(in.Username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid username"})
		return
	}

	if s.managed(in.Username) {
		writeJSON(w, http.StatusOK, s.userResponse(in.Username))
		return
	}

	if out, err := exec.Command("getent", "passwd", in.Username).Output(); err == nil && len(out) > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "linux user exists but is not People-managed",
		})
		return
	}

	home := filepath.Join(s.cfg.HomeRoot, in.Username)
	if !within(s.cfg.HomeRoot, home) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
		return
	}

	if _, err := os.Stat(home); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "home directory already exists"})
		return
	}

	if err := run("useradd", "--create-home", "--user-group", "--groups", "people", "--home-dir", home, "--shell", "/bin/bash", in.Username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rollback := func() {
		_ = run("userdel", "--remove", in.Username)
	}

	directories := []struct {
		path string
		mode os.FileMode
	}{
		{path: filepath.Join(home, "public_html"), mode: 0755},
		{path: filepath.Join(home, ".ssh"), mode: 0700},
	}

	for _, directory := range directories {
		if err := os.MkdirAll(directory.path, directory.mode); err != nil {
			rollback()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	if err := os.Chmod(home, 0700); err != nil {
		rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := run("setfacl", "-m", "u:www-data:--x", home); err != nil {
		rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := run("chown", "-R", in.Username+":"+in.Username, home); err != nil {
		rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := run("chmod", "0755", filepath.Join(home, "public_html")); err != nil {
		rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := run("chmod", "0700", filepath.Join(home, ".ssh")); err != nil {
		rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := os.WriteFile(filepath.Join(s.cfg.StateRoot, in.Username), nil, 0600); err != nil {
		rollback()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, s.userResponse(in.Username))
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username := r.PathValue("username")
	if !validUser.MatchString(username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid username"})
		return
	}
	if !s.managed(username) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user is not People-managed"})
		return
	}

	if err := s.setSuspendedLocked(username, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := run("userdel", "--remove", username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	_ = os.Remove(filepath.Join(s.cfg.StateRoot, username))
	_ = os.Remove(filepath.Join(s.cfg.SuspendedRoot, username))

	if err := s.rebuildSSH(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) suspendUser(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username := r.PathValue("username")
	if !validUser.MatchString(username) || !s.managed(username) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	if err := s.setSuspendedLocked(username, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"username": username,
		"status":   "suspended",
	})
}

func (s *Server) resumeUser(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username := r.PathValue("username")
	if !validUser.MatchString(username) || !s.managed(username) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	if err := s.setSuspendedLocked(username, false); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"username": username,
		"status":   "active",
	})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if !validUser.MatchString(username) || !s.managed(username) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"username": username,
		"status":   s.statusOf(username),
	})
}

func (s *Server) setSuspendedLocked(username string, suspended bool) error {
	if !s.managed(username) {
		return fmt.Errorf("user %q is not People-managed", username)
	}

	marker := filepath.Join(s.cfg.SuspendedRoot, username)
	old := s.statusOf(username) == "suspended"
	if old == suspended {
		return nil
	}

	if suspended {
		if err := os.WriteFile(marker, nil, 0600); err != nil {
			return fmt.Errorf("create suspension marker: %w", err)
		}
	} else if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove suspension marker: %w", err)
	}

	rollback := func() {
		if old {
			_ = os.WriteFile(marker, nil, 0600)
		} else {
			_ = os.Remove(marker)
		}
		_ = s.rebuildSSH()
		_ = run("sshd", "-t")
		_ = run("systemctl", "reload", "ssh")
	}

	if err := s.rebuildSSH(); err != nil {
		rollback()
		return err
	}

	if err := run("sshd", "-t"); err != nil {
		rollback()
		return fmt.Errorf("sshd validation failed: %w", err)
	}

	if err := run("systemctl", "reload", "ssh"); err != nil {
		rollback()
		return fmt.Errorf("ssh reload failed: %w", err)
	}

	if suspended {
		if err := terminateUser(username); err != nil {
			rollback()
			return fmt.Errorf("terminate user sessions failed: %w", err)
		}
	}

	return nil
}

func (s *Server) rebuildSSH() error {
	entries, err := os.ReadDir(s.cfg.SuspendedRoot)
	if err != nil {
		return fmt.Errorf("read suspension state: %w", err)
	}

	var users []string
	for _, entry := range entries {
		if !entry.IsDir() && validUser.MatchString(entry.Name()) {
			users = append(users, entry.Name())
		}
	}
	sort.Strings(users)

	content := "# Managed by people-agent. Do not edit manually.\n"
	if len(users) == 0 {
		content += "# No suspended People users.\n"
	} else {
		content += "DenyUsers " + strings.Join(users, " ") + "\n"
	}

	if err := os.MkdirAll(filepath.Dir(s.cfg.SuspendedSSHConfig), 0755); err != nil {
		return fmt.Errorf("create ssh config directory: %w", err)
	}

	tmp := s.cfg.SuspendedSSHConfig + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return fmt.Errorf("write ssh config: %w", err)
	}

	if err := os.Rename(tmp, s.cfg.SuspendedSSHConfig); err != nil {
		return fmt.Errorf("replace ssh config: %w", err)
	}

	return nil
}

func (s *Server) addKey(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if !validUser.MatchString(username) || !s.managed(username) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	var in keyRequest
	if !decodeJSON(r, &in) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	key := strings.TrimSpace(in.PublicKey)
	if !validKey.MatchString(key) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid public key"})
		return
	}

	path := filepath.Join(s.cfg.HomeRoot, username, ".ssh", "authorized_keys")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == key {
			writeJSON(w, http.StatusOK, map[string]string{"status": "already_present"})
			return
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer f.Close()

	if _, err := f.WriteString(key + "\n"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if err := run("chown", username+":"+username, path); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "added"})
}

func (s *Server) removeKey(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if !validUser.MatchString(username) || !s.managed(username) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	var in keyRequest
	if !decodeJSON(r, &in) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	key := strings.TrimSpace(in.PublicKey)
	if !validKey.MatchString(key) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid public key"})
		return
	}

	path := filepath.Join(s.cfg.HomeRoot, username, ".ssh", "authorized_keys")
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "authorized_keys not found"})
		return
	}

	var kept []string
	removed := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == key {
			removed = true
			continue
		}
		if strings.TrimSpace(line) != "" {
			kept = append(kept, line)
		}
	}

	if !removed {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
		return
	}

	content := strings.Join(kept, "\n")
	if content != "" {
		content += "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (s *Server) storage(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if !validUser.MatchString(username) || !s.managed(username) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	output, err := exec.Command("du", "-sb", filepath.Join(s.cfg.HomeRoot, username)).Output()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	fields := strings.Fields(string(output))
	if len(fields) < 1 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid du output"})
		return
	}

	used, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	response := map[string]int64{"used_bytes": used}
	if s.cfg.DefaultQuotaBytes > 0 {
		response["quota_bytes"] = s.cfg.DefaultQuotaBytes
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) managed(username string) bool {
	_, err := os.Stat(filepath.Join(s.cfg.StateRoot, username))
	return err == nil
}

func (s *Server) statusOf(username string) string {
	if _, err := os.Stat(filepath.Join(s.cfg.SuspendedRoot, username)); err == nil {
		return "suspended"
	}
	return "active"
}

func (s *Server) userResponse(username string) map[string]any {
	return map[string]any{
		"username": username,
		"domain":   username + "." + s.cfg.PublicDomain,
		"ssh":      "ssh " + username + "@" + s.cfg.SSHHost,
		"status":   s.statusOf(username),
	}
}

func terminateUser(username string) error {
	if err := run("loginctl", "terminate-user", username); err == nil {
		return nil
	}

	return run("pkill", "-KILL", "-u", username)
}

func ensureGroup(name string) error {
	if out, err := exec.Command("getent", "group", name).Output(); err == nil && len(out) > 0 {
		return nil
	}
	return run("groupadd", "--system", name)
}

func decodeJSON(r *http.Request, value any) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		return false
	}
	return json.Unmarshal(body, value) == nil
}

func run(name string, args ...string) error {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s: %w", name, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func within(root, path string) bool {
	rootAbs, rootErr := filepath.Abs(root)
	pathAbs, pathErr := filepath.Abs(path)
	if rootErr != nil || pathErr != nil {
		return false
	}
	return pathAbs == rootAbs || strings.HasPrefix(pathAbs, rootAbs+string(os.PathSeparator))
}

func env(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustEnv(key string) string {
	if value := os.Getenv(key); value == "" {
		log.Fatal(errors.New(key + " is required"))
		return ""
	} else {
		return value
	}
}

func envInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
