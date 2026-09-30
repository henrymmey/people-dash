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
)

type Config struct { Listen string; Token string; HomeRoot string; PublicDomain string; SSHHost string; DefaultQuotaBytes int64 }
type Server struct { cfg Config; mux *http.ServeMux }
type userRequest struct { Username string `json:"username"`; Email string `json:"email,omitempty"` }
type keyRequest struct { PublicKey string `json:"public_key"` }
var validUser=regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
var validKey=regexp.MustCompile(`^(ssh-(ed25519|rsa)|ecdsa-sha2-nistp256|sk-(ssh-ed25519|ecdsa-sha2-nistp256)@openssh\.com)\s+[^\s]+(?:\s+.*)?$`)
func main(){
 cfg:=Config{Listen:env("LISTEN","192.168.176.118:8080"),Token:mustEnv("PEOPLE_AGENT_TOKEN"),HomeRoot:env("HOME_ROOT","/home"),PublicDomain:env("PUBLIC_DOMAIN","p.meyerbrief.de"),SSHHost:env("SSH_HOST","ssh.p.meyerbrief.de"),DefaultQuotaBytes:envInt64("DEFAULT_QUOTA_BYTES",2147483648)}
 s:=&Server{cfg:cfg,mux:http.NewServeMux()}; s.routes(); log.Printf("people-agent listening on %s",cfg.Listen); log.Fatal(http.ListenAndServe(cfg.Listen,s.auth(s.mux)))
}
func (s *Server) routes(){ s.mux.HandleFunc("GET /health",func(w http.ResponseWriter,r *http.Request){write(w,200,map[string]any{"status":"ok"})}); s.mux.HandleFunc("POST /v1/users",s.createUser); s.mux.HandleFunc("DELETE /v1/users/{username}",s.deleteUser); s.mux.HandleFunc("POST /v1/users/{username}/ssh-keys",s.addKey); s.mux.HandleFunc("DELETE /v1/users/{username}/ssh-keys",s.removeKey); s.mux.HandleFunc("GET /v1/users/{username}/storage",s.storage) }
func (s *Server) auth(next http.Handler) http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.URL.Path=="/health"{next.ServeHTTP(w,r);return}; got:=r.Header.Get("Authorization"); want:="Bearer "+s.cfg.Token; if subtle.ConstantTimeCompare([]byte(got),[]byte(want))!=1{write(w,401,map[string]string{"error":"unauthorized"});return}; next.ServeHTTP(w,r)})}
func (s *Server) createUser(w http.ResponseWriter,r *http.Request){var in userRequest;if !decode(r,&in)||!validUser.MatchString(in.Username){write(w,400,map[string]string{"error":"invalid username"});return}; home:=filepath.Join(s.cfg.HomeRoot,in.Username); if !within(s.cfg.HomeRoot,home){write(w,400,map[string]string{"error":"invalid path"});return}; if _,err:=os.Stat(home); err==nil{write(w,409,map[string]string{"error":"user already exists"});return}; if err:=run("useradd","--create-home","--home-dir",home,"--shell","/bin/bash",in.Username);err!=nil{write(w,500,map[string]string{"error":err.Error()});return}; if err:=os.MkdirAll(filepath.Join(home,"public_html"),0755);err!=nil{returnErr(w,err)}; if err:=os.MkdirAll(filepath.Join(home,".ssh"),0700);err!=nil{returnErr(w,err)}; _=run("chown","-R",in.Username+":"+in.Username,home); write(w,201,map[string]any{"username":in.Username,"domain":in.Username+"."+s.cfg.PublicDomain,"ssh":"ssh "+in.Username+"@"+s.cfg.SSHHost})}
func (s *Server) deleteUser(w http.ResponseWriter,r *http.Request){u:=r.PathValue("username");if !validUser.MatchString(u){write(w,400,map[string]string{"error":"invalid username"});return};if err:=run("userdel","--remove",u);err!=nil{write(w,500,map[string]string{"error":err.Error()});return};write(w,200,map[string]string{"status":"deleted"})}
func (s *Server) addKey(w http.ResponseWriter,r *http.Request){u:=r.PathValue("username");if !validUser.MatchString(u){write(w,400,map[string]string{"error":"invalid username"});return};var in keyRequest;if !decode(r,&in)||!validKey.MatchString(strings.TrimSpace(in.PublicKey)){write(w,400,map[string]string{"error":"invalid public key"});return};home:=filepath.Join(s.cfg.HomeRoot,u);if _,err:=os.Stat(home);err!=nil{write(w,404,map[string]string{"error":"user not found"});return};dir:=filepath.Join(home,".ssh");if err:=os.MkdirAll(dir,0700);err!=nil{returnErr(w,err)};path:=filepath.Join(dir,"authorized_keys");f,err:=os.OpenFile(path,os.O_CREATE|os.O_APPEND|os.O_WRONLY,0600);if err!=nil{returnErr(w,err)};defer f.Close();line:=strings.TrimSpace(in.PublicKey)+"\n";if _,err=f.WriteString(line);err!=nil{returnErr(w,err)};_ = run("chown",u+":"+u,path);write(w,201,map[string]string{"status":"added"})}
func (s *Server) removeKey(w http.ResponseWriter,r *http.Request){u:=r.PathValue("username");if !validUser.MatchString(u){write(w,400,map[string]string{"error":"invalid username"});return};var in keyRequest;if !decode(r,&in)||!validKey.MatchString(strings.TrimSpace(in.PublicKey)){write(w,400,map[string]string{"error":"invalid public key"});return};path:=filepath.Join(s.cfg.HomeRoot,u,".ssh","authorized_keys");data,err:=os.ReadFile(path);if err!=nil{write(w,404,map[string]string{"error":"authorized_keys not found"});return};want:=strings.TrimSpace(in.PublicKey);var out []string;for _,line:=range strings.Split(string(data),"\n"){if strings.TrimSpace(line)!=""&&strings.TrimSpace(line)!=want{out=append(out,line)}};if err=os.WriteFile(path,[]byte(strings.Join(out,"\n")+"\n"),0600);err!=nil{returnErr(w,err)};write(w,200,map[string]string{"status":"removed"})}
func (s *Server) storage(w http.ResponseWriter,r *http.Request){u:=r.PathValue("username");if !validUser.MatchString(u){write(w,400,map[string]string{"error":"invalid username"});return};home:=filepath.Join(s.cfg.HomeRoot,u);cmd:=exec.Command("du","-sb",home);b,err:=cmd.Output();if err!=nil{write(w,404,map[string]string{"error":"user not found"});return};fields:=strings.Fields(string(b));used,_:=strconv.ParseInt(fields[0],10,64);write(w,200,map[string]int64{"used_bytes":used,"quota_bytes":s.cfg.DefaultQuotaBytes})}
func decode(r *http.Request,v any)bool{b,err:=io.ReadAll(io.LimitReader(r.Body,65536));if err!=nil{return false};return json.Unmarshal(b,v)==nil}
func run(name string,args ...string)error{out,err:=exec.Command(name,args...).CombinedOutput();if err!=nil{return fmt.Errorf("%s failed: %s: %w",name,strings.TrimSpace(string(out)),err)};return nil}
func write(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func returnErr(w http.ResponseWriter,err error){write(w,500,map[string]string{"error":err.Error()})}
func within(root,path string)bool{r,_:=filepath.Abs(root);p,_:=filepath.Abs(path);return p==r||strings.HasPrefix(p,r+string(os.PathSeparator))}
func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d}
func mustEnv(k string)string{v:=os.Getenv(k);if v==""{log.Fatal(errors.New(k+" is required"))};return v}
func envInt64(k string,d int64)int64{if v:=os.Getenv(k);v!=""{if n,e:=strconv.ParseInt(v,10,64);e==nil{return n}};return d}
