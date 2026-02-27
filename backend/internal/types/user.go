package types

type Preference struct {
	HideNSFW    *bool `json:"hideNSFW,omitempty"`
	HideViolent *bool `json:"hideViolent,omitempty"`
}
