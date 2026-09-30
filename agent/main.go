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

type Server struct { cfg Config; mux *http.ServeMux; mu sync.Mutex }

type userRequest struct { Username string `json:"username"`; Email string `json:"email,omitempty"` }
type keyRequest struct { PublicKey string `json:"public_key"` }

var validUser = regexp.MustCompile(`^(?:[a-z]|[a-z][a-z0-9-]{0,30}[a-z0-9])$`)
var validKey = regexp.MustCompile(`^(ssh-(ed25519|rsa)|ecdsa-sha2-nistp256|sk-(ssh-ed25519|ecdsa-sha2-nistp256)@openssh\.com)\s+[^\s]+(?:\s+.*)?$`)

func main() {
	cfg := Config{
		Listen: env("LISTEN", "192.168.176.118:8080"), Token: mustEnv("PEOPLE_AGENT_TOKEN"),
		HomeRoot: env("HOME_ROOT", "/home"), PublicDomain: env("PUBLIC_DOMAIN", "p.meyerbrief.de"), SSHHost: env("SSH_HOST", "ssh.p.meyerbrief.de"),
		StateRoot: env("STATE_ROOT", "/var/lib/people-agent/users"), SuspendedRoot: env("SUSPENDED_ROOT", "/var/lib/people-agent/suspended"),
		SuspendedSSHConfig: env("SUSPENDED_SSH_CONFIG", "/etc/ssh/sshd_config.d/90-people-suspended.conf"), DefaultQuotaBytes: envInt64("DEFAULT_QUOTA_BYTES", 0),
	}
	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	if err := s.ensureRuntime(); err != nil { log.Fatal(err) }
	s.routes()
	srv := &http.Server{Addr: cfg.Listen, Handler: s.auth(s.mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("people-agent listening on %s", cfg.Listen)
	log.Fatal(srv.ListenAndServe())
}

func (s *Server) ensureRuntime() error {
	for _, p := range []string{s.cfg.StateRoot, s.cfg.SuspendedRoot} { if err := os.MkdirAll(p, 0755); err != nil { return fmt.Errorf("mkdir %s: %w", p, err) } }
	for _, cmd := range []string{"getent", "groupadd", "useradd", "userdel", "setfacl", "sshd", "systemctl", "pkill", "loginctl"} { if _, err := exec.LookPath(cmd); err != nil { return fmt.Errorf("%s is required: %w", cmd, err) } }
	if err := ensureGroup("people"); err != nil { return err }
	return s.rebuildSSH()
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	s.mux.HandleFunc("POST /v1/users", s.createUser)
	s.mux.HandleFunc("DELETE /v1/users/{username}", s.deleteUser)
	s.mux.HandleFunc("POST /v1/users/{username}/suspend", s.suspendUser)
	s.mux.HandleFunc("DELETE /v1/users/{username}/suspend", s.resumeUser)
	s.mux.HandleFunc("GET /v1/users/{username}/status", s.status)
	s.mux.HandleFunc("POST /v1/users/{username}/ssh-keys", s.addKey)
	s.mux.HandleFunc("DELETE /v1/users/{username}/ssh-keys", s.removeKey)
	s.mux.HandleFunc("GET /v1/users/{username}/storage", s.storage)
}

func (s *Server) auth(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { if r.URL.Path == "/health" { next.ServeHTTP(w, r); return }; if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.cfg.Token)) != 1 { write(w, 401, map[string]string{"error":"unauthorized"}); return }; next.ServeHTTP(w, r) }) }

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock(); defer s.mu.Unlock()
	var in userRequest
	if !decode(r, &in) || !validUser.MatchString(in.Username) { write(w, 400, map[string]string{"error":"invalid username"}); return }
	if s.managed(in.Username) { write(w, 200, s.userResponse(in.Username)); return }
	if out, err := exec.Command("getent", "passwd", in.Username).Output(); err == nil && len(out) > 0 { write(w,409,map[string]string{"error":"linux user exists but is not People-managed"}); return }
	home := filepath.Join(s.cfg.HomeRoot, in.Username); if !within(s.cfg.HomeRoot, home) { write(w,400,map[string]string{"error":"invalid path"}); return }
	if _, err := os.Stat(home); err == nil { write(w,409,map[string]string{"error":"home directory already exists"}); return }
	if err := run("useradd","--create-home","--user-group","--groups","people","--home-dir",home,"--shell","/bin/bash",in.Username); err != nil { write(w,500,map[string]string{"error":err.Error()}); return }
	rollback := func(){ _ = run("userdel","--remove",in.Username) }
	for _, p := range []struct{path string; mode os.FileMode}{{filepath.Join(home,"public_html"),0755},{filepath.Join(home,".ssh"),0700}} { if err:=os.MkdirAll(p.path,p.mode); err!=nil { rollback(); returnErr(w,err); return } }
	if err:=os.Chmod(home,0700); err!=nil { rollback(); returnErr(w,err); return }
	if err:=run("setfacl","-m","u:www-data:--x",home); err!=nil { rollback(); returnErr(w,err); return }
	if err:=run("chown","-R",in.Username+":"+in.Username,home); err!=nil { rollback(); returnErr(w,err); return }
	if err:=run("chmod","0755",filepath.Join(home,"public_html")); err!=nil { rollback(); returnErr(w,err); return }
	if err:=run("chmod","0700",filepath.Join(home,".ssh")); err!=nil { rollback(); returnErr(w,err); return }
	if err:=os.WriteFile(filepath.Join(s.cfg.StateRoot,in.Username),nil,0600); err!=nil { rollback(); returnErr(w,err); return }
	write(w,201,s.userResponse(in.Username))
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock(); defer s.mu.Unlock(); u:=r.PathValue("username")
	if !validUser.MatchString(u) { write(w,400,map[string]string{"error":"invalid username"}); return }
	if !s.managed(u) { write(w,404,map[string]string{"error":"user is not People-managed"}); return }
	if err:=s.setSuspendedLocked(u,true); err!=nil { write(w,500,map[string]string{"error":err.Error()}); return }
	if err:=run("userdel","--remove",u); err!=nil { write(w,500,map[string]string{"error":err.Error()}); return }
	_ = os.Remove(filepath.Join(s.cfg.StateRoot,u)); _ = os.Remove(filepath.Join(s.cfg.SuspendedRoot,u)); if err:=s.rebuildSSH(); err!=nil { write(w,500,map[string]string{"error":err.Error()}); return }
	write(w,200,map[string]string{"status":"deleted"})
}

func (s *Server) suspendUser(w http.ResponseWriter, r *http.Request) { s.mu.Lock(); defer s.mu.Unlock(); u:=r.PathValue("username"); if !validUser.MatchString(u)||!s.managed(u){write(w,404,map[string]string{"error":"user not found"});return}; if err:=s.setSuspendedLocked(u,true);err!=nil{write(w,500,map[string]string{"error":err.Error()});return}; write(w,200,map[string]string{"username":u,"status":"suspended"}) }
func (s *Server) resumeUser(w http.ResponseWriter, r *http.Request) { s.mu.Lock(); defer s.mu.Unlock(); u:=r.PathValue("username"); if !validUser.MatchString(u)||!s.managed(u){write(w,404,map[string]string{"error":"user not found"});return}; if err:=s.setSuspendedLocked(u,false);err!=nil{write(w,500,map[string]string{"error":err.Error()});return}; write(w,200,map[string]string{"username":u,"status":"active"}) }
func (s *Server) status(w http.ResponseWriter, r *http.Request) { u:=r.PathValue("username"); if !validUser.MatchString(u)||!s.managed(u){write(w,404,map[string]string{"error":"user not found"});return}; write(w,200,map[string]string{"username":u,"status":s.statusOf(u)}) }

func (s *Server) setSuspendedLocked(username string, suspended bool) error {
	if !s.managed(username) { return fmt.Errorf("user %q is not People-managed", username) }
	marker:=filepath.Join(s.cfg.SuspendedRoot,username); old:=s.statusOf(username)=="suspended"; if old==suspended{return nil}
	if suspended { if err:=os.WriteFile(marker,nil,0600);err!=nil{return fmt.Errorf("create suspension marker: %w",err)} } else if err:=os.Remove(marker);err!=nil&&!os.IsNotExist(err){return fmt.Errorf("remove suspension marker: %w",err)}
	rollback:=func(){if old{_ = os.WriteFile(marker,nil,0600)}else{_ = os.Remove(marker)}; _=s.rebuildSSH(); _=run("sshd","-t"); _=run("systemctl","reload","ssh")}
	if err:=s.rebuildSSH();err!=nil{rollback();return err}; if err:=run("sshd","-t");err!=nil{rollback();return fmt.Errorf("sshd validation failed: %w",err)}; if err:=run("systemctl","reload","ssh");err!=nil{rollback();return fmt.Errorf("ssh reload failed: %w",err)}
	if suspended { if err:=terminateUser(username);err!=nil{rollback();return fmt.Errorf("terminate user sessions failed: %w",err)} }
	return nil
}

func (s *Server) rebuildSSH() error {
	entries,err:=os.ReadDir(s.cfg.SuspendedRoot);if err!=nil{return fmt.Errorf("read suspension state: %w",err)};var users []string;for _,e:=range entries{if !e.IsDir()&&validUser.MatchString(e.Name()){users=append(users,e.Name())}};sortStrings(users)
	content:="# Managed by people-agent. Do not edit manually.\n";if len(users)==0{content+="# No suspended People users.\n"}else{content+="DenyUsers "+strings.Join(users," ")+"\n"}
	tmp:=s.cfg.SuspendedSSHConfig+".tmp";if err:=os.WriteFile(tmp,[]byte(content),0644);err!=nil{return fmt.Errorf("write ssh config: %w",err)};return os.Rename(tmp,s.cfg.SuspendedSSHConfig)
}

func (s *Server) addKey(w http.ResponseWriter, r *http.Request) { u:=r.PathValue("username");if !validUser.MatchString(u)||!s.managed(u){write(w,404,map[string]string{"error":"user not found"});return};var in keyRequest;if !decode(r,&in){write(w,400,map[string]string{"error":"invalid JSON"});return};key:=strings.TrimSpace(in.PublicKey);if !validKey.MatchString(key){write(w,400,map[string]string{"error":"invalid public key"});return};path:=filepath.Join(s.cfg.HomeRoot,u,".ssh","authorized_keys");data,err:=os.ReadFile(path);if err!=nil&&!os.IsNotExist(err){returnErr(w,err);return};for _,line:=range strings.Split(string(data),"\n"){if strings.TrimSpace(line)==key{write(w,200,map[string]string{"status":"already_present"});return}};f,err:=os.OpenFile(path,os.O_CREATE|os.O_APPEND|os.O_WRONLY,0600);if err!=nil{returnErr(w,err);return};defer f.Close();if _,err:=f.WriteString(key+"\n");err!=nil{returnErr(w,err);return};if err:=run("chown",u+":"+u,path);err!=nil{returnErr(w,err);return};write(w,201,map[string]string{"status":"added"}) }
func (s *Server) removeKey(w http.ResponseWriter, r *http.Request) { u:=r.PathValue("username");if !validUser.MatchString(u)||!s.managed(u){write(w,404,map[string]string{"error":"user not found"});return};var in keyRequest;if !decode(r,&in){write(w,400,map[string]string{"error":"invalid JSON"});return};want:=strings.TrimSpace(in.PublicKey);if !validKey.MatchString(want){write(w,400,map[string]string{"error":"invalid public key"});return};path:=filepath.Join(s.cfg.HomeRoot,u,".ssh","authorized_keys");data,err:=os.ReadFile(path);if err!=nil{write(w,404,map[string]string{"error":"authorized_keys not found"});return};var out []string;removed:=false;for _,line:=range strings.Split(string(data),"\n"){if strings.TrimSpace(line)==want{removed=true;continue};if strings.TrimSpace(line)!=""{out=append(out,line)}};if !removed{write(w,404,map[string]string{"error":"key not found"});return};content:=strings.Join(out,"\n");if content!=""{content+="\n"};if err:=os.WriteFile(path,[]byte(content),0600);err!=nil{returnErr(w,err);return};write(w,200,map[string]string{"status":"removed"}) }
func (s *Server) storage(w http.ResponseWriter, r *http.Request) { u:=r.PathValue("username");if !validUser.MatchString(u)||!s.managed(u){write(w,404,map[string]string{"error":"user not found"});return};b,err:=exec.Command("du","-sb",filepath.Join(s.cfg.HomeRoot,u)).Output();if err!=nil{returnErr(w,err);return};f:=strings.Fields(string(b));if len(f)<1{write(w,500,map[string]string{"error":"invalid du output"});return};used,err:=strconv.ParseInt(f[0],10,64);if err!=nil{returnErr(w,err);return};resp:=map[string]int64{"used_bytes":used};if s.cfg.DefaultQuotaBytes>0{resp["quota_bytes"]=s.cfg.DefaultQuotaBytes};write(w,200,resp) }

func (s *Server) managed(u string) bool { _,err:=os.Stat(filepath.Join(s.cfg.StateRoot,u));return err==nil }
func (s *Server) statusOf(u string) string { _,err:=os.Stat(filepath.Join(s.cfg.SuspendedRoot,u));if err==nil{return "suspended"};return "active" }
func (s *Server) userResponse(u string) map[string]any {return map[string]any{"username":u,"domain":u+"."+s.cfg.PublicDomain,"ssh":"ssh "+u+"@"+s.cfg.SSHHost,"status":s.statusOf(u)}}
func terminateUser(u string) error { if run("loginctl","terminate-user",u)==nil{return nil}; return run("pkill","-KILL","-u",u) }
func ensureGroup(name string) error { if out,err:=exec.Command("getent","group",name).Output();err==nil&&len(out)>0{return nil};return run("groupadd","--system",name) }
func appendUniqueKey(path,key string) error { data,err:=os.ReadFile(path);if err!=nil&&!os.IsNotExist(err){return err};for _,line:=range strings.Split(string(data),"\n"){if strings.TrimSpace(line)==key{return nil}};f,err:=os.OpenFile(path,os.O_CREATE|os.O_APPEND|os.O_WRONLY,0600);if err!=nil{return err};defer f.Close();_,err=f.WriteString(key+"\n");return err }
func sortStrings(v []string){for i:=0;i<len(v);i++{for j:=i+1;j<len(v);j++{if v[j]<v[i]{v[i],v[j]=v[j],v[i]}}}}
func decode(r *http.Request,v any)bool{b,err:=io.ReadAll(io.LimitReader(r.Body,65536));if err!=nil{return false};return json.Unmarshal(b,v)==nil}
func run(name string,args ...string)error{out,err:=exec.Command(name,args...).CombinedOutput();if err!=nil{return fmt.Errorf("%s failed: %s: %w",name,strings.TrimSpace(string(out)),err)};return nil}
func write(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_=json.NewEncoder(w).Encode(v)}
func returnErr(w http.ResponseWriter,err error){write(w,500,map[string]string{"error":err.Error()})}
func within(root,path string)bool{r,e1:=filepath.Abs(root);p,e2:=filepath.Abs(path);if e1!=nil||e2!=nil{return false};return p==r||strings.HasPrefix(p,r+string(os.PathSeparator))}
func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d}
func mustEnv(k string)string{v:=os.Getenv(k);if v==""{log.Fatal(errors.New(k+" is required"))};return v}
func envInt64(k string,d int64)int64{if v:=os.Getenv(k);v!=""{if n,e:=strconv.ParseInt(v,10,64);e==nil{return n}};return d}
