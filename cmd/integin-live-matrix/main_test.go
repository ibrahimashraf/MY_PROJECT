package main

import "testing"

func TestExplicitPilotServerURL(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		wantOK bool
	}{
		{"accepts pilot", "http://127.0.0.1:18080/", true},
		{"rejects omitted", "", false},
		{"rejects acceptance", "http://127.0.0.1:8080", false},
		{"rejects alternate local port", "http://127.0.0.1:19080", false},
		{"rejects https", "https://127.0.0.1:18080", false},
		{"rejects path", "http://127.0.0.1:18080/sync", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := explicitPilotServerURL(test.raw)
			if test.wantOK {
				if err != nil || got != "http://127.0.0.1:18080" {
					t.Fatalf("explicitPilotServerURL(%q) = %q, %v", test.raw, got, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("explicitPilotServerURL(%q) unexpectedly succeeded", test.raw)
			}
		})
	}
}
