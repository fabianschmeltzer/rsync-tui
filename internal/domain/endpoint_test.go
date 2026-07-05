package domain

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  Endpoint
	}{
		{
			name:  "local",
			value: "/srv/source",
			want:  Endpoint{Kind: EndpointLocal, Path: "/srv/source"},
		},
		{
			name:  "scp",
			value: "alice@example.test:/archive",
			want:  Endpoint{Kind: EndpointSSH, User: "alice", Host: "example.test", Path: "/archive"},
		},
		{
			name:  "url",
			value: "ssh://alice@example.test:2222/archive",
			want:  Endpoint{Kind: EndpointSSH, User: "alice", Host: "example.test", Port: 2222, Path: "/archive"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseEndpoint(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("ParseEndpoint(%q) = %+v, want %+v", test.value, got, test.want)
			}
		})
	}
}

func TestParseEndpointRejectsEmptyValue(t *testing.T) {
	if _, err := ParseEndpoint("  "); err == nil {
		t.Fatal("empty endpoint was accepted")
	}
}
