package usecase

import (
	"testing"

	"smart-wardrobe-be/shared/config"
)

func TestValidateGoogleRedirectURL(t *testing.T) {
	prod := &config.Config{}
	prod.Server.Env = "vps"
	prod.Server.FrontEndOrigins = []string{"https://closy.hycat.online"}

	dev := &config.Config{}
	dev.Server.Env = "local"
	dev.Server.FrontEndOrigins = []string{"http://localhost:3000"}

	cases := []struct {
		name    string
		cfg     *config.Config
		raw     string
		wantErr bool
	}{
		{"exact host with path", prod, "https://closy.hycat.online/auth/callback?x=1", false},
		{"suffix spoof", prod, "https://closy.hycat.online.hehe/", true},
		{"userinfo trick", prod, "https://closy.hycat.online@evil.com/", true},
		{"subdomain", prod, "https://evil.closy.hycat.online/", true},
		{"http in prod", prod, "http://closy.hycat.online/", true},
		{"unknown host", prod, "https://evil.example.com/", true},
		{"empty", prod, "", true},
		{"dev localhost http", dev, "http://localhost:3000/auth/callback", false},
		{"dev wrong host", dev, "http://evil.localhost:3000/", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateGoogleRedirectURL(tc.cfg, tc.raw)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q", tc.raw)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.raw, err)
			}
		})
	}
}
