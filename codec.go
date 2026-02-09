package bamgoo

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"hash/fnv"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/bamgoo/base"
	"github.com/pelletier/go-toml/v2"
)

var (
	codec = &codecModule{
		config: codecConfig{
			Text:   "01234AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz56789-_/.",
			Digit:  "abcdefghijkmnpqrstuvwxyz123456789ACDEFGHJKLMNPQRSTUVWXYZ",
			Salt:   BAMGOO,
			Length: 7,

			Start:    time.Date(2023, 4, 1, 0, 0, 0, 0, time.Local),
			Timebits: 42, Nodebits: 7, Stepbits: 14,
		},
		codecs: make(map[string]Codec, 0),
	}

	errInvalidCodec     = errors.New("Invalid codec.")
	errInvalidCodecData = errors.New("Invalid codec data.")
)

const (
	JSON   = "json"
	XML    = "xml"
	GOB    = "gob"
	TOML   = "toml"
	DIGIT  = "digit"
	DIGITS = "digits"
	TEXT   = "text"
	TEXTS  = "texts"
)

type (
	codecConfig struct {
		Text   string
		Digit  string
		Salt   string
		Length int

		Start    time.Time
		Timebits uint
		Nodebits uint
		Stepbits uint
	}

	Codec struct {
		Name   string
		Text   string
		Alias  []string
		Encode EncodeFunc
		Decode DecodeFunc
	}
	EncodeFunc func(v Any) (Any, error)
	DecodeFunc func(d Any, v Any) (Any, error)

	codecModule struct {
		mutex  sync.Mutex
		config codecConfig
		codecs map[string]Codec
		fastid *fastID
	}
)

func init() {
	Mount(codec)
	codec.registerDefaults()
}

// Register
func (module *codecModule) Register(name string, value Any) {
	switch val := value.(type) {
	case Codec:
		module.Codec(name, val)
	}
}

// Config loads codec config.
func (module *codecModule) Config(global Map) {
	cfg, ok := global["codec"].(Map)
	if !ok {
		return
	}
	if text, ok := cfg["text"].(string); ok && text != "" {
		module.config.Text = text
	}
	if digit, ok := cfg["digit"].(string); ok && digit != "" {
		module.config.Digit = digit
	}
	if salt, ok := cfg["salt"].(string); ok {
		module.config.Salt = salt
	}
	if length, ok := cfg["length"].(int); ok {
		module.config.Length = length
	}
	if length, ok := cfg["length"].(int64); ok {
		module.config.Length = int(length)
	}
	if vv, ok := cfg["start"].(time.Time); ok {
		module.config.Start = vv
	}
	if vv, ok := cfg["start"].(int64); ok {
		module.config.Start = time.Unix(vv, 0)
	}
	if vv, ok := cfg["timebits"].(int); ok {
		module.config.Timebits = uint(vv)
	}
	if vv, ok := cfg["timebits"].(int64); ok {
		module.config.Timebits = uint(vv)
	}
	if vv, ok := cfg["nodebits"].(int); ok {
		module.config.Nodebits = uint(vv)
	}
	if vv, ok := cfg["nodebits"].(int64); ok {
		module.config.Nodebits = uint(vv)
	}
	if vv, ok := cfg["stepbits"].(int); ok {
		module.config.Stepbits = uint(vv)
	}
	if vv, ok := cfg["stepbits"].(int64); ok {
		module.config.Stepbits = uint(vv)
	}
}

func (module *codecModule) Setup() {
	module.fastid = newFastID(module.config.Timebits, module.config.Nodebits, module.config.Stepbits, module.config.Start.Unix())
}
func (module *codecModule) Open()  {}
func (module *codecModule) Start() {}
func (module *codecModule) Stop()  {}
func (module *codecModule) Close() {}

// Codec registers codec.
func (module *codecModule) Codec(name string, config Codec) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if Override() {
			module.codecs[key] = config
		} else {
			if _, ok := module.codecs[key]; !ok {
				module.codecs[key] = config
			}
		}
	}
}

func (module *codecModule) Codecs() map[string]Codec {
	codecs := map[string]Codec{}
	for k, v := range module.codecs {
		codecs[k] = v
	}
	return codecs
}

// Sequence returns snowflake id.
func (module *codecModule) Sequence() int64 {
	if module.fastid == nil {
		module.Setup()
	}
	return module.fastid.NextID()
}

// Generate returns hex id (simple, fast).
func (module *codecModule) Generate(prefixs ...string) string {
	id := module.Sequence()
	return strconv.FormatInt(id, 16)
}

// Encode
func (module *codecModule) Encode(codecName string, v Any) (Any, error) {
	codecName = strings.ToLower(codecName)
	if ccc, ok := module.codecs[codecName]; ok {
		return ccc.Encode(v)
	}
	return nil, errInvalidCodec
}

// Decode
func (module *codecModule) Decode(codecName string, d Any, v Any) (Any, error) {
	codecName = strings.ToLower(codecName)
	if ccc, ok := module.codecs[codecName]; ok {
		return ccc.Decode(d, v)
	}
	return nil, errInvalidCodec
}

// Marshal
func (module *codecModule) Marshal(codecName string, v Any) ([]byte, error) {
	dat, err := module.Encode(codecName, v)
	if err != nil {
		return nil, err
	}
	if bts, ok := dat.([]byte); ok {
		return bts, nil
	}
	return nil, errInvalidCodecData
}

// Unmarshal
func (module *codecModule) Unmarshal(codecName string, d []byte, v Any) error {
	_, err := module.Decode(codecName, d, v)
	return err
}

// Encrypt returns string.
func (module *codecModule) Encrypt(codecName string, v Any) (string, error) {
	dat, err := module.Encode(codecName, v)
	if err != nil {
		return "", err
	}
	switch vv := dat.(type) {
	case string:
		return vv, nil
	case []byte:
		return string(vv), nil
	default:
		return fmt.Sprintf("%v", vv), nil
	}
}

// Decrypt
func (module *codecModule) Decrypt(codecName string, v Any) (Any, error) {
	return module.Decode(codecName, v, nil)
}

// wrappers
func Encode(name string, v Any) (Any, error)             { return codec.Encode(name, v) }
func Decode(name string, data Any, obj Any) (Any, error) { return codec.Decode(name, data, obj) }
func Marshal(name string, obj Any) ([]byte, error)       { return codec.Marshal(name, obj) }
func Unmarshal(name string, data []byte, obj Any) error  { return codec.Unmarshal(name, data, obj) }
func Encrypt(name string, obj Any) (string, error)       { return codec.Encrypt(name, obj) }
func Decrypt(name string, obj Any) (Any, error)          { return codec.Decrypt(name, obj) }

func Sequence() int64                   { return codec.Sequence() }
func Generate(prefixs ...string) string { return codec.Generate(prefixs...) }

// defaults
func (module *codecModule) registerDefaults() {
	module.Codec(JSON, Codec{
		Encode: func(v Any) (Any, error) {
			if bts, ok := v.([]byte); ok {
				return bts, nil
			}
			return json.Marshal(v)
		},
		Decode: func(d Any, v Any) (Any, error) {
			data, ok := toBytes(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			if v != nil {
				return v, json.Unmarshal(data, v)
			}
			var out Any
			return out, json.Unmarshal(data, &out)
		},
	})
	module.Codec(XML, Codec{
		Encode: func(v Any) (Any, error) {
			if bts, ok := v.([]byte); ok {
				return bts, nil
			}
			return xml.Marshal(v)
		},
		Decode: func(d Any, v Any) (Any, error) {
			data, ok := toBytes(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			if v != nil {
				return v, xml.Unmarshal(data, v)
			}
			var out Any
			return out, xml.Unmarshal(data, &out)
		},
	})
	module.Codec(GOB, Codec{
		Encode: func(v Any) (Any, error) {
			if bts, ok := v.([]byte); ok {
				return bts, nil
			}
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(v); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		},
		Decode: func(d Any, v Any) (Any, error) {
			data, ok := toBytes(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			if v == nil {
				var out Any
				v = &out
			}
			dec := gob.NewDecoder(bytes.NewReader(data))
			return v, dec.Decode(v)
		},
	})
	module.Codec(TOML, Codec{
		Encode: func(v Any) (Any, error) {
			if bts, ok := v.([]byte); ok {
				return bts, nil
			}
			return toml.Marshal(v)
		},
		Decode: func(d Any, v Any) (Any, error) {
			data, ok := toBytes(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			if v != nil {
				return v, toml.Unmarshal(data, v)
			}
			var out Any
			return out, toml.Unmarshal(data, &out)
		},
	})
	module.Codec(DIGIT, Codec{
		Encode: func(v Any) (Any, error) {
			n, err := toInt64(v)
			if err != nil {
				return nil, err
			}
			return encodeInt64(n, module.config.Digit, module.config.Salt, module.config.Length)
		},
		Decode: func(d Any, v Any) (Any, error) {
			s, ok := toString(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			return decodeInt64(s, module.config.Digit, module.config.Salt)
		},
	})
	module.Codec(DIGITS, Codec{
		Encode: func(v Any) (Any, error) {
			arr, err := toInt64Slice(v)
			if err != nil {
				return nil, err
			}
			return encodeInt64Slice(arr, module.config.Digit, module.config.Salt, module.config.Length)
		},
		Decode: func(d Any, v Any) (Any, error) {
			s, ok := toString(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			return decodeInt64Slice(s, module.config.Digit, module.config.Salt)
		},
	})
	module.Codec(TEXT, Codec{
		Encode: func(v Any) (Any, error) {
			var data []byte
			switch vv := v.(type) {
			case []byte:
				data = vv
			case string:
				data = []byte(vv)
			default:
				bts, err := json.Marshal(v)
				if err != nil {
					return nil, err
				}
				data = bts
			}
			return encodeBytes(data, module.config.Text, module.config.Salt)
		},
		Decode: func(d Any, v Any) (Any, error) {
			s, ok := toString(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			data, err := decodeBytes(s, module.config.Text, module.config.Salt)
			if err != nil {
				return nil, err
			}
			if v != nil {
				return v, json.Unmarshal(data, v)
			}
			return data, nil
		},
	})
	module.Codec(TEXTS, Codec{
		Encode: func(v Any) (Any, error) {
			arr, err := toStringSlice(v)
			if err != nil {
				return nil, err
			}
			bts, err := json.Marshal(arr)
			if err != nil {
				return nil, err
			}
			return encodeBytes(bts, module.config.Text, module.config.Salt)
		},
		Decode: func(d Any, v Any) (Any, error) {
			s, ok := toString(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			data, err := decodeBytes(s, module.config.Text, module.config.Salt)
			if err != nil {
				return nil, err
			}
			var out []string
			if err := json.Unmarshal(data, &out); err != nil {
				return nil, err
			}
			return out, nil
		},
	})
}

// helpers
func toBytes(v Any) ([]byte, bool) {
	switch vv := v.(type) {
	case []byte:
		return vv, true
	case string:
		return []byte(vv), true
	default:
		return nil, false
	}
}

func toString(v Any) (string, bool) {
	switch vv := v.(type) {
	case string:
		return vv, true
	case []byte:
		return string(vv), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func toInt64(v Any) (int64, error) {
	switch vv := v.(type) {
	case int:
		return int64(vv), nil
	case int8:
		return int64(vv), nil
	case int16:
		return int64(vv), nil
	case int32:
		return int64(vv), nil
	case int64:
		return vv, nil
	case uint:
		return int64(vv), nil
	case uint8:
		return int64(vv), nil
	case uint16:
		return int64(vv), nil
	case uint32:
		return int64(vv), nil
	case uint64:
		return int64(vv), nil
	case float32:
		return int64(vv), nil
	case float64:
		return int64(vv), nil
	case string:
		return strconv.ParseInt(vv, 10, 64)
	default:
		return 0, errInvalidCodecData
	}
}

func toInt64Slice(v Any) ([]int64, error) {
	switch vv := v.(type) {
	case []int64:
		return vv, nil
	case []int:
		out := make([]int64, 0, len(vv))
		for _, n := range vv {
			out = append(out, int64(n))
		}
		return out, nil
	case []Any:
		out := make([]int64, 0, len(vv))
		for _, n := range vv {
			val, err := toInt64(n)
			if err != nil {
				return nil, err
			}
			out = append(out, val)
		}
		return out, nil
	default:
		return nil, errInvalidCodecData
	}
}

func toStringSlice(v Any) ([]string, error) {
	switch vv := v.(type) {
	case []string:
		return vv, nil
	case []Any:
		out := make([]string, 0, len(vv))
		for _, s := range vv {
			str, _ := toString(s)
			out = append(out, str)
		}
		return out, nil
	default:
		return nil, errInvalidCodecData
	}
}

func normalizeAlphabet(alphabet string) ([]rune, error) {
	if alphabet == "" {
		return nil, errInvalidCodecData
	}
	seen := map[rune]bool{}
	out := make([]rune, 0, len(alphabet))
	for _, r := range []rune(alphabet) {
		if seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	if len(out) < 2 {
		return nil, errInvalidCodecData
	}
	return out, nil
}

func rotateAlphabet(alpha []rune, salt string) []rune {
	if len(alpha) == 0 {
		return alpha
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(salt))
	off := int(h.Sum32()) % len(alpha)
	return append(alpha[off:], alpha[:off]...)
}

func pickSeparator(alpha []rune) (string, error) {
	cands := []string{"-", "_", ".", "~", "|", ":", ","}
	set := map[rune]bool{}
	for _, r := range alpha {
		set[r] = true
	}
	for _, c := range cands {
		if !set[rune(c[0])] {
			return c, nil
		}
	}
	return "", errInvalidCodecData
}

func encodeInt64(n int64, alphabet, salt string, minLen int) (string, error) {
	if n < 0 {
		return "", errInvalidCodecData
	}
	alpha, err := normalizeAlphabet(alphabet)
	if err != nil {
		return "", err
	}
	alpha = rotateAlphabet(alpha, salt)
	if n == 0 {
		return string(alpha[0]), nil
	}
	base := int64(len(alpha))
	var out []rune
	for n > 0 {
		r := n % base
		out = append(out, alpha[r])
		n = n / base
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if minLen > 0 && len(out) < minLen {
		pad := alpha[0]
		for len(out) < minLen {
			out = append([]rune{pad}, out...)
		}
	}
	return string(out), nil
}

func decodeInt64(s, alphabet, salt string) (int64, error) {
	alpha, err := normalizeAlphabet(alphabet)
	if err != nil {
		return 0, err
	}
	alpha = rotateAlphabet(alpha, salt)
	index := map[rune]int64{}
	for i, r := range alpha {
		index[r] = int64(i)
	}
	var n int64
	for _, r := range []rune(s) {
		v, ok := index[r]
		if !ok {
			return 0, errInvalidCodecData
		}
		n = n*int64(len(alpha)) + v
	}
	return n, nil
}

func encodeInt64Slice(ns []int64, alphabet, salt string, minLen int) (string, error) {
	sep, err := pickSeparator([]rune(alphabet))
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(ns))
	for _, n := range ns {
		enc, err := encodeInt64(n, alphabet, salt, minLen)
		if err != nil {
			return "", err
		}
		parts = append(parts, enc)
	}
	return strings.Join(parts, sep), nil
}

func decodeInt64Slice(s, alphabet, salt string) ([]int64, error) {
	sep, err := pickSeparator([]rune(alphabet))
	if err != nil {
		return nil, err
	}
	if s == "" {
		return []int64{}, nil
	}
	parts := strings.Split(s, sep)
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		val, err := decodeInt64(p, alphabet, salt)
		if err != nil {
			return nil, err
		}
		out = append(out, val)
	}
	return out, nil
}

func encodeBytes(data []byte, alphabet, salt string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	alpha, err := normalizeAlphabet(alphabet)
	if err != nil {
		return "", err
	}
	alpha = rotateAlphabet(alpha, salt)
	base := big.NewInt(int64(len(alpha)))
	n := new(big.Int).SetBytes(data)
	if n.Sign() == 0 {
		return string(alpha[0]), nil
	}
	var out []rune
	r := new(big.Int)
	for n.Sign() > 0 {
		n, r = new(big.Int).DivMod(n, base, r)
		out = append(out, alpha[r.Int64()])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out), nil
}

func decodeBytes(s, alphabet, salt string) ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}
	alpha, err := normalizeAlphabet(alphabet)
	if err != nil {
		return nil, err
	}
	alpha = rotateAlphabet(alpha, salt)
	index := map[rune]int64{}
	for i, r := range alpha {
		index[r] = int64(i)
	}
	base := big.NewInt(int64(len(alpha)))
	n := big.NewInt(0)
	for _, r := range []rune(s) {
		v, ok := index[r]
		if !ok {
			return nil, errInvalidCodecData
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(v))
	}
	return n.Bytes(), nil
}

// fallback: base64 urlsafe when alphabet is invalid (not used by default)
func encodeBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeBase64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// fast id (snowflake-ish)
type fastID struct {
	timeStart int64
	timeBits  uint
	stepBits  uint
	nodeBits  uint
	timeMask  int64
	stepMask  int64
	nodeID    int64
	lastID    int64
}

func newFastID(timeBits, nodeBits, stepBits uint, timeStart int64) *fastID {
	machineID := int64(0)
	timeMask := ^(int64(-1) << timeBits)
	stepMask := ^(int64(-1) << stepBits)
	nodeMask := ^(int64(-1) << nodeBits)
	return &fastID{
		timeStart: timeStart,
		timeBits:  timeBits,
		stepBits:  stepBits,
		nodeBits:  nodeBits,
		timeMask:  timeMask,
		stepMask:  stepMask,
		nodeID:    machineID & nodeMask,
		lastID:    0,
	}
}

func (f *fastID) currentTimestamp() int64 {
	return (time.Now().UnixNano() - f.timeStart) >> 20 & f.timeMask
}

func (f *fastID) NextID() int64 {
	for {
		localLast := f.lastID
		seq := f.sequence(localLast)
		lastTime := f.time(localLast)
		now := f.currentTimestamp()
		if now > lastTime {
			seq = 0
		} else if seq >= f.stepMask {
			time.Sleep(time.Duration(0xFFFFF - (time.Now().UnixNano() & 0xFFFFF)))
			continue
		} else {
			seq++
		}
		newID := now<<(f.nodeBits+f.stepBits) + seq<<f.nodeBits + f.nodeID
		if newID > localLast {
			f.lastID = newID
			return newID
		}
		time.Sleep(time.Duration(20))
	}
}

func (f *fastID) sequence(id int64) int64 { return (id >> f.nodeBits) & f.stepMask }
func (f *fastID) time(id int64) int64     { return id >> (f.nodeBits + f.stepBits) }
