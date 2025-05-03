package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"shm/internal/config"
	"shm/internal/lib/sl"
	"shm/internal/model"
	"shm/internal/server/middleware"
	"shm/internal/server/request"
	"shm/internal/server/response"
	"shm/internal/service"
	"strconv"
)

type Server struct {
	server *http.Server
	sites  *service.SitesService
	config config.ServerConfig
}

func New(sites *service.SitesService, config config.ServerConfig) *Server {
	router := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:    config.Address,
			Handler: middleware.Logging(router),
		},
		sites:  sites,
		config: config,
	}

	router.HandleFunc("GET /sites", s.getSites)
	router.HandleFunc("GET /sites/{id}", s.getSite)
	router.HandleFunc("POST /sites", s.addSite)
	router.HandleFunc("DELETE /sites/{id}", s.deleteSite)

	return s
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) getSites(w http.ResponseWriter, r *http.Request) {
	sites, err := s.sites.GetAllSites(context.Background())
	if err != nil {
		slog.Error("failed to get all monitored sites", sl.Error(err))
		response.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, sites)
}

func (s *Server) getSite(w http.ResponseWriter, r *http.Request) {
	strId := r.PathValue("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		slog.Error("invalid id", sl.Error(err))
		response.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid id"))
		return
	}

	site, err := s.sites.GetSiteById(context.Background(), int64(id))
	if err != nil {
		slog.Error("failed to get site by id", slog.Int("id", id), sl.Error(err))
		response.WriteError(w, http.StatusInternalServerError, err)
		return
	} else if site == nil {
		response.WriteError(w, http.StatusNotFound, fmt.Errorf("no site with such id"))
		return
	}

	response.WriteJSON(w, http.StatusOK, site)
}

func (s *Server) addSite(w http.ResponseWriter, r *http.Request) {
	var site model.Site
	if err := request.ReadJSON(r, &site); err != nil {
		slog.Error("invalid site", sl.Error(err))
		response.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid site"))
		return
	}

	err := s.sites.AddSite(context.Background(), site.Url)
	if err != nil {
		slog.Error("failed to add site", sl.Error(err))
		response.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	response.WriteJSON(w, http.StatusNoContent, "")
}

func (s *Server) deleteSite(w http.ResponseWriter, r *http.Request) {
	strId := r.PathValue("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		slog.Error("invalid id", sl.Error(err))
		response.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid id"))
		return
	}

	err = s.sites.DeleteSiteById(context.Background(), int64(id))
	if err != nil {
		slog.Error(
			"failed to delete site by id",
			slog.Int("id", id),
			sl.Error(err),
		)
		response.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	response.WriteJSON(w, http.StatusNoContent, "")
}
