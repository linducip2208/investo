package service

import (
	"strconv"
	"strings"
)

type NLParser struct{}

func (p *NLParser) Parse(query string) (ScreenerCriteria, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	criteria := ScreenerCriteria{}

	numberAfter := func(word string) (float64, bool) {
		idx := strings.Index(q, word)
		if idx < 0 {
			return 0, false
		}
		rest := q[idx+len(word):]
		rest = strings.TrimSpace(rest)
		rest = strings.TrimPrefix(rest, "=")
		rest = strings.TrimSpace(rest)
		rest = strings.TrimPrefix(rest, "<")
		rest = strings.TrimSpace(rest)
		rest = strings.TrimPrefix(rest, ">")
		rest = strings.TrimSpace(rest)

		perc := false
		if strings.HasSuffix(rest, "%") || strings.Contains(rest, "persen") || strings.Contains(rest, "percent") {
			perc = true
		}

		field := strings.Fields(rest)
		for _, f := range field {
			f = strings.TrimSuffix(f, "%")
			f = strings.TrimSuffix(f, "x")
			v, err := strconv.ParseFloat(f, 64)
			if err == nil {
				if perc {
					return v, true
				}
				return v, true
			}
		}
		return 0, false
	}

	hasWord := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(q, w) {
				return true
			}
		}
		return false
	}

	_ = hasWord

	// PER
	if strings.Contains(q, "p/e") || strings.Contains(q, "per") {
		if v, ok := numberAfter("p/e"); ok {
			setPERRange(&criteria, q, v)
		} else if v, ok := numberAfter("per "); ok {
			setPERRange(&criteria, q, v)
		} else if v, ok := numberAfter("per<"); ok {
			criteria.MaxPER = v
		} else if v, ok := numberAfter("per <"); ok {
			criteria.MaxPER = v
		}
	}

	if strings.Contains(q, "pbv") || strings.Contains(q, "p/b") {
		if v, ok := numberAfter("p/b "); ok {
			setPBVRange(&criteria, q, v)
		} else if v, ok := numberAfter("pbv "); ok {
			setPBVRange(&criteria, q, v)
		} else if v, ok := numberAfter("pbv<"); ok {
			criteria.MaxPBV = v
		} else if v, ok := numberAfter("pbv <"); ok {
			criteria.MaxPBV = v
		} else if v, ok := numberAfter("p/b<"); ok {
			criteria.MaxPBV = v
		}
	}

	if strings.Contains(q, "roe") {
		if v, ok := numberAfter("roe "); ok {
			if strings.Contains(q, "di bawah") || strings.Contains(q, "<") {
				criteria.MaxROE = v
			} else {
				criteria.MinROE = v
			}
		} else if v, ok := numberAfter("roe>"); ok {
			criteria.MinROE = v
		}
	}

	if strings.Contains(q, "der") {
		if v, ok := numberAfter("der "); ok {
			if strings.Contains(q, "di bawah") || strings.Contains(q, "<") {
				criteria.MaxDER = v
			} else {
				criteria.MinDER = v
			}
		} else if v, ok := numberAfter("der<"); ok {
			criteria.MaxDER = v
		}
	}

	// dividend
	if strings.Contains(q, "dividen") || strings.Contains(q, "dividend") {
		if v, ok := numberAfter("dividen "); ok {
			criteria.MinDivYield = v
		} else if v, ok := numberAfter("dividend "); ok {
			criteria.MinDivYield = v
		}
	}

	// sectors
	sectorMap := map[string]int64{
		"bank": 8, "perbankan": 8,
		"tambang": 3, "mining": 3,
		"teknologi": 7, "tech": 7, "technology": 7,
		"kesehatan": 6, "healthcare": 6, "health": 6,
		"properti": 9, "property": 9,
		"konsumen": 1, "consumer": 1,
		"infrastruktur": 11, "infrastructure": 11,
		"energi": 3, "energy": 3,
	}
	for key, sid := range sectorMap {
		if strings.Contains(q, key) {
			criteria.SectorID = sid
			break
		}
	}

	// market cap
	if hasWord("big cap", "large cap", "big-cap", "large-cap") {
		criteria.MarketCapMin = 100_000_000_000_000
	} else if hasWord("mid cap", "medium cap", "mid-cap", "medium-cap") {
		criteria.MarketCapMin = 5_000_000_000_000
		criteria.MarketCapMax = 100_000_000_000_000
	} else if hasWord("small cap", "small-cap") {
		criteria.MarketCapMax = 5_000_000_000_000
	}

	// value / murah
	if hasWord("value", "murah", "cheap") {
		if criteria.MaxPER == 0 {
			criteria.MaxPER = 15
		}
		if criteria.MaxPBV == 0 {
			criteria.MaxPBV = 1.5
		}
	}

	// growth / tumbuh
	if hasWord("growth", "tumbuh") {
		if criteria.MinROE == 0 {
			criteria.MinROE = 15
		}
	}

	// expensive / mahal
	if hasWord("mahal", "expensive") {
		if criteria.MinPER == 0 {
			criteria.MinPER = 25
		}
	}

	criteria.SortBy = "score"
	criteria.SortOrder = "desc"

	return criteria, nil
}

func setPERRange(c *ScreenerCriteria, q string, v float64) {
	if strings.Contains(q, "per<") || strings.Contains(q, "per <") || strings.Contains(q, "per di bawah") || strings.Contains(q, "per kurang") {
		c.MaxPER = v
	} else if strings.Contains(q, "per>") || strings.Contains(q, "per >") || strings.Contains(q, "per di atas") || strings.Contains(q, "per lebih") {
		c.MinPER = v
	} else {
		c.MaxPER = v
	}
}

func setPBVRange(c *ScreenerCriteria, q string, v float64) {
	if strings.Contains(q, "pbv<") || strings.Contains(q, "pbv <") || strings.Contains(q, "pbv di bawah") || strings.Contains(q, "pbv kurang") || strings.Contains(q, "p/b<") {
		c.MaxPBV = v
	} else if strings.Contains(q, "pbv>") || strings.Contains(q, "pbv >") || strings.Contains(q, "pbv di atas") || strings.Contains(q, "pbv lebih") {
		c.MinPBV = v
	} else {
		c.MaxPBV = v
	}
}
