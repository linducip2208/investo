package model

type AlertCondition struct {
	Type      string  `json:"type"`
	Symbol    string  `json:"symbol"`
	Value     float64 `json:"value"`
	Period    int     `json:"period,omitempty"`
	Activated bool    `json:"activated"`
}
