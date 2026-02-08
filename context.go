package bamgoo

import (
	"context"
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

// Context carries invocation data for method/service.
type Context struct {
	*Meta

	Name    string
	Config  *coreEntry
	Setting Map
	Value   Map
}
