package colorutils

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// HexToHSL converts a hex color string (e.g., "#FFD1DC", "FFD1DC", "#FFF") to HSL values.
// Returns h (0 to 360), s (0 to 100), l (0 to 100), and an error if format is invalid.
func HexToHSL(hex string) (h, s, l float64, err error) {
	hex = strings.TrimPrefix(hex, "#")
	hex = strings.TrimSpace(hex)

	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}

	if len(hex) != 6 {
		return 0, 0, 0, errors.New("invalid hex color length, must be 3 or 6 characters")
	}

	rgbHex, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse hex color values: %w", err)
	}

	r := float64((rgbHex >> 16) & 0xFF) / 255.0
	g := float64((rgbHex >> 8) & 0xFF) / 255.0
	b := float64(rgbHex & 0xFF) / 255.0

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))

	l = (max + min) / 2.0

	if max == min {
		h = 0
		s = 0
	} else {
		d := max - min
		if l > 0.5 {
			s = d / (2.0 - max - min)
		} else {
			s = d / (max + min)
		}

		switch max {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6.0
			}
		case g:
			h = (b - r)/d + 2.0
		case b:
			h = (r - g)/d + 4.0
		}
		h /= 6.0
	}

	h = math.Round(h * 360.0)
	s = math.Round(s * 100.0)
	l = math.Round(l * 100.0)

	return h, s, l, nil
}

// ResolveHSLFromColorName resolves HSL values and HEX string from common Vietnamese fashion color names or a hex string.
func ResolveHSLFromColorName(colorStr string) (h, s, l float64, hex string, found bool) {
	if h, s, l, err := HexToHSL(colorStr); err == nil {
		// Normalize hex string to standard format (with #)
		cleanHex := strings.TrimPrefix(colorStr, "#")
		if len(cleanHex) == 3 {
			cleanHex = string([]byte{cleanHex[0], cleanHex[0], cleanHex[1], cleanHex[1], cleanHex[2], cleanHex[2]})
		}
		return h, s, l, "#" + strings.ToUpper(cleanHex), true
	}

	colorMap := map[string]string{
		"đen":        "#000000",
		"trắng":      "#FFFFFF",
		"đỏ":         "#FF0000",
		"xanh lá":    "#00FF00",
		"xanh lục":   "#00FF00",
		"xanh dương": "#0000FF",
		"xanh biển":  "#0000FF",
		"xanh navy":  "#000080",
		"vàng":       "#FFFF00",
		"hồng":       "#FFC0CB",
		"xám":        "#808080",
		"ghi":        "#808080",
		"nâu":        "#A52A2A",
		"cam":        "#FFA500",
		"tím":        "#800080",
		"be":         "#F5F5DC",
		"beige":      "#F5F5DC",
	}

	normalized := strings.ToLower(strings.TrimSpace(colorStr))
	for k, hexCode := range colorMap {
		if strings.Contains(normalized, k) {
			if h, s, l, err := HexToHSL(hexCode); err == nil {
				return h, s, l, hexCode, true
			}
		}
	}

	return 0, 0, 0, "", false
}

// ResolveFashionColor resolves HSL values and HEX string, prioritizing hex code over color name.
func ResolveFashionColor(colorName, colorHex string) (h, s, l float64, hex string, ok bool) {
	if colorHex != "" {
		if hVal, sVal, lVal, err := HexToHSL(colorHex); err == nil {
			return hVal, sVal, lVal, colorHex, true
		}
	}
	if colorName != "" {
		return ResolveHSLFromColorName(colorName)
	}
	return 0, 0, 0, "", false
}
