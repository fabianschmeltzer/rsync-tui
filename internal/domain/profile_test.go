package domain

import "testing"

func validProfile() Profile {
	profile := NewProfile("test")
	profile.Source.Path = "/source"
	profile.Destination.Path = "/destination"
	return profile
}

func TestProfileValidation(t *testing.T) {
	profile := validProfile()
	if !profile.RemoveEmptyDirs {
		t.Fatal("new profiles must remove empty directories after a move")
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}

	profile.Destination = Endpoint{Kind: EndpointSSH, Host: "backup", Path: "/data"}
	profile.Source = Endpoint{Kind: EndpointSSH, Host: "source", Path: "/data"}
	if err := profile.Validate(); err == nil {
		t.Fatal("remote-to-remote profile was accepted")
	}
}

func TestScheduledDestructiveRequiresLimits(t *testing.T) {
	profile := validProfile()
	profile.Mode = ModeMirror
	profile.Schedule = Schedule{Enabled: true, OnCalendar: "daily"}
	if err := profile.Validate(); err == nil {
		t.Fatal("scheduled mirror without authorization was accepted")
	}
	profile.Safety.AllowUnattendedDestructive = true
	if err := profile.Validate(); err == nil {
		t.Fatal("scheduled mirror without max_delete was accepted")
	}
	profile.Safety.MaxDelete = 10
	if err := profile.Validate(); err != nil {
		t.Fatalf("authorized scheduled mirror rejected: %v", err)
	}
}

func TestEndpointRejectsUnsafeHost(t *testing.T) {
	endpoint := Endpoint{Kind: EndpointSSH, Host: "host;rm -rf /", Path: "/data"}
	if err := endpoint.Validate(); err == nil {
		t.Fatal("unsafe SSH host was accepted")
	}
	endpoint = Endpoint{Kind: EndpointSSH, Host: "-oProxyCommand=evil", Path: "/data"}
	if err := endpoint.Validate(); err == nil {
		t.Fatal("option-like SSH host was accepted")
	}
}
