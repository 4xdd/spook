package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequire(t *testing.T) {
	keys, _, err := Load(t.TempDir()+"/keys", []string{"secret"})
	if err != nil {
		t.Fatal(err)
	}

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := Require(keys, ok)

	cases := []struct {
		name   string
		path   string
		header http.Header
		query  string
		status int
	}{
		{name: "health open", path: "/health", status: http.StatusOK},
		{name: "spa open", path: "/", status: http.StatusOK},
		{name: "api missing", path: "/api/v1/stats", status: http.StatusUnauthorized},
		{
			name:   "api bearer",
			path:   "/api/v1/stats",
			header: http.Header{"Authorization": []string{"Bearer secret"}},
			status: http.StatusOK,
		},
		{
			name:   "api header",
			path:   "/api/v1/art/x",
			header: http.Header{"X-Spook-Access-Key": []string{"secret"}},
			status: http.StatusOK,
		},
		{
			name:   "api query",
			path:   "/api/v1/stream/1",
			query:  "key=secret",
			status: http.StatusOK,
		},
		{
			name:   "api bad key",
			path:   "/api/v1/stats",
			header: http.Header{"Authorization": []string{"Bearer nope"}},
			status: http.StatusUnauthorized,
		},
		{
			name:   "options passthrough",
			path:   "/api/v1/stats",
			status: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			method := http.MethodGet
			if tc.name == "options passthrough" {
				method = http.MethodOptions
			}
			url := tc.path
			if tc.query != "" {
				url += "?" + tc.query
			}
			req := httptest.NewRequest(method, url, nil)
			for k, values := range tc.header {
				for _, v := range values {
					req.Header.Add(k, v)
				}
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, tc.status, rec.Body.String())
			}
		})
	}
}
