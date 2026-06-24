package service

func resolveUserBillingRateMultipliers(apiKey *APIKey, effectiveGroupMultiplier float64, account *Account) (float64, float64) {
	accountUserMultiplier := account.UserBillingRateMultiplier()
	tokenMultiplier := effectiveGroupMultiplier * accountUserMultiplier
	imageMultiplier := resolveImageRateMultiplier(apiKey, effectiveGroupMultiplier) * accountUserMultiplier
	return tokenMultiplier, imageMultiplier
}
