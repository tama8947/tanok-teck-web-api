package handlers

import "github.com/tanok/tanok-web-api/internal/services"

type Handler struct {
	Services *services.Services
}

func NewHandler(svc *services.Services) *Handler {
	return &Handler{Services: svc}
}
