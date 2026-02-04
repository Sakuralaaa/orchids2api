package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"orchids-api/internal/api"
	"orchids-api/internal/config"
	"orchids-api/internal/debug"
	"orchids-api/internal/handler"
	"orchids-api/internal/loadbalancer"
	"orchids-api/internal/middleware"
	"orchids-api/internal/store"
	"orchids-api/web"
)

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "\"'")
			os.Setenv(key, value)
		}
	}
}

func main() {
	loadEnv()

	cfg := config.Load()

	// 启动时清理所有调试日志
	if cfg.DebugEnabled {
		debug.CleanupAllLogs()
		log.Println("已清理调试日志目录")
	}

	dataDir := filepath.Join(".", "data")
	os.MkdirAll(dataDir, 0755)
	dbPath := filepath.Join(dataDir, "orchids.db")

	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer s.Close()

	lb := loadbalancer.New(s)
	apiHandler := api.New(s)
	h := handler.NewWithLoadBalancer(cfg, lb)

	mux := http.NewServeMux()

	// API v1 endpoints
	mux.HandleFunc("/v1/messages", h.HandleMessages)
	mux.HandleFunc("/v1/models", h.HandleModels)
	mux.HandleFunc("/v1/models/", h.HandleModelInfo)

	// Load balancer status endpoint
	mux.HandleFunc("/api/loadbalancer/stats", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := lb.GetStats()
		json.NewEncoder(w).Encode(stats)
	}))

	mux.HandleFunc("/api/loadbalancer/health", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		health := lb.GetAllAccountHealth()
		json.NewEncoder(w).Encode(health)
	}))

	mux.HandleFunc("/api/loadbalancer/strategy", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]string{"strategy": string(lb.GetStrategy())})
		case http.MethodPut, http.MethodPost:
			var req struct {
				Strategy string `json:"strategy"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			lb.SetStrategy(loadbalancer.Strategy(req.Strategy))
			json.NewEncoder(w).Encode(map[string]string{"strategy": req.Strategy})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/accounts", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, apiHandler.HandleAccounts))
	mux.HandleFunc("/api/accounts/", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, apiHandler.HandleAccountByID))
	mux.HandleFunc("/api/export", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, apiHandler.HandleExport))
	mux.HandleFunc("/api/import", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, apiHandler.HandleImport))
	mux.HandleFunc("/api/register", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, apiHandler.HandleRegister))
	mux.HandleFunc("/api/register/verify", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, apiHandler.HandleRegisterVerify))
	mux.HandleFunc("/api/register/batch", middleware.BasicAuth(cfg.AdminUser, cfg.AdminPass, apiHandler.HandleBatchRegister))

	mux.HandleFunc(cfg.AdminPath+"/", middleware.BasicAuthHandler(cfg.AdminUser, cfg.AdminPass, http.StripPrefix(cfg.AdminPath, web.StaticHandler())))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("Server running on port %s", cfg.Port)
	log.Printf("Admin UI: http://localhost:%s%s", cfg.Port, cfg.AdminPath)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatal(err)
	}
}
