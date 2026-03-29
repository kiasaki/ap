package ap

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gitlab.com/golang-commonmark/markdown"
)

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func Checkm(err error, message string) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", message, err))
	}
}

func Assert(cond bool, message string, args ...any) {
	if !cond {
		panic(fmt.Errorf(message, args...))
	}
}

func Env(key, alt string) string {
	return Or(os.Getenv(key), alt)
}

func Or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func UUID() string {
	u := [16]byte{}
	_, err := rand.Read(u[:16])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", u)
}

func FormatMillions(v int64) string {
	millions := float64(v) / 1_000_000.0
	return fmt.Sprintf("%.1fM", millions)
}

func FormatIntComma(v int64) string {
	s := strconv.FormatInt(v, 10)
	if len(s) <= 3 {
		return s
	}
	sign := ""
	if s[0] == '-' {
		sign = "-"
		s = s[1:]
	}
	head := len(s) % 3
	if head == 0 {
		head = 3
	}
	out := s[:head]
	for i := head; i < len(s); i += 3 {
		out += "," + s[i:i+3]
	}
	return sign + out
}

func ToSnakeCase(src string) string {
	buf := ""
	for i, v := range src {
		if i > 0 && isUpper(v) && !isUpper([]rune(src)[i-1]) {
			buf += "_"
		}
		buf += string(v)
	}
	return strings.ToLower(buf)
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func StringToJSON(v string) J {
	j := J{}
	Check(json.Unmarshal([]byte(v), &j))
	return j
}

func JSONToString(v interface{}) string {
	s, err := json.Marshal(v)
	Check(err)
	return string(s)
}

func MarkdownToHTML(text string) string {
	md := markdown.New(
		markdown.HTML(true),
		markdown.Tables(true),
		markdown.Typographer(true),
		markdown.XHTMLOutput(true),
		markdown.Nofollow(true))
	return md.RenderToString([]byte(text))
}

func FloatInOptions(value float64, options []float64) bool {
	const epsilon = 0.0001
	for _, option := range options {
		fmt.Println("floatInOptions", math.Abs(option-value), value, option, options)
		if math.Abs(option-value) < epsilon {
			return true
		}
	}
	return false
}

type StringOption interface {
	GetID() string
}

func StringInOptions[T StringOption](value string, options []T) bool {
	for _, option := range options {
		if option.GetID() == value {
			return true
		}
	}
	return false
}

func Truncate(text string, length int) string {
	if len(text) > length {
		return text[:length-1] + "…"
	}
	return text
}

func TimeAgo(t time.Time) string {
	now := time.Now()
	d := now.Sub(t)
	if t.IsZero() || t.After(now) || d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
}

func FormatDatetime(t time.Time) string {
	loc, err := time.LoadLocation("America/Toronto")
	if err != nil {
		loc = time.Local
	}
	return t.In(loc).Format("Jan 2, 15:04")
}

func QuoteTranscriptValue(s string) string {
	if s == "" {
		return "\"\""
	}
	if !strings.ContainsAny(s, " \t\n\r") {
		return s
	}
	return strconv.Quote(s)
}

func ToolInputText(v string) (r string) {
	defer func() {
		if err := recover(); err != nil {
			r = v
		}
	}()
	j := StringToJSON(v)
	if l, ok := j["todos"]; ok {
		return strconv.Itoa(len(l.([]interface{})))
	}
	return Or(Or(Or(j.Get("path"), j.Get("command")), j.Get("query")), v)
}

func HTTPGet(v interface{}, url string, headers map[string]string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HttpGet: status not ok %d: %s", resp.StatusCode, string(bs))
	}
	if err := json.Unmarshal(bs, v); err != nil {
		return err
	}
	return nil
}

func HTTPPost(v interface{}, url string, headers map[string]string, body interface{}) {
	req, err := http.NewRequest("POST", url, strings.NewReader(JSONToString(body)))
	Check(err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	Check(err)
	defer resp.Body.Close()
	bs, err := io.ReadAll(resp.Body)
	Check(err)
	if resp.StatusCode != 200 {
		panic(fmt.Errorf("HttpPost: status not ok %d: %s", resp.StatusCode, string(bs)))
	}
	Check(json.Unmarshal(bs, v))
}
