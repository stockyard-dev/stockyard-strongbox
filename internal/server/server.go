package server

import (
	"encoding/json"
	"github.com/stockyard-dev/stockyard-strongbox/internal/store"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}
	s.mux.HandleFunc("GET /api/secrets", s.list)
	s.mux.HandleFunc("POST /api/secrets", s.set)
	s.mux.HandleFunc("GET /api/secrets/{id}", s.get)
	s.mux.HandleFunc("DELETE /api/secrets/{id}", s.del)
	s.mux.HandleFunc("GET /api/resolve", s.resolve)
	s.mux.HandleFunc("GET /api/environments", s.environments)
	s.mux.HandleFunc("GET /api/audit", s.audit)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		wj(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/strongbox/"})
	})
	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func wj(w http.ResponseWriter, c int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	json.NewEncoder(w).Encode(v)
}
func we(w http.ResponseWriter, c int, m string) { wj(w, c, map[string]string{"error": m}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", 302)
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"secrets": oe(s.db.ListSecrets(r.URL.Query().Get("environment")))})
}
func (s *Server) set(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Value       string `json:"value"`
		Environment string `json:"environment"`
		Description string `json:"description"`
		Actor       string `json:"actor"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" {
		we(w, 400, "name required")
		return
	}
	if req.Environment == "" {
		req.Environment = "default"
	}
	sec := &store.Secret{Name: req.Name, Value: req.Value, Environment: req.Environment, Description: req.Description}
	if err := s.db.SetSecret(sec, req.Actor); err != nil {
		we(w, 500, err.Error())
		return
	}
	wj(w, 200, map[string]any{"id": sec.ID, "name": sec.Name, "version": sec.Version})
}
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	sec := s.db.GetSecretByID(r.PathValue("id"))
	if sec == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, sec)
}
func (s *Server) del(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteSecret(r.PathValue("id"), r.URL.Query().Get("actor"))
	wj(w, 200, map[string]string{"deleted": "ok"})
}
func (s *Server) resolve(w http.ResponseWriter, r *http.Request) {
	env := r.URL.Query().Get("env")
	if env == "" {
		env = "default"
	}
	wj(w, 200, s.db.ResolveEnv(env))
}
func (s *Server) environments(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"environments": oe(s.db.Environments())})
}
func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"audit": oe(s.db.ListAudit(50))})
}
func (s *Server) stats(w http.ResponseWriter, r *http.Request) { wj(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	st := s.db.Stats()
	wj(w, 200, map[string]any{"status": "ok", "service": "strongbox", "secrets": st.Secrets})
}
func oe[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }

// ─── personalization (auto-added) ──────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("%s: warning: could not parse config.json: %v", "strongbox", err)
		return
	}
	s.pCfg = cfg
	log.Printf("%s: loaded personalization from %s", "strongbox", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read body"}`, 400)
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		http.Error(w, `{"error":"save failed"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":"saved"}`))
}
