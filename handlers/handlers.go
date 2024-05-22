package handlers

import (
	"net/http"
	"worker-service/worker"
)

func WorkerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		worker.GetWorkers(w, r)
	case http.MethodPost:
		worker.CreateWorker(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
