package service

import (
	"math"
	"time"

	"investo/internal/repository"
)

type WashSale struct {
	StockCode      string    `json:"stock_code"`
	SellDate       time.Time `json:"sell_date"`
	SellPrice      float64   `json:"sell_price"`
	SellQty        float64   `json:"sell_qty"`
	BuyDate        time.Time `json:"buy_date"`
	BuyPrice       float64   `json:"buy_price"`
	BuyQty         float64   `json:"buy_qty"`
	LossDisallowed float64   `json:"loss_disallowed"`
	DaysBetween    int       `json:"days_between"`
}

type WashSaleService struct {
	PaperTradingRepo *repository.PaperTradingRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	PortfolioRepo     *repository.PortfolioRepository
}

func NewWashSaleService(
	paperTradingRepo *repository.PaperTradingRepository,
	portfolioItemRepo *repository.PortfolioItemRepository,
	portfolioRepo *repository.PortfolioRepository,
) *WashSaleService {
	return &WashSaleService{
		PaperTradingRepo: paperTradingRepo,
		PortfolioItemRepo: portfolioItemRepo,
		PortfolioRepo:     portfolioRepo,
	}
}

func (s *WashSaleService) DetectWashSales(userID int64) ([]WashSale, error) {
	portfolios, err := s.PortfolioRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	type trade struct {
		stockCode string
		date      time.Time
		price     float64
		qty       float64
		tradeType string
	}

	var allTrades []trade

	for _, portfolio := range portfolios {
		items, err := s.PortfolioItemRepo.FindByPortfolioID(portfolio.ID)
		if err != nil {
			continue
		}
		for _, item := range items {
			if item.Type == "SELL" || item.Type == "sell" {
				allTrades = append(allTrades, trade{
					stockCode: formatStockCodeFromID(item.StockID),
					date:      item.CreatedAt,
					price:     item.AvgPrice,
					qty:       item.Quantity,
					tradeType: "SELL",
				})
			}
			if item.Type == "BUY" || item.Type == "buy" {
				allTrades = append(allTrades, trade{
					stockCode: formatStockCodeFromID(item.StockID),
					date:      item.CreatedAt,
					price:     item.AvgPrice,
					qty:       item.Quantity,
					tradeType: "BUY",
				})
			}
		}
	}

	paperPortfolios, _ := s.PaperTradingRepo.FindPortfoliosByUserID(userID)
	for _, pp := range paperPortfolios {
		paperTrades, err := s.PaperTradingRepo.FindTradesByPortfolioID(pp.ID, 1000)
		if err != nil {
			continue
		}
		for _, pt := range paperTrades {
			allTrades = append(allTrades, trade{
				stockCode: pt.StockCode,
				date:      pt.ExecutedAt,
				price:     pt.Price,
				qty:       pt.Quantity,
				tradeType: pt.Type,
			})
		}
	}

	var washSales []WashSale

	for i := 0; i < len(allTrades); i++ {
		sell := allTrades[i]
		if sell.tradeType != "SELL" {
			continue
		}
		for j := 0; j < len(allTrades); j++ {
			buy := allTrades[j]
			if buy.tradeType != "BUY" {
				continue
			}
			if sell.stockCode != buy.stockCode {
				continue
			}
			if !buy.date.After(sell.date) {
				continue
			}
			daysBetween := int(buy.date.Sub(sell.date).Hours() / 24)
			if daysBetween > 30 {
				continue
			}
			if sell.price <= buy.price {
				continue
			}

			lossPerShare := sell.price - buy.price
			qtyMatched := math.Min(sell.qty, buy.qty)
			lossDisallowed := lossPerShare * qtyMatched

			washSales = append(washSales, WashSale{
				StockCode:      sell.stockCode,
				SellDate:       sell.date,
				SellPrice:      sell.price,
				SellQty:        sell.qty,
				BuyDate:        buy.date,
				BuyPrice:       buy.price,
				BuyQty:         buy.qty,
				LossDisallowed: math.Round(lossDisallowed*100) / 100,
				DaysBetween:    daysBetween,
			})
		}
	}

	return washSales, nil
}

func formatStockCodeFromID(stockID int64) string {
	codes := map[int64]string{
		1: "BBCA", 2: "BBRI", 3: "TLKM", 4: "ASII", 5: "UNVR",
	}
	if code, ok := codes[stockID]; ok {
		return code
	}
	return ""
}
