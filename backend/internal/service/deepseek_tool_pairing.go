package service

import (
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// dropOrphanDeepSeekToolMessages removes tool replies without a preceding
// assistant declaration. Keep raw messages so provider extensions survive.
func dropOrphanDeepSeekToolMessages(body []byte) []byte {
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	announced := make(map[string]struct{})
	kept := make([]string, 0, len(messages.Array()))
	changed := false
	for _, message := range messages.Array() {
		switch message.Get("role").String() {
		case "assistant":
			for _, call := range message.Get("tool_calls").Array() {
				if id := call.Get("id").String(); id != "" {
					announced[id] = struct{}{}
				}
			}
		case "tool":
			id := message.Get("tool_call_id").String()
			if _, ok := announced[id]; id == "" || !ok {
				changed = true
				continue
			}
		}
		kept = append(kept, message.Raw)
	}
	if !changed {
		return body
	}
	return replaceDeepSeekChatMessages(body, kept)
}

// normalizeDeepSeekChatMessages places answered tool calls and their replies
// together in call order. Unanswered calls and duplicate replies cannot form a
// valid pair. Only tool_calls is edited; all other message fields stay raw.
func normalizeDeepSeekChatMessages(body []byte) []byte {
	raw := gjson.GetBytes(body, "messages")
	if !raw.IsArray() {
		return body
	}
	messages := raw.Array()
	owners := make(map[string]int)
	replies := make(map[int]map[string]string)
	for i, message := range messages {
		switch message.Get("role").String() {
		case "assistant":
			for _, call := range message.Get("tool_calls").Array() {
				if id := call.Get("id").String(); id != "" {
					owners[id] = i
				}
			}
		case "tool":
			id := message.Get("tool_call_id").String()
			if owner, ok := owners[id]; id != "" && ok {
				if replies[owner] == nil {
					replies[owner] = make(map[string]string)
				}
				// Match the bridge's last-reply-wins behavior, within this
				// declaration only; reused IDs in later turns stay independent.
				replies[owner][id] = message.Raw
			}
		}
	}
	out := make([]string, 0, len(messages))
	for i, message := range messages {
		if message.Get("role").String() == "tool" {
			continue
		}
		calls := message.Get("tool_calls").Array()
		if message.Get("role").String() != "assistant" || len(calls) == 0 {
			out = append(out, message.Raw)
			continue
		}
		keptCalls := make([]string, 0, len(calls))
		keptReplies := make([]string, 0, len(calls))
		seen := make(map[string]bool)
		for _, call := range calls {
			id := call.Get("id").String()
			if reply, ok := replies[i][id]; id != "" && ok && !seen[id] {
				seen[id] = true
				keptCalls = append(keptCalls, call.Raw)
				keptReplies = append(keptReplies, reply)
			}
		}
		assistant := message.Raw
		if len(keptCalls) != len(calls) {
			var err error
			if len(keptCalls) == 0 {
				content := message.Get("content")
				if !content.Exists() || content.Type == gjson.Null || (content.Type == gjson.String && content.String() == "") || (content.IsArray() && len(content.Array()) == 0) {
					continue
				}
				assistant, err = sjson.Delete(assistant, "tool_calls")
			} else {
				assistant, err = sjson.SetRaw(assistant, "tool_calls", "["+strings.Join(keptCalls, ",")+"]")
			}
			if err != nil {
				return body
			}
		}
		out = append(out, assistant)
		out = append(out, keptReplies...)
	}
	if len(out) == len(messages) {
		unchanged := true
		for i := range out {
			if out[i] != messages[i].Raw {
				unchanged = false
				break
			}
		}
		if unchanged {
			return body
		}
	}
	return replaceDeepSeekChatMessages(body, out)
}

func replaceDeepSeekChatMessages(body []byte, messages []string) []byte {
	out, err := sjson.SetRawBytes(body, "messages", []byte("["+strings.Join(messages, ",")+"]"))
	if err != nil {
		return body
	}
	return out
}
