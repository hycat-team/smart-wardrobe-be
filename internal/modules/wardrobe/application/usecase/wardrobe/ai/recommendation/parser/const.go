package parser

import "regexp"

var (
	reOrphanTelex   = regexp.MustCompile(`\b([a-z]+)([sfrxjw])\b`)
	reSplitSentence = regexp.MustCompile(`[,;\-\.]`)

	protectedWords = map[string]bool{
		"tennis": true, "summer": true, "winter": true, "interview": true,
		"golf": true, "jeans": true, "shorts": true, "pants": true,
		"shoes": true, "socks": true, "dress": true, "classics": true,
		"classic": true, "streetwear": true, "bar": true, "wear": true,
		"new": true, "show": true, "view": true, "raw": true, "crew": true,
		"glow": true, "flow": true, "slow": true, "mix": true, "max": true,
		"tax": true, "box": true, "flex": true, "wax": true, "fix": true,
		"sex": true, "plus": true, "class": true, "cross": true, "glass": true,
		"boss": true, "loss": true, "miss": true, "mess": true, "press": true,
		"stress": true, "process": true, "business": true, "focus": true,
		"minus": true, "bus": true, "gas": true, "yes": true, "this": true,
	}

	LexicalStopwords = map[string]bool{
		"ban": true, "can": true, "cho": true, "cua": true, "dang": true,
		"dep": true, "do": true, "giup": true, "hom": true, "mac": true,
		"minh": true, "mot": true, "muon": true, "nay": true, "nha": true,
		"phoi": true, "thich": true, "troi": true, "voi": true, "vua": true,
		"wear": true, "want": true, "outfit": true, "please": true,
	}
)
