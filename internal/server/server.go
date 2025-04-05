package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"shm/internal/model"
	"shm/internal/repository"
	"shm/internal/server/middleware"
	"shm/internal/server/request"
	"shm/internal/server/response"
	"strconv"
	"time"
)

type Server struct {
	server *http.Server
	sites  *repository.Sites
}

func New(db *sql.DB, address string) *Server {
	router := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:    address,
			Handler: middleware.Logging(router),
		},
		sites: repository.NewSitesRepo(db),
	}

	router.HandleFunc("GET /site", s.getSites)
	router.HandleFunc("GET /site/{id}", s.getSite)
	router.HandleFunc("POST /site", s.addSite)
	router.HandleFunc("DELETE /site/{id}", s.deleteSite)

	return s
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) getSites(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sites, err := s.sites.GetAllSites(ctx)
	if err != nil {
		slog.Error("failed to get all monitored sites", slog.String("error", err.Error()))
		response.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, sites)
}

func (s *Server) getSite(w http.ResponseWriter, r *http.Request) {
	strId := r.PathValue("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		slog.Error("invalid id", slog.String("error", err.Error()))
		response.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid id"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	site, err := s.sites.GetSiteById(ctx, int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.WriteError(w, http.StatusNotFound, fmt.Errorf("no site with such id"))
			return
		}
		slog.Error(
			"failed to get site by id",
			slog.Int("id", id),
			slog.String("error", err.Error()),
		)
		response.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, site)
}

func (s *Server) addSite(w http.ResponseWriter, r *http.Request) {
	var site model.Site
	if err := request.ReadJSON(r, &site); err != nil {
		slog.Error("invalid site", slog.String("error", err.Error()))
		response.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid site"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.sites.AddSite(ctx, site.Url)
	if err != nil {
		slog.Error("failed to add site", slog.String("error", err.Error()))
		response.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	response.WriteJSON(w, http.StatusNoContent, "")
}

func (s *Server) deleteSite(w http.ResponseWriter, r *http.Request) {
	strId := r.PathValue("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		slog.Error("invalid id", slog.String("error", err.Error()))
		response.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid id"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.sites.DeleteSiteById(ctx, int64(id))
	if err != nil {
		slog.Error(
			"failed to delete site by id",
			slog.Int("id", id),
			slog.String("error", err.Error()),
		)
		response.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	response.WriteJSON(w, http.StatusNoContent, "")
}
