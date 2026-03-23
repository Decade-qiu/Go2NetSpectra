package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/query"

	"github.com/gorilla/mux"
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

// RunLegacyHTTPServer reports that the legacy HTTP API is no longer supported.
func RunLegacyHTTPServer(ctx context.Context, cfg *config.Config) error {
	return fmt.Errorf("legacy HTTP API is no longer supported; use ns-api/v2 for Grafana-compatible queries")
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
			aggResp, err := service.aggregateFlows(r.Context(), &query.AggregationRequest{
				EndTime:  &endTime,
				TaskName: target.Target,
			})
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
