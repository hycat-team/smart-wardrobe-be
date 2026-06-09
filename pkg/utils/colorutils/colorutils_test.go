package colorutils

import (
	"testing"
)

func TestHexToHSL(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		wantH   float64
		wantS   float64
		wantL   float64
		wantErr bool
	}{
		{
			name:    "Pure Black",
			hex:     "#000000",
			wantH:   0,
			wantS:   0,
			wantL:   0,
			wantErr: false,
		},
		{
			name:    "Pure White",
			hex:     "#FFFFFF",
			wantH:   0,
			wantS:   0,
			wantL:   100,
			wantErr: false,
		},
		{
			name:    "Pure Red",
			hex:     "#FF0000",
			wantH:   0,
			wantS:   100,
			wantL:   50,
			wantErr: false,
		},
		{
			name:    "Pure Green",
			hex:     "#00FF00",
			wantH:   120,
			wantS:   100,
			wantL:   50,
			wantErr: false,
		},
		{
			name:    "Pure Blue",
			hex:     "#0000FF",
			wantH:   240,
			wantS:   100,
			wantL:   50,
			wantErr: false,
		},
		{
			name:    "Short Format White",
			hex:     "FFF",
			wantH:   0,
			wantS:   0,
			wantL:   100,
			wantErr: false,
		},
		{
			name:    "Short Format Red with Hash",
			hex:     "#F00",
			wantH:   0,
			wantS:   100,
			wantL:   50,
			wantErr: false,
		},
		{
			name:    "Olive Green shade",
			hex:     "#556B2F",
			wantH:   82,
			wantS:   39,
			wantL:   30,
			wantErr: false,
		},
		{
			name:    "Invalid Hex Length",
			hex:     "#FFFF",
			wantH:   0,
			wantS:   0,
			wantL:   0,
			wantErr: true,
		},
		{
			name:    "Invalid Hex Characters",
			hex:     "#ZZZZZZ",
			wantH:   0,
			wantS:   0,
			wantL:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotH, gotS, gotL, err := HexToHSL(tt.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("HexToHSL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotH != tt.wantH || gotS != tt.wantS || gotL != tt.wantL {
					t.Errorf("HexToHSL() got = (%v, %v, %v), want = (%v, %v, %v)", gotH, gotS, gotL, tt.wantH, tt.wantS, tt.wantL)
				}
			}
		})
	}
}

func TestResolveHSLFromColorName(t *testing.T) {
	tests := []struct {
		name      string
		colorStr  string
		wantH     float64
		wantS     float64
		wantL     float64
		wantFound bool
	}{
		{
			name:      "Hex string match",
			colorStr:  "#FF0000",
			wantH:     0,
			wantS:     100,
			wantL:     50,
			wantFound: true,
		},
		{
			name:      "Vietnamese color name match",
			colorStr:  "Đỏ tươi",
			wantH:     0,
			wantS:     100,
			wantL:     50,
			wantFound: true,
		},
		{
			name:      "Vietnamese compound color match",
			colorStr:  "Xanh navy đậm",
			wantH:     240,
			wantS:     100,
			wantL:     25,
			wantFound: true,
		},
		{
			name:      "No match",
			colorStr:  "màu lạ lùng",
			wantH:     0,
			wantS:     0,
			wantL:     0,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotH, gotS, gotL, gotHex, gotFound := ResolveHSLFromColorName(tt.colorStr)
			if gotFound != tt.wantFound {
				t.Errorf("ResolveHSLFromColorName() gotFound = %v, want = %v", gotFound, tt.wantFound)
				return
			}
			if gotFound {
				if gotH != tt.wantH || gotS != tt.wantS || gotL != tt.wantL {
					t.Errorf("ResolveHSLFromColorName() got = (%v, %v, %v), want = (%v, %v, %v)", gotH, gotS, gotL, tt.wantH, tt.wantS, tt.wantL)
				}
				if gotHex == "" {
					t.Errorf("ResolveHSLFromColorName() got empty hex for matched color")
				}
			}
		})
	}
}
