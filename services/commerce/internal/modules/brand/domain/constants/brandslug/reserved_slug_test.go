package brandslug

import "testing"

func TestIsReserved(t *testing.T) {
	for _, slug := range []string{"brands", "admin", "BRAND-PORTAL", " profile "} {
		if !IsReserved(slug) {
			t.Errorf("expected %q to be reserved", slug)
		}
	}

	if IsReserved("mori-studio") {
		t.Error("expected mori-studio to be available")
	}
}
