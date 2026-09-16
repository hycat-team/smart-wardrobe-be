package parser

import "regexp"

var (
	// reOrphanTelex dùng để tìm các ký tự Telex đứng một mình ở cuối từ tiếng Anh bị gõ nhầm (ví dụ: s, f, r, x, j, w).
	reOrphanTelex = regexp.MustCompile(`\b([a-z]+)([sfrxjw])\b`)
	// reSplitSentence dùng để phân tách câu dựa trên các ký tự phân tách như dấu phẩy, chấm phẩy, gạch ngang, hoặc dấu chấm.
	reSplitSentence = regexp.MustCompile(`[,;\-\.]`)

	// protectedWords danh sách các từ tiếng Anh thông dụng chứa đuôi s, f, r, x, j, w cần bảo vệ để tránh bị bộ lọc Telex xóa nhầm.
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

	// LexicalStopwords danh sách các từ dừng (stopwords) trong cả tiếng Việt và tiếng Anh bị loại bỏ khi xử lý tìm kiếm từ khóa.
	LexicalStopwords = map[string]bool{
		"ban": true, "can": true, "cho": true, "cua": true, "dang": true,
		"dep": true, "do": true, "giup": true, "hom": true, "mac": true,
		"minh": true, "mot": true, "muon": true, "nay": true, "nha": true,
		"phoi": true, "thich": true, "troi": true, "voi": true, "vua": true,
		"wear": true, "want": true, "outfit": true, "please": true,
	}
)
