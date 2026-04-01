package server

type Tier string

const (
	TierFree Tier = "free"
	TierPro  Tier = "pro"
)

type Limits struct {
	Tier        Tier
	Description string
}

func LimitsFor(tier string) Limits {
	if tier == "pro" {
		return Limits{Tier: TierPro, Description: "Unlimited vaults and secrets"}
	}
	return Limits{Tier: TierFree, Description: "2 vaults, 10 secrets"}
}

func (l Limits) IsPro() bool {
	return l.Tier == TierPro
}
