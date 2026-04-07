package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DB struct {
	db  *sql.DB
	key []byte
}
type Secret struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	Environment string `json:"environment,omitempty"`
	Description string `json:"description,omitempty"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
type AuditEntry struct {
	ID         string `json:"id"`
	SecretName string `json:"secret_name"`
	Action     string `json:"action"`
	Actor      string `json:"actor,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(d, "strongbox.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS secrets(id TEXT PRIMARY KEY,name TEXT NOT NULL,value TEXT DEFAULT '',environment TEXT DEFAULT 'default',description TEXT DEFAULT '',version INTEGER DEFAULT 1,created_at TEXT DEFAULT(datetime('now')),updated_at TEXT DEFAULT(datetime('now')),UNIQUE(name,environment))`,
		`CREATE TABLE IF NOT EXISTS audit(id TEXT PRIMARY KEY,secret_name TEXT DEFAULT '',action TEXT NOT NULL,actor TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`,
	} {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	h := sha256.Sum256([]byte(d + "strongbox-default-key"))
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(resource TEXT NOT NULL,record_id TEXT NOT NULL,data TEXT NOT NULL DEFAULT '{}',PRIMARY KEY(resource, record_id))`)
	return &DB{db: db, key: h[:]}, nil
}
func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }
func (d *DB) encrypt(plain string) string {
	block, err := aes.NewCipher(d.key)
	if err != nil {
		return plain
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return plain
	}
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	ct := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ct)
}
func (d *DB) decrypt(enc string) string {
	data, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return enc
	}
	block, err := aes.NewCipher(d.key)
	if err != nil {
		return enc
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return enc
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return enc
	}
	plain, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return enc
	}
	return string(plain)
}
func (d *DB) SetSecret(s *Secret, actor string) error {
	var existID string
	var oldVer int
	err := d.db.QueryRow(`SELECT id,version FROM secrets WHERE name=? AND environment=?`, s.Name, s.Environment).Scan(&existID, &oldVer)
	t := now()
	encrypted := d.encrypt(s.Value)
	if err == sql.ErrNoRows {
		s.ID = genID()
		s.Version = 1
		s.CreatedAt = t
		s.UpdatedAt = t
		_, err := d.db.Exec(`INSERT INTO secrets VALUES(?,?,?,?,?,?,?,?)`, s.ID, s.Name, encrypted, s.Environment, s.Description, 1, t, t)
		if err != nil {
			return err
		}
		d.audit(s.Name, "created", actor)
		return nil
	}
	s.ID = existID
	s.Version = oldVer + 1
	d.db.Exec(`UPDATE secrets SET value=?,description=?,version=?,updated_at=? WHERE id=?`, encrypted, s.Description, s.Version, t, existID)
	d.audit(s.Name, "updated", actor)
	return nil
}
func (d *DB) GetSecret(name, env string) *Secret {
	if env == "" {
		env = "default"
	}
	var s Secret
	if d.db.QueryRow(`SELECT id,name,value,environment,description,version,created_at,updated_at FROM secrets WHERE name=? AND environment=?`, name, env).Scan(&s.ID, &s.Name, &s.Value, &s.Environment, &s.Description, &s.Version, &s.CreatedAt, &s.UpdatedAt) != nil {
		return nil
	}
	s.Value = d.decrypt(s.Value)
	return &s
}
func (d *DB) GetSecretByID(id string) *Secret {
	var s Secret
	if d.db.QueryRow(`SELECT id,name,value,environment,description,version,created_at,updated_at FROM secrets WHERE id=?`, id).Scan(&s.ID, &s.Name, &s.Value, &s.Environment, &s.Description, &s.Version, &s.CreatedAt, &s.UpdatedAt) != nil {
		return nil
	}
	s.Value = d.decrypt(s.Value)
	return &s
}
func (d *DB) ListSecrets(env string) []Secret {
	q := `SELECT id,name,environment,description,version,created_at,updated_at FROM secrets`
	args := []any{}
	if env != "" && env != "all" {
		q += ` WHERE environment=?`
		args = append(args, env)
	}
	q += ` ORDER BY name`
	rows, _ := d.db.Query(q, args...)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []Secret
	for rows.Next() {
		var s Secret
		rows.Scan(&s.ID, &s.Name, &s.Environment, &s.Description, &s.Version, &s.CreatedAt, &s.UpdatedAt)
		s.Value = "••••••••"
		o = append(o, s)
	}
	return o
}
func (d *DB) DeleteSecret(id, actor string) error {
	s := d.GetSecretByID(id)
	if s != nil {
		d.audit(s.Name, "deleted", actor)
	}
	_, err := d.db.Exec(`DELETE FROM secrets WHERE id=?`, id)
	return err
}
func (d *DB) Environments() []string {
	rows, _ := d.db.Query(`SELECT DISTINCT environment FROM secrets ORDER BY environment`)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []string
	for rows.Next() {
		var e string
		rows.Scan(&e)
		o = append(o, e)
	}
	return o
}
func (d *DB) audit(name, action, actor string) {
	d.db.Exec(`INSERT INTO audit(id,secret_name,action,actor,created_at)VALUES(?,?,?,?,?)`, genID(), name, action, actor, now())
}
func (d *DB) ListAudit(limit int) []AuditEntry {
	if limit <= 0 {
		limit = 50
	}
	rows, _ := d.db.Query(`SELECT id,secret_name,action,actor,created_at FROM audit ORDER BY created_at DESC LIMIT ?`, limit)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []AuditEntry
	for rows.Next() {
		var a AuditEntry
		rows.Scan(&a.ID, &a.SecretName, &a.Action, &a.Actor, &a.CreatedAt)
		o = append(o, a)
	}
	return o
}
func (d *DB) ResolveEnv(env string) map[string]string {
	secrets := d.ListSecrets(env)
	m := map[string]string{}
	for _, s := range secrets {
		full := d.GetSecret(s.Name, env)
		if full != nil {
			m[s.Name] = full.Value
		}
	}
	return m
}

type Stats struct {
	Secrets      int `json:"secrets"`
	Environments int `json:"environments"`
}

func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM secrets`).Scan(&s.Secrets)
	s.Environments = len(d.Environments())
	return s
}

var _ = hex.EncodeToString
var _2 = strings.Join

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
