package api

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/query"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type queryRequest struct {
	Targets []struct {
		Target string `json:"target"`
	} `json:"targets"`
	Range struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"range"`
}

type timeSeriesResponse struct {
	Target     string      `json:"target"`
	Datapoints [][]float64 `json:"datapoints"`
}

type legacyHTTPHandler struct {
	querier query.Querier
	cfg     *config.Config
}

// RunLegacyHTTPServer starts the legacy HTTP/JSON API server and blocks until shutdown.
func RunLegacyHTTPServer(ctx context.Context, cfg *config.Config) error {
	querier, err := newExactQuerier(cfg)
	if err != nil {
		return err
	}

	handler := &legacyHTTPHandler{querier: querier, cfg: cfg}
	router := mux.NewRouter()
	router.HandleFunc("/", handler.healthCheckHandler).Methods(http.MethodGet)
	router.HandleFunc("/search", handler.searchHandler).Methods(http.MethodPost)
	router.HandleFunc("/query", handler.queryHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/aggregate", handler.aggregateFlowsHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/flows/trace", handler.traceFlowHandler).Methods(http.MethodPost)

	server := &http.Server{
		Addr:    cfg.API.HTTPListenAddr,
		Handler: router,
	}

	return runHTTPServer(ctx, server, fmt.Sprintf("API server starting on %s", server.Addr))
}

func (h *legacyHTTPHandler) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *legacyHTTPHandler) searchHandler(w http.ResponseWriter, r *http.Request) {
	taskNames := make([]string, 0, len(h.cfg.Aggregator.Exact.Tasks))
	for _, task := range h.cfg.Aggregator.Exact.Tasks {
		taskNames = append(taskNames, task.Name)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(taskNames); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode search response: %v", err), http.StatusInternalServerError)
	}
}

func (h *legacyHTTPHandler) queryHandler(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endTime := req.Range.To
	if endTime.IsZero() {
		endTime = time.Now().Add(24 * time.Hour)
	}

	response := make([]timeSeriesResponse, 0, len(req.Targets))
	for _, target := range req.Targets {
		aggReq := &v1.AggregationRequest{
			EndTime:  timestamppb.New(endTime),
			TaskName: target.Target,
		}

		aggResp, err := h.querier.AggregateFlows(r.Context(), aggReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var totalPackets float64
		if len(aggResp.Summaries) > 0 {
			totalPackets = float64(aggResp.Summaries[0].TotalPackets)
		}

		response = append(response, timeSeriesResponse{
			Target: target.Target,
			Datapoints: [][]float64{
				{totalPackets, float64(endTime.Unix() * 1000)},
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode query response: %v", err), http.StatusInternalServerError)
	}
}

func (h *legacyHTTPHandler) aggregateFlowsHandler(w http.ResponseWriter, r *http.Request) {
	var req v1.AggregationRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := h.querier.AggregateFlows(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to query flows: %v", err), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := protojson.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(jsonBytes); err != nil {
		log.Printf("failed to write aggregate response: %v", err)
	}
}

func (h *legacyHTTPHandler) traceFlowHandler(w http.ResponseWriter, r *http.Request) {
	var req v1.TraceFlowRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	if err := protojson.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := h.querier.TraceFlow(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to trace flow: %v", err), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := protojson.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(jsonBytes); err != nil {
		log.Printf("failed to write trace response: %v", err)
	}
}

func newGrafanaHTTPHandler(service *QueryServiceServer) http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		var req queryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		endTime := req.Range.To
		if endTime.IsZero() {
			endTime = time.Now().Add(24 * time.Hour)
		}

		response := make([]timeSeriesResponse, 0, len(req.Targets))
		for _, target := range req.Targets {
			aggReq := &v1.AggregationRequest{
				EndTime:  timestamppb.New(endTime),
				TaskName: target.Target,
			}

			aggResp, err := service.AggregateFlows(r.Context(), aggReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var totalPackets float64
			if len(aggResp.Summaries) > 0 {
				totalPackets = float64(aggResp.Summaries[0].TotalPackets)
			}

			response = append(response, timeSeriesResponse{
				Target: target.Target,
				Datapoints: [][]float64{
					{totalPackets, float64(endTime.Unix() * 1000)},
				},
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, fmt.Sprintf("failed to encode grafana response: %v", err), http.StatusInternalServerError)
		}
	}).Methods(http.MethodPost)

	return router
}

func runHTTPServer(ctx context.Context, server *http.Server, startupMessage string) error {
	errCh := make(chan error, 1)
	go func() {
		log.Println(startupMessage)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err, ok := <-errCh:
		if ok && err != nil {
			return err
		}
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to shut down http server: %w", err)
	}

	return nil
}
