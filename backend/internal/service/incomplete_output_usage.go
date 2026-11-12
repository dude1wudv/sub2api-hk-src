package service

// incompleteOutputUsage retains counters only. A confirmed cumulative snapshot
// covers the prefix at Report; subsequent deltas are estimated independently.
// Splitting one UTF-8 string across SSE events must not round every fragment up.
type incompleteOutputUsage struct {
	semanticBytes                uint64
	runes, ascii                 uint64
	reportedRunes, reportedASCII uint64
	initialOutput                int
	initialEstimated             bool
	authoritative                bool
	authoritativeOutput          int
	reportedAtBytes              uint64
	reportedEstimated            bool
	zeroConfirmed                bool
}

func (u *incompleteOutputUsage) AddText(text string) {
	if u == nil {
		return
	}
	for _, r := range text {
		if u.runes >= maxStreamOutputUsageBytes {
			break
		}
		u.runes++
		if r <= 0x7f {
			u.ascii++
		}
	}
	u.semanticBytes = min(maxStreamOutputUsageBytes, u.semanticBytes+uint64(len(text)))
}

func (u *incompleteOutputUsage) HasOutput() bool { return u != nil && u.runes > 0 }

func (u *incompleteOutputUsage) Report(tokens int, provisional, estimated bool) {
	if u == nil || tokens < 0 || tokens > maxStreamOutputUsageTokens {
		return
	}
	if provisional {
		if tokens >= u.initialOutput {
			u.initialOutput, u.initialEstimated = tokens, estimated && tokens > 0
		}
		return
	}
	// An estimate may never displace a confirmed prefix, nor may a stale or
	// placeholder snapshot erase the unmetered tail after that prefix.
	if u.authoritative && !u.reportedEstimated && (estimated || tokens < u.authoritativeOutput) {
		return
	}
	u.authoritative, u.authoritativeOutput, u.reportedEstimated = true, tokens, estimated
	u.zeroConfirmed = false
	u.reportedRunes, u.reportedASCII, u.reportedAtBytes = u.runes, u.ascii, u.semanticBytes
}

// ReportFinal accepts an explicitly present terminal zero as real usage. Initial
// and intermediate zero placeholders must use Report instead.
func (u *incompleteOutputUsage) ReportFinal(tokens int, estimated bool) {
	if u == nil || tokens < 0 || tokens > maxStreamOutputUsageTokens {
		return
	}
	if estimated {
		u.Report(tokens, false, true)
		return
	}
	// A real terminal value is authoritative even when it corrects an earlier
	// cumulative snapshot downward. In particular, an explicit final zero is
	// never replaced by an earlier positive snapshot or a local estimate.
	u.authoritative, u.authoritativeOutput, u.reportedEstimated = true, tokens, false
	u.zeroConfirmed = tokens == 0
	u.reportedRunes, u.reportedASCII, u.reportedAtBytes = u.runes, u.ascii, u.semanticBytes
}

func incompleteTextTokens(runes, ascii uint64) int {
	if ascii*5 >= runes*4 {
		return int(min(uint64(maxStreamOutputUsageTokens), (runes+3)/4))
	}
	return int(min(uint64(maxStreamOutputUsageTokens), runes))
}

func (u *incompleteOutputUsage) Apply(existing int) (int, bool) {
	if u == nil {
		return existing, false
	}
	tail := incompleteTextTokens(u.runes-u.reportedRunes, u.ascii-u.reportedASCII)
	if u.authoritative && !u.reportedEstimated {
		if tail == 0 && (u.authoritativeOutput > 0 || u.runes == 0 || u.zeroConfirmed) {
			return u.authoritativeOutput, false
		}
		if u.authoritativeOutput > 0 || u.zeroConfirmed {
			return min(maxStreamOutputUsageTokens, u.authoritativeOutput+tail), true
		}
	}
	existing = max(existing, u.initialOutput)
	if u.runes == 0 {
		if u.authoritative {
			return max(existing, u.authoritativeOutput), u.reportedEstimated
		}
		return existing, u.initialEstimated
	}
	estimate := incompleteTextTokens(u.runes, u.ascii)
	if u.authoritative {
		estimate = max(estimate, min(maxStreamOutputUsageTokens, u.authoritativeOutput+tail))
	}
	return max(existing, estimate), true
}
