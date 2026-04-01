package server
import("encoding/json";"net/http";"strconv";"github.com/stockyard-dev/stockyard-strongbox/internal/store")
func(s *Server)handleListVaults(w http.ResponseWriter,r *http.Request){list,_:=s.db.ListVaults();if list==nil{list=[]store.Vault{}};writeJSON(w,200,list)}
func(s *Server)handleCreateVault(w http.ResponseWriter,r *http.Request){
    if !s.limits.IsPro(){n,_:=s.db.CountVaults();if n>=2{writeError(w,403,"free tier: 2 vaults max");return}}
    var v store.Vault;json.NewDecoder(r.Body).Decode(&v)
    if v.Name==""{writeError(w,400,"name required");return}
    if err:=s.db.CreateVault(&v);err!=nil{writeError(w,500,err.Error());return}
    writeJSON(w,201,v)}
func(s *Server)handleDeleteVault(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);s.db.DeleteVault(id);writeJSON(w,200,map[string]string{"status":"deleted"})}
func(s *Server)handleListSecrets(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);list,_:=s.db.ListSecrets(id);if list==nil{list=[]store.Secret{}};writeJSON(w,200,list)}
func(s *Server)handleUpsertSecret(w http.ResponseWriter,r *http.Request){
    if !s.limits.IsPro(){n,_:=s.db.CountSecrets();if n>=10{writeError(w,403,"free tier: 10 secrets max");return}}
    id,_:=strconv.ParseInt(r.PathValue("id"),10,64)
    var req struct{Key string `json:"key"`;Value string `json:"value"`};json.NewDecoder(r.Body).Decode(&req)
    if req.Key==""{writeError(w,400,"key required");return}
    enc,err:=store.Encrypt(req.Value);if err!=nil{writeError(w,500,"encryption failed");return}
    if err:=s.db.UpsertSecret(id,req.Key,enc);err!=nil{writeError(w,500,err.Error());return}
    writeJSON(w,200,map[string]string{"status":"ok","key":req.Key})}
func(s *Server)handleGetSecret(w http.ResponseWriter,r *http.Request){
    id,_:=strconv.ParseInt(r.PathValue("id"),10,64)
    key:=r.PathValue("key")
    enc,err:=s.db.GetSecretValue(id,key);if err!=nil{writeError(w,500,err.Error());return}
    if enc==""{writeError(w,404,"secret not found");return}
    plain,err:=store.Decrypt(enc);if err!=nil{writeError(w,500,"decryption failed");return}
    writeJSON(w,200,map[string]string{"key":key,"value":plain})}
func(s *Server)handleDeleteSecret(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);s.db.DeleteSecret(id);writeJSON(w,200,map[string]string{"status":"deleted"})}
func(s *Server)handleStats(w http.ResponseWriter,r *http.Request){v,_:=s.db.CountVaults();sc,_:=s.db.CountSecrets();writeJSON(w,200,map[string]interface{}{"vaults":v,"secrets":sc})}
