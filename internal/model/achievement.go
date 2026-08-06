package model

import "time"

type Achievement struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Icon        string     `json:"icon"`
	Category    string     `json:"category"`
	Target      int        `json:"target"`
	Unlocked    bool       `json:"unlocked"`
	Progress    int        `json:"progress"`
	UnlockedAt  *time.Time `json:"unlocked_at"`
}

var PrebuiltAchievements = []Achievement{
	{
		Key:         "newbie_investor",
		Name:        "Newbie Investor",
		Description: "Membuat portofolio pertama",
		Icon:        "🍼",
		Category:    "Pemula",
		Target:      1,
	},
	{
		Key:         "first_trade",
		Name:        "First Trade",
		Description: "Menambahkan saham pertama ke portofolio",
		Icon:        "📈",
		Category:    "Pemula",
		Target:      1,
	},
	{
		Key:         "analyst",
		Name:        "Analyst",
		Description: "Menjalankan screener 10 kali",
		Icon:        "🔍",
		Category:    "Expert",
		Target:      10,
	},
	{
		Key:         "idea_maker",
		Name:        "Idea Maker",
		Description: "Posting 5 ide investasi",
		Icon:        "💡",
		Category:    "Komunitas",
		Target:      5,
	},
	{
		Key:         "top_predictor",
		Name:        "Top Predictor",
		Description: "Masuk top 10 leaderboard prediksi",
		Icon:        "🏆",
		Category:    "Expert",
		Target:      1,
	},
	{
		Key:         "7day_streak",
		Name:        "7-Day Streak",
		Description: "Login 7 hari berturut-turut",
		Icon:        "📅",
		Category:    "Dedikasi",
		Target:      7,
	},
	{
		Key:         "dividend_king",
		Name:        "Dividend King",
		Description: "Mengoleksi 10 dividen dalam portofolio",
		Icon:        "💰",
		Category:    "Expert",
		Target:      10,
	},
	{
		Key:         "sharp_shooter",
		Name:        "Sharp Shooter",
		Description: "Akurasi prediksi > 80% (min 5 prediksi)",
		Icon:        "🎯",
		Category:    "Expert",
		Target:      5,
	},
	{
		Key:         "knowledge_seeker",
		Name:        "Knowledge Seeker",
		Description: "Membaca 20 artikel blog",
		Icon:        "📚",
		Category:    "Dedikasi",
		Target:      20,
	},
	{
		Key:         "investo_master",
		Name:        "Investo Master",
		Description: "Unlock semua achievement lainnya",
		Icon:        "🌟",
		Category:    "Legend",
		Target:      9,
	},
}
