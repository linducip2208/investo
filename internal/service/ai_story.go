package service

import (
	"encoding/json"
	"fmt"
	"time"

	"investo/internal/repository"
)

type StockStory struct {
	Code        string         `json:"code"`
	Title       string         `json:"title"`
	Subtitle    string         `json:"subtitle"`
	Chapters    []StoryChapter `json:"chapters"`
	GeneratedAt string         `json:"generated_at"`
}

type StoryChapter struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Milestone string `json:"milestone"`
}

type AIStoryService struct {
	AI              *AIService
	StockRepo       *repository.StockRepository
	StockPriceRepo  *repository.StockPriceRepository
}

func (s *AIStoryService) GenerateStory(code string) (*StockStory, error) {
	stock, err := s.StockRepo.FindByCode(code)
	if err != nil {
		return nil, fmt.Errorf("stock not found: %w", err)
	}

	aiAvailable := s.AI != nil && s.AI.IsConfigured()

	if aiAvailable {
		result, err := s.generateStoryWithAI(stock.Code, stock.Name)
		if err == nil && result != nil {
			return result, nil
		}
	}

	return s.generateStoryFallback(stock.Code, stock.Name), nil
}

func (s *AIStoryService) generateStoryWithAI(code, name string) (*StockStory, error) {
	prompt := fmt.Sprintf(`Tulis narasi dramatis perjalanan saham %s (%s) dalam format cerita berseri.
Format JSON:
{
  "title": "judul cerita yang dramatis",
  "subtitle": "subtitle menarik",
  "chapters": [
    {"number": 1, "title": "Bab 1 ...", "content": "narasi panjang bab ini", "milestone": "event penting"},
    {"number": 2, "title": "Bab 2 ...", "content": "narasi panjang bab ini", "milestone": "event penting"},
    {"number": 3, "title": "Bab 3 ...", "content": "narasi panjang bab ini", "milestone": "event penting"},
    {"number": 4, "title": "Bab Terakhir ...", "content": "narasi panjang bab ini", "milestone": "event penting"}
  ]
}

Gaya: naratif dramatis seperti novel bisnis, bahasa Indonesia. Bikin pembaca terinspirasi.`, code, name)

	resp, err := s.AI.Chat("Kamu adalah penulis finansial. Tulis narasi dramatis perjalanan sebuah saham. Response HARUS JSON valid.", prompt)
	if err != nil {
		return nil, err
	}

	var result StockStory
	if err := json.Unmarshal([]byte(cleanJSON(resp)), &result); err != nil {
		return nil, fmt.Errorf("parse AI story: %w", err)
	}

	result.Code = code
	result.GeneratedAt = time.Now().Format("02 January 2006")
	return &result, nil
}

func (s *AIStoryService) generateStoryFallback(code, name string) *StockStory {
	return &StockStory{
		Code:     code,
		Title:    fmt.Sprintf("Perjalanan %s: Dari Awal Hingga Kini", code),
		Subtitle: fmt.Sprintf("Kisah perjuangan %s di bursa saham Indonesia", name),
		Chapters: []StoryChapter{
			{
				Number:    1,
				Title:     "Bab 1: Awal yang Sederhana",
				Content:   fmt.Sprintf("%s memulai perjalanannya di Bursa Efek Indonesia dengan langkah yang sederhana. Saat pertama kali IPO, tidak banyak yang menyangka perusahaan ini akan tumbuh sebesar sekarang. Dengan fundamental yang solid dan manajemen yang visioner, %s perlahan membangun reputasi di mata investor.", name, name),
				Milestone: fmt.Sprintf("IPO %s di BEI", code),
			},
			{
				Number:    2,
				Title:     "Bab 2: Melewati Badai Krisis",
				Content:   fmt.Sprintf("Tidak ada perjalanan bisnis yang mulus. %s juga menghadapi masa-masa sulit — mulai dari krisis ekonomi global, perubahan regulasi, hingga pandemi yang mengguncang pasar. Tapi di setiap badai, %s selalu berhasil bertahan. Adaptasi dan inovasi menjadi kunci survival mereka.", name, name),
				Milestone: "Bertahan melewati krisis ekonomi",
			},
			{
				Number:    3,
				Title:     "Bab 3: Ekspansi dan Kebangkitan",
				Content:   fmt.Sprintf("Setelah melewati masa sulit, %s memasuki era ekspansi. Akuisisi strategis, penetrasi pasar baru, dan digitalisasi menjadi mesin pertumbuhan. Investor mulai melirik kembali. Harga saham %s perlahan naik, mencerminkan kepercayaan pasar terhadap prospek jangka panjang perusahaan.", name, code),
				Milestone: "Ekspansi ke pasar baru",
			},
			{
				Number:    4,
				Title:     "Bab Terakhir: Menuju Masa Depan",
				Content:   fmt.Sprintf("Hari ini, %s berdiri sebagai salah satu pemain utama di sektornya. Dengan strategi yang tepat dan eksekusi yang disiplin, masa depan %s terlihat cerah. Bagi investor jangka panjang, %s adalah bukti bahwa kesabaran dan keyakinan pada fundamental yang kuat akan membuahkan hasil. Perjalanan belum berakhir — ini baru permulaan.", name, code, code),
				Milestone: "Visi masa depan",
			},
		},
		GeneratedAt: time.Now().Format("02 January 2006"),
	}
}
