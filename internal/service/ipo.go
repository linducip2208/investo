package service

import (
	"math"
	"time"
)

type IPOEvent struct {
	StockCode         string  `json:"stock_code"`
	Name              string  `json:"name"`
	Sector            string  `json:"sector"`
	OfferPrice        float64 `json:"offer_price"`
	OfferShares       int64   `json:"offer_shares"`
	BookBuildingStart string  `json:"book_building_start"`
	BookBuildingEnd   string  `json:"book_building_end"`
	ListingDate       string  `json:"listing_date"`
	FirstDayClose     float64 `json:"first_day_close"`
	ReturnPercent     float64 `json:"return_percent"`
	IsIllustrative  bool    `json:"is_illustrative"`
	Source          string  `json:"source,omitempty"`
	Note            string  `json:"note,omitempty"`
}

type IPOService struct{}

func (s *IPOService) GetUpcomingIPOs() []IPOEvent {
	now := time.Now()
	future := now.AddDate(0, 0, 14)
	future2 := now.AddDate(0, 0, 30)
	future3 := now.AddDate(0, 0, 45)

	// NOTE: GOTO (listing 2022) dan PGEO/PERT (listing 2023) sudah tercatat
	// di bursa sehingga dipindah ke GetRecentIPOs. Tanpa feed jadwal IPO
	// resmi, sisa entri di bawah adalah contoh ilustratif.
	note := "Data ilustratif — verifikasi jadwal resmi di keterbukaan informasi BEI/IDX"
	return []IPOEvent{
		{
			StockCode: "MEDI", Name: "Medikaloka Investama Tbk", Sector: "Kesehatan",
			OfferPrice: 450, OfferShares: 8000000000,
			BookBuildingStart: now.Format("02 Jan 2006"), BookBuildingEnd: future.Format("02 Jan 2006"),
			ListingDate: future2.Format("02 Jan 2006"),
			IsIllustrative: true, Source: "ilustratif", Note: note,
		},
		{
			StockCode: "SUNI", Name: "Sunindo Pratama Tbk", Sector: "Industri",
			OfferPrice: 250, OfferShares: 5000000000,
			BookBuildingStart: future.Format("02 Jan 2006"), BookBuildingEnd: future2.Format("02 Jan 2006"),
			ListingDate: future3.Format("02 Jan 2006"),
			IsIllustrative: true, Source: "ilustratif", Note: note,
		},
	}
}

func (s *IPOService) GetRecentIPOs(days int) []IPOEvent {
	past := time.Now().AddDate(0, 0, -days)
	_ = past

	note := "Data ilustratif — contoh, bukan data real-time"
	return []IPOEvent{
		{
			StockCode: "BUKA", Name: "Bukalapak.com Tbk", Sector: "Teknologi",
			OfferPrice: 850, OfferShares: 25000000000,
			BookBuildingStart: "01 Jul 2021", BookBuildingEnd: "15 Jul 2021",
			ListingDate: "06 Agu 2021", FirstDayClose: 870, ReturnPercent: 2.35,
			IsIllustrative: true, Source: "ilustratif", Note: note,
		},
		{
			StockCode: "GOTO", Name: "GoTo Gojek Tokopedia Tbk", Sector: "Teknologi",
			OfferPrice: 338, OfferShares: 48000000000,
			BookBuildingStart: "01 Apr 2022", BookBuildingEnd: "07 Apr 2022",
			ListingDate: "11 Apr 2022", FirstDayClose: 338, ReturnPercent: 0,
			IsIllustrative: true, Source: "ilustratif", Note: note + " (sudah tercatat di BEI sejak 2022)",
		},
		{
			StockCode: "PGEO", Name: "Pertamina Geothermal Energy Tbk", Sector: "Energi",
			OfferPrice: 875, OfferShares: 15000000000,
			BookBuildingStart: "01 Feb 2023", BookBuildingEnd: "16 Feb 2023",
			ListingDate: "24 Feb 2023", FirstDayClose: 875, ReturnPercent: 0,
			IsIllustrative: true, Source: "ilustratif", Note: note + " (sudah tercatat di BEI sejak 2023)",
		},
		{
			StockCode: "MTEL", Name: "Dayamitra Telekomunikasi Tbk", Sector: "Infrastruktur",
			OfferPrice: 700, OfferShares: 12000000000,
			BookBuildingStart: "15 Jun 2026", BookBuildingEnd: "28 Jun 2026",
			ListingDate: "15 Jul 2026", FirstDayClose: 740, ReturnPercent: 5.71,
			IsIllustrative: true, Source: "ilustratif", Note: note,
		},
		{
			StockCode: "ARTO", Name: "Bank Jago Tbk", Sector: "Keuangan",
			OfferPrice: 3500, OfferShares: 3000000000,
			BookBuildingStart: "01 Jun 2026", BookBuildingEnd: "14 Jun 2026",
			ListingDate: "30 Jun 2026", FirstDayClose: 4200, ReturnPercent: 20.0,
			IsIllustrative: true, Source: "ilustratif", Note: note,
		},
		{
			StockCode: "WIFI", Name: "PT Solusi Sinergi Digital Tbk", Sector: "Teknologi",
			OfferPrice: 100, OfferShares: 20000000000,
			BookBuildingStart: "01 Feb 2026", BookBuildingEnd: "14 Feb 2026",
			ListingDate: "25 Feb 2026", FirstDayClose: 134, ReturnPercent: 34.0,
			IsIllustrative: true, Source: "ilustratif", Note: note,
		},
	}
}

func (s *IPOService) FormatReturnPercent(v float64) string {
	if v >= 0 {
		return "+" + formatFloat2(v) + "%"
	}
	return formatFloat2(v) + "%"
}

func formatFloat2(v float64) string {
	v = math.Round(v*100) / 100
	return formatFloatV(v)
}

func formatFloatV(v float64) string {
	s := ""
	if v < 0 {
		s = "-"
		v = -v
	}
	whole := int64(v)
	frac := int64(math.Round((v - float64(whole)) * 100))
	if frac == 0 {
		return s + formatInt2(whole)
	}
	return s + formatInt2(whole) + "." + formatInt2Pad(frac)
}

func formatInt2(n int64) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return formatInt2(n/10) + string(rune('0'+n%10))
}

func formatInt2Pad(n int64) string {
	s := formatInt2(n)
	if len(s) < 2 {
		return "0" + s
	}
	return s
}
