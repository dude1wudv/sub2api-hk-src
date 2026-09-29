package handler

const slowTTFTSlotVetoContextKey = "slow_ttft_slot_veto"

// Slow protection shares the existing bounded admission retry loop, but must
// not report a profit-control failure to clients when that loop is exhausted.
func postSlotVetoMessage(reason string) string {
	if reason == "slow_ttft_paused" {
		return "No available accounts: slow first-output protection is active"
	}
	return profitVetoExhaustedMessage
}
