package model

import "time"

type Stock struct {
	ID               int64     `json:"id" db:"id"`
	Code             string    `json:"code" db:"code"`
	Name             string    `json:"name" db:"name"`
	SectorID         int64     `json:"sector_id" db:"sector_id"`
	Subsector        string    `json:"subsector" db:"subsector"`
	ListingDate      time.Time `json:"listing_date" db:"listing_date"`
	SharesOutstanding int64    `json:"shares_outstanding" db:"shares_outstanding"`
	Description      string    `json:"description" db:"description"`
	LogoURL          string    `json:"logo_url" db:"logo_url"`
	Website          string    `json:"website" db:"website"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type StockPrice struct {
	ID       int64     `json:"id" db:"id"`
	StockID  int64     `json:"stock_id" db:"stock_id"`
	Date     time.Time `json:"date" db:"date"`
	Open     float64   `json:"open" db:"open"`
	High     float64   `json:"high" db:"high"`
	Low      float64   `json:"low" db:"low"`
	Close    float64   `json:"close" db:"close"`
	Volume   int64     `json:"volume" db:"volume"`
	AdjClose float64   `json:"adj_close" db:"adj_close"`
}

type StockFundamental struct {
	ID               int64     `json:"id" db:"id"`
	StockID          int64     `json:"stock_id" db:"stock_id"`
	Period           string    `json:"period" db:"period"`
	ReportType       string    `json:"report_type" db:"report_type"`
	Source           string    `json:"source" db:"source"`
	Revenue          float64   `json:"revenue" db:"revenue"`
	NetIncome        float64   `json:"net_income" db:"net_income"`
	EPS              float64   `json:"eps" db:"eps"`
	BVPS             float64   `json:"bvps" db:"bvps"`
	TotalAssets      float64   `json:"total_assets" db:"total_assets"`
	TotalLiabilities float64   `json:"total_liabilities" db:"total_liabilities"`
	Equity           float64   `json:"equity" db:"equity"`
	ROE              float64   `json:"roe" db:"roe"`
	ROA              float64   `json:"roa" db:"roa"`
	PER              float64   `json:"per" db:"per"`
	PBV              float64   `json:"pbv" db:"pbv"`
	DER              float64   `json:"der" db:"der"`
	NetProfitMargin  float64   `json:"net_profit_margin" db:"net_profit_margin"`
	DividendYield    float64   `json:"dividend_yield" db:"dividend_yield"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type StockAction struct {
	ID          int64     `json:"id" db:"id"`
	StockID     int64     `json:"stock_id" db:"stock_id"`
	ActionType  string    `json:"action_type" db:"action_type"`
	ExDate      time.Time `json:"ex_date" db:"ex_date"`
	RecordDate  time.Time `json:"record_date" db:"record_date"`
	PaymentDate time.Time `json:"payment_date" db:"payment_date"`
	Description string    `json:"description" db:"description"`
	Ratio       float64   `json:"ratio" db:"ratio"`
	Price       float64   `json:"price" db:"price"`
}

type Sector struct {
	ID          int64  `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Slug        string `json:"slug" db:"slug"`
	Description string `json:"description" db:"description"`
}
