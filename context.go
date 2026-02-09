package bamgoo

import (
	"context"
	"os"
	"sync"
	"time"

	. "github.com/bamgoo/base"
)

type (
	Meta struct {
		mutex sync.RWMutex
		ctx   context.Context

		traceId  string
		spanId   string
		parentId string

		language string
		timezone int
		token    string

		result Res

		tempfiles []string
		payload   Map
		id        string
	}

	Metadata struct {
		TraceId  string `json:"tid,omitempty"`
		SpanId   string `json:"sid,omitempty"`
		ParentId string `json:"pid,omitempty"`
		Language string `json:"l,omitempty"`
		Timezone int    `json:"z,omitempty"`
		Token    string `json:"t,omitempty"`
	}
)

func NewMeta() *Meta {
	return &Meta{ctx: context.Background()}
}

// close releases temp files/directories created by meta.
func (m *Meta) close() {
	for _, file := range m.tempfiles {
		_ = os.Remove(file)
	}
}

func (m *Meta) WithContext(ctx context.Context) *Meta {
	if ctx == nil {
		ctx = context.Background()
	}
	m.ctx = ctx
	return m
}

func (m *Meta) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *Meta) TraceId(id ...string) string {
	if len(id) > 0 {
		m.traceId = id[0]
	}
	return m.traceId
}

func (m *Meta) SpanId(id ...string) string {
	if len(id) > 0 {
		m.spanId = id[0]
	}
	return m.spanId
}

func (m *Meta) ParentId(id ...string) string {
	if len(id) > 0 {
		m.parentId = id[0]
	}
	return m.parentId
}

func (m *Meta) Language(v ...string) string {
	if len(v) > 0 {
		m.language = v[0]
	}
	return m.language
}

// String returns localized string by language.
func (m *Meta) String(key string, args ...Any) string {
	return String(m.Language(), key, args...)
}
func (m *Meta) Timezone(zones ...*time.Location) *time.Location {
	if len(zones) > 0 {
		_, offset := time.Now().In(zones[0]).Zone()
		m.timezone = offset
	}
	if m.timezone == 0 {
		return time.Local
	}
	return time.FixedZone("", m.timezone)
}

func (m *Meta) Token(v ...string) string {
	if len(v) > 0 {
		m.token = v[0]
	}
	return m.token
}

// Verify stores token into meta. Placeholder for signature verification.
func (m *Meta) Verify(token string) error {
	if token == "" {
		return nil
	}
	m.token = token
	return nil
}

// Signed returns whether token is present (placeholder for signature verification).
func (m *Meta) Signed(_ ...string) bool {
	return m.token != ""
}

// Unsigned is the negation of Signed.
func (m *Meta) Unsigned(roles ...string) bool {
	return !m.Signed(roles...)
}

// Authed returns whether token is present (placeholder for auth verification).
func (m *Meta) Authed(_ ...string) bool {
	return m.token != ""
}

// Unauthed is the negation of Authed.
func (m *Meta) Unauthed(roles ...string) bool {
	return !m.Authed(roles...)
}

// TokenId returns token id placeholder.
func (m *Meta) TokenId() string {
	return m.id
}

// Payload returns token payload placeholder.
func (m *Meta) Payload() Map {
	if m.payload == nil {
		return Map{}
	}
	return m.payload
}

func (m *Meta) Result(res ...Res) Res {
	if len(res) > 0 {
		m.result = res[0]
		return res[0]
	}

	if m.result == nil {
		return OK
	}

	ret := m.result
	m.result = nil
	return ret
}

func (m *Meta) Metadata(data ...Metadata) Metadata {
	if len(data) > 0 {
		d := data[0]
		m.traceId = d.TraceId
		m.spanId = d.SpanId
		m.parentId = d.ParentId
		m.language = d.Language
		m.timezone = d.Timezone
		m.token = d.Token

		if d.Token != "" {
			_ = m.Verify(d.Token)
		}
	}

	return Metadata{
		TraceId:  m.traceId,
		SpanId:   m.spanId,
		ParentId: m.parentId,
		Language: m.language,
		Timezone: m.timezone,
		Token:    m.token,
	}
}

// TempFile creates a temp file and tracks it for cleanup.
func (m *Meta) TempFile(patterns ...string) (*os.File, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.tempfiles == nil {
		m.tempfiles = make([]string, 0)
	}

	file, err := tempFile(patterns...)
	if err == nil {
		m.tempfiles = append(m.tempfiles, file.Name())
	}
	return file, err
}

// TempDir creates a temp dir and tracks it for cleanup.
func (m *Meta) TempDir(patterns ...string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.tempfiles == nil {
		m.tempfiles = make([]string, 0)
	}

	name, err := tempDir(patterns...)
	if err == nil {
		m.tempfiles = append(m.tempfiles, name)
	}
	return name, err
}

// Invoke calls another service (local first, then bus).
// It stores the result in meta and returns only the data.
func (m *Meta) Invoke(name string, values ...Map) Map {
	var value Map
	if len(values) > 0 {
		value = values[0]
	}
	data, res := core.Invoke(m, name, value)
	m.result = res
	return data
}

// CloseMeta should be called after request finishes to cleanup meta.
func CloseMeta(meta *Meta) {
	if meta == nil {
		return
	}
	meta.close()
}

func tempFile(patterns ...string) (*os.File, error) {
	pattern := ""
	if len(patterns) > 0 {
		pattern = patterns[0]
	}
	dir := os.TempDir()
	return os.CreateTemp(dir, pattern)
}

func tempDir(patterns ...string) (string, error) {
	pattern := ""
	if len(patterns) > 0 {
		pattern = patterns[0]
	}
	dir := os.TempDir()
	return os.MkdirTemp(dir, pattern)
}

// Context carries invocation data for method/service.
type Context struct {
	*Meta

	Name    string
	Config  *coreEntry
	Setting Map
	Value   Map
}
