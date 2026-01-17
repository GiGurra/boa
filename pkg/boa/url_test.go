package boa

import (
	"net/url"
	"testing"
)

// Tests for *url.URL support

func TestURL_Required(t *testing.T) {
	type Params struct {
		Endpoint Required[*url.URL] `descr:"API endpoint"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			expected, _ := url.Parse("https://api.example.com/v1")
			if p.Endpoint.Value().String() != expected.String() {
				t.Errorf("expected %v, got %v", expected, p.Endpoint.Value())
			}
		}).
		RunArgs([]string{"--endpoint", "https://api.example.com/v1"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestURL_Optional(t *testing.T) {
	type Params struct {
		Endpoint Optional[*url.URL] `descr:"API endpoint"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if !p.Endpoint.HasValue() {
				t.Error("expected endpoint to have value")
			}
			expected, _ := url.Parse("http://localhost:8080")
			if (*p.Endpoint.Value()).String() != expected.String() {
				t.Errorf("expected %v, got %v", expected, *p.Endpoint.Value())
			}
		}).
		RunArgs([]string{"--endpoint", "http://localhost:8080"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestURL_WithPath(t *testing.T) {
	type Params struct {
		Endpoint Required[*url.URL] `descr:"API endpoint"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			u := p.Endpoint.Value()
			if u.Scheme != "https" {
				t.Errorf("expected scheme https, got %s", u.Scheme)
			}
			if u.Host != "api.example.com" {
				t.Errorf("expected host api.example.com, got %s", u.Host)
			}
			if u.Path != "/api/v2/users" {
				t.Errorf("expected path /api/v2/users, got %s", u.Path)
			}
		}).
		RunArgs([]string{"--endpoint", "https://api.example.com/api/v2/users"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestURL_WithQueryParams(t *testing.T) {
	type Params struct {
		Endpoint Required[*url.URL] `descr:"API endpoint"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			u := p.Endpoint.Value()
			if u.Query().Get("page") != "1" {
				t.Errorf("expected query param page=1, got %s", u.Query().Get("page"))
			}
			if u.Query().Get("limit") != "10" {
				t.Errorf("expected query param limit=10, got %s", u.Query().Get("limit"))
			}
		}).
		RunArgs([]string{"--endpoint", "https://api.example.com/items?page=1&limit=10"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestURL_Raw(t *testing.T) {
	type Params struct {
		Endpoint *url.URL `descr:"API endpoint" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			expected, _ := url.Parse("https://raw.example.com")
			if p.Endpoint.String() != expected.String() {
				t.Errorf("expected %v, got %v", expected, p.Endpoint)
			}
		}).
		RunArgs([]string{"--endpoint", "https://raw.example.com"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestURL_EnvVar(t *testing.T) {
	type Params struct {
		Endpoint Required[*url.URL] `descr:"API endpoint" env:"TEST_API_URL"`
	}

	t.Setenv("TEST_API_URL", "https://env.example.com/api")

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			expected, _ := url.Parse("https://env.example.com/api")
			if p.Endpoint.Value().String() != expected.String() {
				t.Errorf("expected %v, got %v", expected, p.Endpoint.Value())
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestURL_ParseFormats(t *testing.T) {
	type Params struct {
		Addr Required[*url.URL] `descr:"URL address"`
	}

	testCases := []struct {
		name     string
		input    string
		scheme   string
		host     string
		path     string
	}{
		{"https", "https://example.com", "https", "example.com", ""},
		{"http with port", "http://localhost:8080", "http", "localhost:8080", ""},
		{"with path", "https://example.com/path/to/resource", "https", "example.com", "/path/to/resource"},
		{"file scheme", "file:///tmp/test.txt", "file", "", "/tmp/test.txt"},
		{"with auth", "https://user:pass@example.com", "https", "example.com", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := Params{}
			wasRun := false

			NewCmdT2("test", &params).
				WithRunFunc(func(p *Params) {
					wasRun = true
					u := p.Addr.Value()
					if u.Scheme != tc.scheme {
						t.Errorf("expected scheme %s, got %s", tc.scheme, u.Scheme)
					}
					if u.Host != tc.host {
						t.Errorf("expected host %s, got %s", tc.host, u.Host)
					}
					if u.Path != tc.path {
						t.Errorf("expected path %s, got %s", tc.path, u.Path)
					}
				}).
				RunArgs([]string{"--addr", tc.input})

			if !wasRun {
				t.Fatal("run func was not called")
			}
		})
	}
}
