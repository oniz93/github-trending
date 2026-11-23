package github

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestRateLimitWait(t *testing.T) {
	resp := func(code int, headers map[string]string) *http.Response {
		r := &http.Response{StatusCode: code, Header: http.Header{}}
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		return r
	}
	reset := strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10)

	cases := []struct {
		name        string
		resp        *http.Response
		body        string
		attempt     int
		wantLimited bool
		min, max    time.Duration
	}{
		{"ok is not limited", resp(200, nil), "", 0, false, 0, 0},
		{"plain 403 is a real error", resp(403, nil), `{"message":"Repository access blocked"}`, 0, false, 0, 0},
		{"retry-after wins", resp(429, map[string]string{"Retry-After": "30"}), "", 0, true, 30 * time.Second, 30 * time.Second},
		{"primary limit sleeps to reset", resp(403, map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": reset}),
			`{"message":"API rate limit exceeded for user ID 1."}`, 0, true, 9 * time.Minute, 11 * time.Minute},
		{"secondary limit starts at a minute", resp(403, nil), `{"message":"You have exceeded a secondary rate limit."}`, 0, true, time.Minute, time.Minute},
		{"secondary limit caps at 16m", resp(429, nil), "", 30, true, 16 * time.Minute, 16 * time.Minute},
	}
	for _, c := range cases {
		wait, limited := rateLimitWait(c.resp, []byte(c.body), c.attempt)
		if limited != c.wantLimited || wait < c.min || wait > c.max {
			t.Errorf("%s: got (%v, %v), want limited=%v wait in [%v, %v]", c.name, wait, limited, c.wantLimited, c.min, c.max)
		}
	}
}
