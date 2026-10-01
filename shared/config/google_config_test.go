package config

import (
	"strings"
	"testing"
)

func validGoogle() Google {
	return Google{
		WebClientID:     "web-client",
		WebClientSecret: "web-secret",
		RedirectURI:     "https://api.example.com/api/v1/auth/google/callback",
		StateTTLMinutes: 10,
	}
}

func TestValidateGoogleDisabledWhenUnset(t *testing.T) {
	if err := validateGoogle(&Config{}); err != nil {
		t.Fatalf("empty Google config must disable the feature, got %v", err)
	}
}

func TestValidateGoogleRejectsPartialConfig(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Google)
		wantSub string
	}{
		{
			name:    "redirect_uri missing",
			mutate:  func(g *Google) { g.RedirectURI = "" },
			wantSub: "redirect_uri is required",
		},
		{
			name:    "web client id missing",
			mutate:  func(g *Google) { g.WebClientID = "" },
			wantSub: "GOOGLE_WEB_CLIENT_ID",
		},
		{
			name:    "web client secret missing",
			mutate:  func(g *Google) { g.WebClientSecret = "" },
			wantSub: "GOOGLE_WEB_CLIENT_SECRET",
		},
		{
			name:    "state ttl not positive",
			mutate:  func(g *Google) { g.StateTTLMinutes = 0 },
			wantSub: "state_ttl_minutes must be greater than 0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := validGoogle()
			g.ClientIDRaw = "android-client"
			g.ClientIDs = googleAudiences(g.ClientIDRaw, g.WebClientID)
			tc.mutate(&g)

			err := validateGoogle(&Config{Google: g})
			if err == nil {
				t.Fatal("expected the partial config to be rejected at startup")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q must name the offending setting %q", err, tc.wantSub)
			}
		})
	}
}

// A mobile client ID listed first is the exact production failure that motivated
// splitting the web client out of the audience list: the web flow then fails with
// an opaque redirect_uri_mismatch from Google. The web client must survive every
// audience ordering.
func TestGoogleAudiencesAlwaysIncludeWebClient(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "mobile only", raw: "android-client", want: []string{"web-client", "android-client"}},
		{name: "empty raw", raw: "", want: []string{"web-client"}},
		{
			name: "web repeated in the list",
			raw:  "web-client,android-client",
			want: []string{"web-client", "android-client"},
		},
		{
			name: "duplicates and blank entries dropped",
			raw:  " android-client , ,android-client,,ios-client",
			want: []string{"web-client", "android-client", "ios-client"},
		},
		{name: "spaces trimmed", raw: " android-client ", want: []string{"web-client", "android-client"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := googleAudiences(tc.raw, "web-client")
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
