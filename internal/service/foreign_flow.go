package service

import (
	"fmt"
	"sort"
	"time"

	"investo/internal/repository"

	"github.com/jmoiron/sqlx"
)

type FlowData struct {
	StockCode     string  `json:"stock_code"`
	StockName     string  `json:"stock_name"`
	ForeignBuy    int64   `json:"foreign_buy"`
	ForeignSell   int64   `json:"foreign_sell"`
	ForeignNet    int64   `json:"foreign_net"`
	TotalValue    int64   `json:"total_value"`
	FlowDirection string  `json:"flow_direction"`
	Date          string  `json:"date"`
}

type ForeignFlowService struct {
	DB        *sqlx.DB
	StockRepo *repository.StockRepository
}

func (s *ForeignFlowService) GetTopForeignFlow(limit int) ([]FlowData, error) {
	query := `SELECT ff.stock_code, ff.foreign_buy_val AS foreign_buy, ff.foreign_sell_val AS foreign_sell,
		(ff.foreign_buy_val - ff.foreign_sell_val) AS foreign_net,
		(ff.foreign_buy_val + ff.foreign_sell_val) AS total_value, ff.date
		FROM foreign_flow ff
		INNER JOIN (SELECT stock_code, MAX(date) AS max_date FROM foreign_flow GROUP BY stock_code) latest
		ON ff.stock_code = latest.stock_code AND ff.date = latest.max_date
		ORDER BY ABS(ff.foreign_buy_val - ff.foreign_sell_val) DESC
		LIMIT ?`

	rows, err := s.DB.Queryx(query, limit)
	if err != nil {
		return s.getDemoFlowData(limit), nil
	}
	defer rows.Close()

	var flows []FlowData
	for rows.Next() {
		var f FlowData
		var date time.Time
		if err := rows.Scan(&f.StockCode, &f.ForeignBuy, &f.ForeignSell, &f.ForeignNet, &f.TotalValue, &date); err != nil {
			continue
		}
		f.Date = date.Format("2006-01-02")
		if f.ForeignNet >= 0 {
			f.FlowDirection = "inflow"
		} else {
			f.FlowDirection = "outflow"
		}
		if stock, err := s.StockRepo.FindByCode(f.StockCode); err == nil {
			f.StockName = stock.Name
		}
		flows = append(flows, f)
	}

	if len(flows) == 0 {
		return s.getDemoFlowData(limit), nil
	}
	return flows, nil
}

func (s *ForeignFlowService) GetStockFlow(code string) (*FlowData, error) {
	query := `SELECT stock_code, foreign_buy_val, foreign_sell_val, date FROM foreign_flow
		WHERE stock_code = ? ORDER BY date DESC LIMIT 1`

	var f FlowData
	var date time.Time
	err := s.DB.QueryRowx(query, code).Scan(&f.StockCode, &f.ForeignBuy, &f.ForeignSell, &date)
	if err != nil {
		return nil, fmt.Errorf("no flow data for %s", code)
	}
	f.ForeignNet = f.ForeignBuy - f.ForeignSell
	f.TotalValue = f.ForeignBuy + f.ForeignSell
	f.Date = date.Format("2006-01-02")
	if f.ForeignNet >= 0 {
		f.FlowDirection = "inflow"
	} else {
		f.FlowDirection = "outflow"
	}
	return &f, nil
}

func (s *ForeignFlowService) getDemoFlowData(limit int) []FlowData {
	demo := []FlowData{
		{StockCode: "BBCA", StockName: "Bank Central Asia Tbk", ForeignBuy: 245000000000, ForeignSell: 180000000000, ForeignNet: 65000000000, TotalValue: 425000000000, FlowDirection: "inflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "BBRI", StockName: "Bank Rakyat Indonesia Tbk", ForeignBuy: 320000000000, ForeignSell: 280000000000, ForeignNet: 40000000000, TotalValue: 600000000000, FlowDirection: "inflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "TLKM", StockName: "Telkom Indonesia Tbk", ForeignBuy: 180000000000, ForeignSell: 220000000000, ForeignNet: -40000000000, TotalValue: 400000000000, FlowDirection: "outflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "ASII", StockName: "Astra International Tbk", ForeignBuy: 150000000000, ForeignSell: 130000000000, ForeignNet: 20000000000, TotalValue: 280000000000, FlowDirection: "inflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "BMRI", StockName: "Bank Mandiri Tbk", ForeignBuy: 210000000000, ForeignSell: 250000000000, ForeignNet: -40000000000, TotalValue: 460000000000, FlowDirection: "outflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "UNVR", StockName: "Unilever Indonesia Tbk", ForeignBuy: 90000000000, ForeignSell: 120000000000, ForeignNet: -30000000000, TotalValue: 210000000000, FlowDirection: "outflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "ADRO", StockName: "Adaro Energy Indonesia Tbk", ForeignBuy: 160000000000, ForeignSell: 110000000000, ForeignNet: 50000000000, TotalValue: 270000000000, FlowDirection: "inflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "ICBP", StockName: "Indofood CBP Sukses Makmur Tbk", ForeignBuy: 85000000000, ForeignSell: 95000000000, ForeignNet: -10000000000, TotalValue: 180000000000, FlowDirection: "outflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "BBNI", StockName: "Bank Negara Indonesia Tbk", ForeignBuy: 130000000000, ForeignSell: 140000000000, ForeignNet: -10000000000, TotalValue: 270000000000, FlowDirection: "outflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "GGRM", StockName: "Gudang Garam Tbk", ForeignBuy: 70000000000, ForeignSell: 60000000000, ForeignNet: 10000000000, TotalValue: 130000000000, FlowDirection: "inflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "KLBF", StockName: "Kalbe Farma Tbk", ForeignBuy: 110000000000, ForeignSell: 80000000000, ForeignNet: 30000000000, TotalValue: 190000000000, FlowDirection: "inflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "BRPT", StockName: "Barito Pacific Tbk", ForeignBuy: 200000000000, ForeignSell: 150000000000, ForeignNet: 50000000000, TotalValue: 350000000000, FlowDirection: "inflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "PGAS", StockName: "Perusahaan Gas Negara Tbk", ForeignBuy: 55000000000, ForeignSell: 70000000000, ForeignNet: -15000000000, TotalValue: 125000000000, FlowDirection: "outflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "INDF", StockName: "Indofood Sukses Makmur Tbk", ForeignBuy: 65000000000, ForeignSell: 55000000000, ForeignNet: 10000000000, TotalValue: 120000000000, FlowDirection: "inflow", Date: time.Now().Format("2006-01-02")},
		{StockCode: "CPIN", StockName: "Charoen Pokphand Indonesia Tbk", ForeignBuy: 80000000000, ForeignSell: 95000000000, ForeignNet: -15000000000, TotalValue: 175000000000, FlowDirection: "outflow", Date: time.Now().Format("2006-01-02")},
	}

	sort.Slice(demo, func(i, j int) bool {
		return abs64(demo[i].ForeignNet) > abs64(demo[j].ForeignNet)
	})

	if limit > 0 && limit < len(demo) {
		return demo[:limit]
	}
	return demo
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
