package handler

import "investo/internal/model"

func safeUser(u *model.User) *model.User {
	if u == nil {
		return &model.User{Name: "Guest", Email: "", Role: "visitor"}
	}
	return u
}
