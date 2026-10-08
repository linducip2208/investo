package service

import (
	"fmt"
	"strings"

	"investo/internal/repository"
)

type SupplyChainLink struct {
	From         string  `json:"from"`
	FromName     string  `json:"from_name"`
	To           string  `json:"to"`
	ToName       string  `json:"to_name"`
	Relationship string  `json:"relationship"`
	Strength     float64 `json:"strength"`
	Description  string  `json:"description"`
}

type ConcentrationRisk struct {
	StockCode      string  `json:"stock_code"`
	StockName      string  `json:"stock_name"`
	TopCustomerPct float64 `json:"top_customer_pct"`
	TopSupplierPct float64 `json:"top_supplier_pct"`
	RiskLevel      string  `json:"risk_level"`
	Detail         string  `json:"detail"`
}

type SupplyChainService struct {
	StockRepo  *repository.StockRepository
	SectorRepo *repository.SectorRepository
}

var supplyChainData = map[string][]SupplyChainLink{
	"BBCA": {
		{From: "BBCA", FromName: "Bank Central Asia", To: "ADRO", ToName: "Adaro Energy", Relationship: "creditor", Strength: 85, Description: "Major creditor for mining operations"},
		{From: "BBCA", FromName: "Bank Central Asia", To: "ASII", ToName: "Astra International", Relationship: "creditor", Strength: 90, Description: "Automotive and infrastructure financing"},
		{From: "BBCA", FromName: "Bank Central Asia", To: "UNTR", ToName: "United Tractors", Relationship: "creditor", Strength: 80, Description: "Heavy equipment financing"},
		{From: "BBCA", FromName: "Bank Central Asia", To: "INDF", ToName: "Indofood Sukses Makmur", Relationship: "creditor", Strength: 75, Description: "Consumer goods working capital"},
		{From: "BBCA", FromName: "Bank Central Asia", To: "ICBP", ToName: "Indofood CBP", Relationship: "creditor", Strength: 70, Description: "Food manufacturing expansion"},
		{From: "BBCA", FromName: "Bank Central Asia", To: "TLKM", ToName: "Telkom Indonesia", Relationship: "creditor", Strength: 65, Description: "Telecom infrastructure financing"},
		{From: "BBCA", FromName: "Bank Central Asia", To: "SMGR", ToName: "Semen Indonesia", Relationship: "creditor", Strength: 75, Description: "Cement industry working capital"},
		{From: "BBCA", FromName: "Bank Central Asia", To: "UNVR", ToName: "Unilever Indonesia", Relationship: "creditor", Strength: 70, Description: "FMCG supply chain financing"},
	},
	"ASII": {
		{From: "ASII", FromName: "Astra International", To: "AUTO", ToName: "Astra Otoparts", Relationship: "customer", Strength: 95, Description: "Automotive components subsidiary"},
		{From: "ASII", FromName: "Astra International", To: "UNTR", ToName: "United Tractors", Relationship: "supplier", Strength: 85, Description: "Heavy equipment for mining operations"},
		{From: "ASII", FromName: "Astra International", To: "ADRO", ToName: "Adaro Energy", Relationship: "customer", Strength: 60, Description: "Mining equipment customer"},
	},
	"INDF": {
		{From: "INDF", FromName: "Indofood Sukses Makmur", To: "ICBP", ToName: "Indofood CBP", Relationship: "customer", Strength: 95, Description: "Noodle and food manufacturing subsidiary"},
		{From: "INDF", FromName: "Indofood Sukses Makmur", To: "BBCA", ToName: "Bank Central Asia", Relationship: "customer", Strength: 70, Description: "Banking services for operations"},
	},
	"ICBP": {
		{From: "ICBP", FromName: "Indofood CBP", To: "INDF", ToName: "Indofood Sukses Makmur", Relationship: "supplier", Strength: 95, Description: "Wheat flour and raw material supply"},
	},
	"TLKM": {
		{From: "TLKM", FromName: "Telkom Indonesia", To: "BBCA", ToName: "Bank Central Asia", Relationship: "customer", Strength: 60, Description: "Digital banking infrastructure"},
		{From: "TLKM", FromName: "Telkom Indonesia", To: "UNVR", ToName: "Unilever Indonesia", Relationship: "customer", Strength: 50, Description: "Enterprise connectivity"},
	},
	"UNTR": {
		{From: "UNTR", FromName: "United Tractors", To: "ADRO", ToName: "Adaro Energy", Relationship: "supplier", Strength: 80, Description: "Mining equipment supplier"},
		{From: "UNTR", FromName: "United Tractors", To: "ASII", ToName: "Astra International", Relationship: "customer", Strength: 85, Description: "Heavy equipment dealer"},
	},
	"ADRO": {
		{From: "ADRO", FromName: "Adaro Energy", To: "UNTR", ToName: "United Tractors", Relationship: "customer", Strength: 80, Description: "Heavy equipment buyer"},
		{From: "ADRO", FromName: "Adaro Energy", To: "BBCA", ToName: "Bank Central Asia", Relationship: "customer", Strength: 85, Description: "Mining project financing"},
	},
	"SMGR": {
		{From: "SMGR", FromName: "Semen Indonesia", To: "BBCA", ToName: "Bank Central Asia", Relationship: "customer", Strength: 75, Description: "Infrastructure project financing"},
	},
}

var concentrationData = map[string]ConcentrationRisk{
	"ADRO": {StockCode: "ADRO", StockName: "Adaro Energy", TopCustomerPct: 40, TopSupplierPct: 30, RiskLevel: "Medium", Detail: "Diversified customer base but heavy reliance on mining equipment suppliers"},
	"INDF": {StockCode: "INDF", StockName: "Indofood Sukses Makmur", TopCustomerPct: 25, TopSupplierPct: 20, RiskLevel: "Low", Detail: "Well-diversified across consumer segments"},
	"ICBP": {StockCode: "ICBP", StockName: "Indofood CBP", TopCustomerPct: 55, TopSupplierPct: 70, RiskLevel: "High", Detail: ">50% revenue from parent company INDF, >50% supply from parent company INDF"},
	"UNTR": {StockCode: "UNTR", StockName: "United Tractors", TopCustomerPct: 35, TopSupplierPct: 45, RiskLevel: "Medium", Detail: "Strong relationship with both Astra as parent and mining customers"},
	"BBCA": {StockCode: "BBCA", StockName: "Bank Central Asia", TopCustomerPct: 15, TopSupplierPct: 10, RiskLevel: "Low", Detail: "Well-diversified portfolio across all economic sectors"},
	"ASII": {StockCode: "ASII", StockName: "Astra International", TopCustomerPct: 20, TopSupplierPct: 30, RiskLevel: "Low", Detail: "Conglomerate structure with diversified business lines"},
	"TLKM": {StockCode: "TLKM", StockName: "Telkom Indonesia", TopCustomerPct: 60, TopSupplierPct: 25, RiskLevel: "High", Detail: "High dependency on government contracts and consumer broadband"},
	"SMGR": {StockCode: "SMGR", StockName: "Semen Indonesia", TopCustomerPct: 30, TopSupplierPct: 35, RiskLevel: "Medium", Detail: "Infrastructure projects drive revenue concentration"},
}

func (s *SupplyChainService) GetSupplyChain(code string) ([]SupplyChainLink, error) {
	code = strings.ToUpper(code)
	links, ok := supplyChainData[code]
	if !ok {
		var allLinks []SupplyChainLink
		for _, links := range supplyChainData {
			for _, l := range links {
				if strings.ToUpper(l.To) == code || strings.ToUpper(l.From) == code {
					allLinks = append(allLinks, l)
				}
			}
		}
		if len(allLinks) == 0 {
			return nil, fmt.Errorf("no supply chain data for %s", code)
		}
		return allLinks, nil
	}
	return links, nil
}

func (s *SupplyChainService) GetAllLinks() ([]SupplyChainLink, error) {
	var allLinks []SupplyChainLink
	seen := make(map[string]bool)
	for _, links := range supplyChainData {
		for _, l := range links {
			key := l.From + "-" + l.To
			if !seen[key] {
				seen[key] = true
				allLinks = append(allLinks, l)
			}
		}
	}
	if allLinks == nil {
		allLinks = []SupplyChainLink{}
	}
	return allLinks, nil
}

func (s *SupplyChainService) AnalyzeConcentration(code string) (*ConcentrationRisk, error) {
	code = strings.ToUpper(code)
	risk, ok := concentrationData[code]
	if !ok {
		return &ConcentrationRisk{
			StockCode:      code,
			StockName:      code,
			TopCustomerPct: 25,
			TopSupplierPct: 25,
			RiskLevel:      "Low",
			Detail:         "Concentration data not available, assuming diversified structure",
		}, nil
	}

	stock, err := s.StockRepo.FindByCode(code)
	if err == nil {
		risk.StockName = stock.Name
	}

	return &risk, nil
}

func (s *SupplyChainService) GetAllConcentrations() ([]ConcentrationRisk, error) {
	var risks []ConcentrationRisk
	for _, r := range concentrationData {
		risks = append(risks, r)
	}
	if risks == nil {
		risks = []ConcentrationRisk{}
	}
	return risks, nil
}
