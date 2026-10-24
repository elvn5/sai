package routes_test

import (
	"strings"
	"testing"
	"time"
)

func TestPasswordByteLimit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		char  string
		count int
	}{
		{"ASCII", "a", 72},
		{"Cyrillic", "я", 36},
		{"Emoji", "🏔", 18},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := setup(t)
			valid := strings.Repeat(tc.char, tc.count)
			long := valid + tc.char
			register := func(password string) response {
				return e.do("POST", "/auth/user/register", `{"email":"length@example.kg","password":"`+password+`"}`)
			}
			if r := register(long); r.status != 400 || !strings.Contains(r.body, `"password":["Password must contain at most 72 UTF-8 bytes"]`) {
				t.Fatalf("long registration: %d %s", r.status, r.body)
			}
			if r := register(valid); r.status != 200 {
				t.Fatalf("72-byte registration: %d %s", r.status, r.body)
			}
			e.do("POST", "/auth/user/verify", `{"identifier":"length@example.kg","code":"`+e.email.codes["length@example.kg"]+`"}`)
			*e.clock = e.clock.Add(2 * time.Minute)
			e.do("POST", "/auth/user/password/forgot", `{"identifier":"length@example.kg"}`)
			code := e.email.codes["length@example.kg"]
			reset := func(password string) response {
				return e.do("POST", "/auth/user/password/reset", `{"identifier":"length@example.kg","code":"`+code+`","password":"`+password+`"}`)
			}
			if r := reset(long); r.status != 400 || !strings.Contains(r.body, `"password"`) {
				t.Fatalf("long reset: %d %s", r.status, r.body)
			}
			if r := reset(valid); r.status != 200 {
				t.Fatalf("72-byte reset after rejected attempt: %d %s", r.status, r.body)
			}
			if r := e.do("POST", "/auth/user/login", `{"identifier":"length@example.kg","password":"`+valid+`"}`); r.status != 200 || len(r.setCookie) == 0 {
				t.Fatalf("72-byte login: %d %s", r.status, r.body)
			}
		})
	}
}
