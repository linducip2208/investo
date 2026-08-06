package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"investo/internal/middleware"
	"investo/internal/model"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/go-chi/chi/v5"
)

type CommunityHandler struct {
	CommunityRepo     *repository.CommunityRepository
	PortfolioRepo     *repository.PortfolioRepository
	PortfolioItemRepo *repository.PortfolioItemRepository
	StockRepo         *repository.StockRepository
	Templates         *template.Template
	ChatService       *service.ChatService
}

func (h *CommunityHandler) IdeasList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "latest"
	}
	stockCode := r.URL.Query().Get("stock")
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := 12
	offset := (page - 1) * perPage

	ideas, total, err := h.CommunityRepo.ListIdeas(offset, perPage, sortBy, stockCode)
	if err != nil {
		log.Printf("IdeasList: ListIdeas error: %v", err)
		http.Error(w, "Gagal memuat ide", http.StatusInternalServerError)
		return
	}

	totalPages := (total + perPage - 1) / perPage

	type IdeaWithMeta struct {
		model.InvestmentIdea
		TimeAgo string
	}

	var ideasWithMeta []IdeaWithMeta
	for i := range ideas {
		iw := IdeaWithMeta{InvestmentIdea: ideas[i], TimeAgo: repository.FormatTimeAgo(ideas[i].CreatedAt)}
		if user != nil {
			isLiked, _ := h.CommunityRepo.IsLiked(ideas[i].ID, user.ID)
			iw.IsLiked = isLiked
		}
		ideasWithMeta = append(ideasWithMeta, iw)
	}

	data := map[string]interface{}{
		"Title":      "Ide Investasi - Investo",
		"User":       user,
		"Ideas":      ideasWithMeta,
		"SortBy":     sortBy,
		"StockCode":  stockCode,
		"Page":       page,
		"TotalPages": totalPages,
		"Total":      total,
	}
	if err := h.Templates.ExecuteTemplate(w, "community/ideas.html", data); err != nil {
		log.Printf("IdeasList: ExecuteTemplate error: %v", err)
	}
}

func (h *CommunityHandler) IdeasCreate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	stockCode := strings.TrimSpace(strings.ToUpper(r.FormValue("stock_code")))
	chartImage := strings.TrimSpace(r.FormValue("chart_image"))

	if title == "" || content == "" {
		http.Error(w, "Judul dan konten wajib diisi", http.StatusBadRequest)
		return
	}

	if len(title) > 500 {
		title = title[:500]
	}

	idea := &model.InvestmentIdea{
		UserID:      user.ID,
		Title:       title,
		Content:     content,
		StockCode:   stockCode,
		ChartImage:  chartImage,
		IsPublished: true,
	}

	if _, err := h.CommunityRepo.CreateIdea(idea); err != nil {
		http.Error(w, "Gagal membuat ide", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/ideas", http.StatusSeeOther)
}

func (h *CommunityHandler) IdeasDetail(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	idea, err := h.CommunityRepo.FindIdeaByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	comments, _ := h.CommunityRepo.FindCommentsByIdeaID(id)

	type CommentWithMeta struct {
		model.IdeaComment
		TimeAgo string
	}

	var commentsWithMeta []CommentWithMeta
	for i := range comments {
		commentsWithMeta = append(commentsWithMeta, CommentWithMeta{
			IdeaComment: comments[i],
			TimeAgo:     repository.FormatTimeAgo(comments[i].CreatedAt),
		})
	}

	isLiked := false
	if user != nil {
		isLiked, _ = h.CommunityRepo.IsLiked(id, user.ID)
	}

	data := map[string]interface{}{
		"Title":    idea.Title + " - Ide Investasi - Investo",
		"User":     user,
		"Idea":     idea,
		"Comments": commentsWithMeta,
		"IsLiked":  isLiked,
		"TimeAgo":  repository.FormatTimeAgo(idea.CreatedAt),
	}
	h.Templates.ExecuteTemplate(w, "community/idea-detail.html", data)
}

func (h *CommunityHandler) IdeasDelete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.CommunityRepo.DeleteIdea(id, user.ID); err != nil {
		http.Error(w, "Gagal menghapus ide", http.StatusForbidden)
		return
	}

	http.Redirect(w, r, "/ideas", http.StatusSeeOther)
}

func (h *CommunityHandler) IdeasLike(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid idea ID"})
		return
	}

	liked, err := h.CommunityRepo.LikeIdea(id, user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	idea, _ := h.CommunityRepo.FindIdeaByID(id)
	likesCount := 0
	if idea != nil {
		likesCount = idea.LikesCount
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"liked":       liked,
		"likes_count": likesCount,
	})
}

func (h *CommunityHandler) IdeasComment(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Redirect(w, r, "/ideas/"+idStr, http.StatusSeeOther)
		return
	}

	comment := &model.IdeaComment{
		IdeaID:  id,
		UserID:  user.ID,
		Content: content,
	}

	if _, err := h.CommunityRepo.CreateComment(comment); err != nil {
		http.Error(w, "Gagal menambahkan komentar", http.StatusInternalServerError)
		return
	}

	h.CommunityRepo.IncrementCommentsCount(id)

	http.Redirect(w, r, "/ideas/"+idStr, http.StatusSeeOther)
}

func (h *CommunityHandler) IdeasCommentDelete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ideaIDStr := chi.URLParam(r, "id")
	commentIDStr := chi.URLParam(r, "commentId")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.CommunityRepo.DeleteComment(commentID, user.ID); err != nil {
		http.Error(w, "Gagal menghapus komentar", http.StatusForbidden)
		return
	}

	http.Redirect(w, r, "/ideas/"+ideaIDStr, http.StatusSeeOther)
}

func (h *CommunityHandler) DiscussionList(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	code := chi.URLParam(r, "code")

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := 20
	offset := (page - 1) * perPage

	discussions, total, err := h.CommunityRepo.FindDiscussionsByStock(code, offset, perPage)
	if err != nil {
		http.Error(w, "Gagal memuat diskusi", http.StatusInternalServerError)
		return
	}

	totalPages := (total + perPage - 1) / perPage

	type DiscussionWithMeta struct {
		model.StockDiscussion
		TimeAgo string
	}

	var discussionsWithMeta []DiscussionWithMeta
	for i := range discussions {
		discussionsWithMeta = append(discussionsWithMeta, DiscussionWithMeta{
			StockDiscussion: discussions[i],
			TimeAgo:         repository.FormatTimeAgo(discussions[i].CreatedAt),
		})
	}

	stock, _ := h.StockRepo.FindByCode(code)

	data := map[string]interface{}{
		"Title":        "Diskusi " + code + " - Investo",
		"User":         user,
		"StockCode":    code,
		"Stock":        stock,
		"Discussions":  discussionsWithMeta,
		"Page":         page,
		"TotalPages":   totalPages,
		"Total":        total,
	}
	h.Templates.ExecuteTemplate(w, "community/discussion.html", data)
}

func (h *CommunityHandler) DiscussionCreate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	code := chi.URLParam(r, "code")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Redirect(w, r, "/saham/"+code+"/diskusi", http.StatusSeeOther)
		return
	}

	msg := &model.StockDiscussion{
		StockCode: code,
		UserID:    user.ID,
		Content:   content,
	}

	if _, err := h.CommunityRepo.CreateDiscussion(msg); err != nil {
		http.Error(w, "Gagal mengirim pesan", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/saham/"+code+"/diskusi", http.StatusSeeOther)
}

func (h *CommunityHandler) SharePortfolio(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(id)
	if err != nil || portfolio.UserID != user.ID {
		http.NotFound(w, r)
		return
	}

	existing, _ := h.CommunityRepo.FindShareByPortfolioID(id, user.ID)
	if existing != nil {
		http.Redirect(w, r, "/dashboard/portfolios/"+idStr, http.StatusSeeOther)
		return
	}

	share := &model.SharedPortfolio{
		PortfolioID: id,
		UserID:      user.ID,
		IsActive:    true,
	}

	if _, err := h.CommunityRepo.CreateShare(share); err != nil {
		http.Error(w, "Gagal membagikan portfolio", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard/portfolios/"+idStr, http.StatusSeeOther)
}

func (h *CommunityHandler) ViewSharedPortfolio(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	share, err := h.CommunityRepo.FindShareByToken(token)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	h.CommunityRepo.IncrementShareViews(share.ID)

	portfolio, err := h.PortfolioRepo.FindByID(share.PortfolioID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	items, _ := h.PortfolioItemRepo.GetWithStock(share.PortfolioID)

	type SharedViewItem struct {
		Code             string
		Name             string
		Quantity         float64
		AvgPrice         float64
		AvgPriceFormatted string
		Notes            string
	}

	var holdings []SharedViewItem
	for _, item := range items {
		holdings = append(holdings, SharedViewItem{
			Code:             item.Stock.Code,
			Name:             item.Stock.Name,
			Quantity:         item.Item.Quantity,
			AvgPrice:         item.Item.AvgPrice,
			AvgPriceFormatted: formatRupiah(int64(item.Item.AvgPrice)),
			Notes:            item.Item.Notes,
		})
	}

	data := map[string]interface{}{
		"Title":     portfolio.Name + " - Portfolio Dibagikan - Investo",
		"User":  safeUser(middleware.GetUser(r)),
		"Portfolio": portfolio,
		"Holdings":  holdings,
		"Share":     share,
	}
	h.Templates.ExecuteTemplate(w, "portfolio/shared.html", data)
}

func (h *CommunityHandler) UnsharePortfolio(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	portfolio, err := h.PortfolioRepo.FindByID(id)
	if err != nil || portfolio.UserID != user.ID {
		http.NotFound(w, r)
		return
	}

	share, err := h.CommunityRepo.FindShareByPortfolioID(id, user.ID)
	if err != nil {
		http.Redirect(w, r, "/dashboard/portfolios/"+idStr, http.StatusSeeOther)
		return
	}

	h.CommunityRepo.DeactivateShare(share.ID, user.ID)

	http.Redirect(w, r, "/dashboard/portfolios/"+idStr, http.StatusSeeOther)
}

func (h *CommunityHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	entries, err := h.CommunityRepo.GetLeaderboard(20)
	if err != nil {
		http.Error(w, "Gagal memuat leaderboard", http.StatusInternalServerError)
		return
	}

	for i := range entries {
		if len(entries[i].UserName) > 2 {
			name := []rune(entries[i].UserName)
			masked := string(name[0])
			for j := 1; j < len(name)-1; j++ {
				masked += "*"
			}
			masked += string(name[len(name)-1])
			entries[i].UserName = masked
		}
	}

	data := map[string]interface{}{
		"Title":   "Top Investor - Leaderboard - Investo",
		"User":    user,
		"Entries": entries,
	}
	h.Templates.ExecuteTemplate(w, "community/leaderboard.html", data)
}

func formatRupiah(n int64) string {
	s := strconv.FormatInt(n, 10)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return "Rp " + string(result)
}

func (h *CommunityHandler) ChatPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rooms, _ := h.ChatService.GetActiveStockRooms()

	data := map[string]interface{}{
		"Title": "Chat Komunitas - Investo",
		"User":  user,
		"Rooms": rooms,
	}
	h.Templates.ExecuteTemplate(w, "community/chat.html", data)
}

func (h *CommunityHandler) ChatSendMessage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	var payload struct {
		Message    string  `json:"message"`
		Room       *string `json:"room"`
		ReceiverID *int64  `json:"receiver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"}, nil)
		return
	}

	if payload.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message required"}, nil)
		return
	}

	msg, err := h.ChatService.SendMessage(user.ID, payload.ReceiverID, payload.Room, payload.Message)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}, nil)
		return
	}

	writeJSON(w, http.StatusOK, msg, nil)
}

func (h *CommunityHandler) ChatRoomMessages(w http.ResponseWriter, r *http.Request) {
	room := chi.URLParam(r, "room")
	if room == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room required"}, nil)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := h.ChatService.GetRoomMessages(room, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}, nil)
		return
	}

	if messages == nil {
		messages = []model.ChatMessage{}
	}

	writeJSON(w, http.StatusOK, messages, nil)
}

func (h *CommunityHandler) DMMessages(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	otherIDStr := chi.URLParam(r, "userID")
	otherID, err := strconv.ParseInt(otherIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"}, nil)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := h.ChatService.GetDMMessages(user.ID, otherID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}, nil)
		return
	}

	if messages == nil {
		messages = []model.ChatMessage{}
	}

	writeJSON(w, http.StatusOK, messages, nil)
}

func (h *CommunityHandler) DMConversationsPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	conversations, _ := h.ChatService.GetDMConversations(user.ID)

	data := map[string]interface{}{
		"Title":         "Pesan - Investo",
		"User":          user,
		"Conversations": conversations,
	}
	h.Templates.ExecuteTemplate(w, "community/chat.html", data)
}

func (h *CommunityHandler) FollowUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	followingIDStr := chi.URLParam(r, "id")
	followingID, err := strconv.ParseInt(followingIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"}, nil)
		return
	}

	if followingID == user.ID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tidak bisa follow diri sendiri"}, nil)
		return
	}

	if err := h.ChatService.Follow(user.ID, followingID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}, nil)
		return
	}

	followersCount, _ := h.ChatService.GetFollowersCount(followingID)
	followingCount, _ := h.ChatService.GetFollowingCount(user.ID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"followers_count": followersCount,
		"following_count": followingCount,
	}, nil)
}

func (h *CommunityHandler) UnfollowUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	followingIDStr := chi.URLParam(r, "id")
	followingID, err := strconv.ParseInt(followingIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"}, nil)
		return
	}

	if err := h.ChatService.Unfollow(user.ID, followingID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}, nil)
		return
	}

	followersCount, _ := h.ChatService.GetFollowersCount(followingID)
	followingCount, _ := h.ChatService.GetFollowingCount(user.ID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"followers_count": followersCount,
		"following_count": followingCount,
	}, nil)
}

func (h *CommunityHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"}, nil)
		return
	}

	followers, err := h.ChatService.GetFollowers(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}, nil)
		return
	}

	if followers == nil {
		followers = []model.User{}
	}

	writeJSON(w, http.StatusOK, followers, nil)
}

func (h *CommunityHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"}, nil)
		return
	}

	following, err := h.ChatService.GetFollowing(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}, nil)
		return
	}

	if following == nil {
		following = []model.User{}
	}

	writeJSON(w, http.StatusOK, following, nil)
}

func (h *CommunityHandler) DMConversationsJSON(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"}, nil)
		return
	}

	conversations, err := h.ChatService.GetDMConversations(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()}, nil)
		return
	}

	if conversations == nil {
		conversations = []model.DMConversation{}
	}

	writeJSON(w, http.StatusOK, conversations, nil)
}
