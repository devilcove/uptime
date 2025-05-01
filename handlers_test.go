package main

import (
	"net/url"
	"testing"
)

func Test_validateURL(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{
			name: "invalid scheme",
			args: "htttp://test.example.com/junk",
			want: false,
		},
		{
			name: "no scheme",
			args: "/test.example.com/junk",
			want: false,
		},
		{
			name: "no host",
			args: "https://",
			want: false,
		},
		{
			name: "invalid host",
			args: "https://example.invalid/junk",
			want: false,
		},
		{
			name: "good",
			args: "https://example.com/junk",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse(tt.args)
			t.Log(u.Scheme, u.Host, u.Path)
			if got := validateURL(tt.args); got != tt.want {
				t.Errorf("validateURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
