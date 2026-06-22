package chat

import (
	"fmt"
	"regexp"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

var reWardrobeKeywords = regexp.MustCompile(`\b(tu do|ao|quan|vay|dam|giay|ao-khoac|ao khoac|do cua|mac|phoi|style|gu|mac gi)\b`)
var transitionRegex = regexp.MustCompile(`([\r\n.?!])([A-ZĐÂĂÊÔƠƯ][a-zàáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ])`)

// buildChatSystemPrompt creates a compact fashion-aware system prompt for chat generation.
func buildChatSystemPrompt(summary string, wardrobeItems []*entities.WardrobeItem, recent []*entities.Message) string {
	var builder strings.Builder
	builder.WriteString("You are the AI fashion stylist of Closy. You must reply to the user in natural, friendly Vietnamese. Do not suggest buying external products.\n")
	builder.WriteString("IMPORTANT: Please respond directly without writing down your internal thinking process or planning steps to save output tokens. However, if your generation process forces you to output thoughts, you MUST write the marker '===RESPONSE===' on its own line to separate your thinking process from your final response. Everything before '===RESPONSE===' is your internal thoughts, and everything after is your actual response.\n")
	builder.WriteString("CONSTRAINTS & RULES:\n")
	builder.WriteString("- Only mention or propose coordinating outfits from the user's personal wardrobe when the user explicitly asks or requests it. Do not proactively invite them to coordinate outfits in standard greetings or chitchat.\n")
	builder.WriteString("- If and only if the user explicitly requests to coordinate outfits or choose clothes from their personal wardrobe, respond naturally and conversationally, and MUST append the token '[ACTION:REDIRECT_OUTFIT]' at the very end of your response to trigger the dedicated outfit coordination feature.\n")
	builder.WriteString("- Do NOT append '[ACTION:REDIRECT_OUTFIT]' for normal greetings, casual conversation, or general fashion advice without a specific request to coordinate/recommend outfits from their wardrobe.\n")
	if strings.TrimSpace(summary) != "" {
		builder.WriteString("Summary of previous conversation:\n")
		builder.WriteString(summary)
		builder.WriteString("\n")
	}

	if len(wardrobeItems) > 0 {
		builder.WriteString("Available wardrobe items:\n")
		limit := min(len(wardrobeItems), 20)
		for i := range limit {
			item := wardrobeItems[i]
			builder.WriteString("- ")
			if item.Category != nil {
				builder.WriteString(item.Category.Name)
				builder.WriteString(" ")
			}
			if item.Color != nil {
				builder.WriteString(*item.Color)
				builder.WriteString(" ")
			}
			if item.Style != nil {
				builder.WriteString(*item.Style)
				builder.WriteString(" ")
			}
			builder.WriteString("\n")
		}
	}

	builder.WriteString("5 most recent messages:\n")
	for _, item := range recent {
		fmt.Fprintf(&builder, "%s: %s\n", item.Sender, item.Content)
	}

	return builder.String()
}

// isWardrobeRelatedQuery detects whether the query contains keywords asking about wardrobe or styles.
func isWardrobeRelatedQuery(content string, recent []*entities.Message) bool {
	normalized := strings.ToLower(shared.RemoveVietnameseSigns(content))
	if reWardrobeKeywords.MatchString(normalized) {
		return true
	}
	// Also check last 2 messages for ongoing fashion context
	limit := min(len(recent), 2)
	for i := range limit {
		msg := recent[len(recent)-1-i]
		normalizedRecent := strings.ToLower(shared.RemoveVietnameseSigns(msg.Content))
		if reWardrobeKeywords.MatchString(normalizedRecent) {
			return true
		}
	}
	return false
}

// FilterThinkTags takes a channel of text chunks and returns a new channel emitting only text after the '===RESPONSE===' marker.
// If the marker is not found by the end of the stream, it falls back to a heuristic cleanup of bullet-point thoughts.
func FilterThinkTags(aiTextChan <-chan string, onCleanChunk func(string)) <-chan string {
	outChan := make(chan string, 100)
	go func() {
		defer close(outChan)
		var buffer strings.Builder
		hasMarker := false
		marker := "===RESPONSE==="
		checkedStart := false

		for t := range aiTextChan {
			if hasMarker {
				outChan <- t
				onCleanChunk(t)
				continue
			}

			buffer.WriteString(t)
			str := buffer.String()

			if !checkedStart {
				if buffer.Len() >= 15 {
					trimmedStr := strings.TrimSpace(str)
					if !startsWithThoughts(trimmedStr) {
						hasMarker = true
						outChan <- str
						onCleanChunk(str)
						buffer.Reset()
						continue
					}
					checkedStart = true
				}
			}

			// Try marker
			if _, after, ok := strings.Cut(str, marker); ok {
				hasMarker = true
				cleanStart := strings.TrimLeft(after, "\r\n ")
				if cleanStart != "" {
					outChan <- cleanStart
					onCleanChunk(cleanStart)
				}
				buffer.Reset()
				continue
			}

			// Try transition regex matching in the current buffer
			matches := transitionRegex.FindAllStringSubmatchIndex(str, -1)
			if len(matches) > 0 {
				lastMatch := matches[len(matches)-1]
				startOfResponse := lastMatch[4] // Group 2 index
				hasMarker = true
				cleanStart := str[startOfResponse:]
				cleanStart = strings.TrimLeft(cleanStart, "\r\n ")
				if cleanStart != "" {
					outChan <- cleanStart
					onCleanChunk(cleanStart)
				}
				buffer.Reset()
			}
		}

		if !hasMarker && buffer.Len() > 0 {
			cleanText := stripThoughtsFallback(buffer.String())
			if cleanText != "" {
				outChan <- cleanText
				onCleanChunk(cleanText)
			}
		}
	}()
	return outChan
}

func stripThoughtsFallback(text string) string {
	text = strings.TrimSpace(text)
	if !startsWithThoughts(text) {
		return text
	}

	lines := strings.Split(text, "\n")
	var lastThoughtLineIdx = -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "-") || startsWithThoughts(trimmed) {
			lastThoughtLineIdx = i
		}
	}

	if lastThoughtLineIdx == -1 {
		return text
	}

	lastLine := lines[lastThoughtLineIdx]
	matches := transitionRegex.FindAllStringSubmatchIndex(lastLine, -1)
	if len(matches) > 0 {
		lastMatch := matches[len(matches)-1]
		startOfResponse := lastMatch[4] // Group 2
		responseText := lastLine[startOfResponse:]

		var builder strings.Builder
		builder.WriteString(responseText)
		for i := lastThoughtLineIdx + 1; i < len(lines); i++ {
			builder.WriteString("\n")
			builder.WriteString(lines[i])
		}
		return strings.TrimSpace(builder.String())
	}

	var builder strings.Builder
	for i := lastThoughtLineIdx + 1; i < len(lines); i++ {
		builder.WriteString(lines[i])
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func startsWithThoughts(str string) bool {
	str = strings.TrimSpace(str)
	if str == "" {
		return false
	}
	if strings.HasPrefix(str, "*") || strings.HasPrefix(str, "-") {
		return true
	}
	lower := strings.ToLower(str)
	keywords := []string{
		"ai:", "ai ", "user", "constraint", "role", "the ", "therefore", "based",
		"context", "goal", "greeting", "input", "thought", "draft",
	}
	for _, kw := range keywords {
		if strings.HasPrefix(lower, kw) {
			return true
		}
	}
	return false
}
