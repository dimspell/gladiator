package model

type WellKnown struct {
	ZeroTier ZeroTier `json:"zeroTier,omitempty"`
}

type ZeroTier struct {
	Enabled     bool   `json:"enabled,omitempty"`
	Network     string `json:"network,omitempty"`
	AuthAddress string `json:"authAddress,omitempty"`
}
