package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"strings"
	"warehouse-app/internal/db"
	"warehouse-app/internal/handlers"
)

//go:embed web/*
var webFS embed.FS

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	dsn := getEnv("DB_DSN", "warehouse_user:WarehousePass2026!@tcp(127.0.0.1:3306)/warehouse_db?parseTime=true&loc=Local")
	port := getEnv("PORT", "8080")

	log.Println("Menginisialisasi koneksi database MySQL...")
	database, err := db.InitDB(dsn)
	if err != nil {
		log.Fatalf("Koneksi DB gagal: %v", err)
	}
	defer database.Close()

	log.Println("Database dan migrasi tabel siap.")
	h := handlers.NewHandler(database)

	mux := http.NewServeMux()

	// -------------------------------------------------------------
	// Autentikasi Publik
	// -------------------------------------------------------------
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/auth/me", h.Me)

	// Autentikasi Terproteksi
	mux.HandleFunc("POST /api/auth/change-password", h.RequireAuth(h.ChangePassword))

	// -------------------------------------------------------------
	// REST API Terproteksi (Memerlukan Cookie Sesi Valid)
	// -------------------------------------------------------------

	// REST API Dashboard Utama Analitik
	mux.HandleFunc("GET /api/dashboard/overview", h.RequireAuth(h.GetCentralDashboard))

	// REST API Kategori
	mux.HandleFunc("GET /api/categories", h.RequireAuth(h.ListCategories))
	mux.HandleFunc("POST /api/categories", h.RequireAuth(h.CreateCategory))
	mux.HandleFunc("PUT /api/categories/{id}", h.RequireAuth(h.UpdateCategory))
	mux.HandleFunc("DELETE /api/categories/{id}", h.RequireAuth(h.DeleteCategory))

	// REST API Lokasi / Rak
	mux.HandleFunc("GET /api/locations", h.RequireAuth(h.ListLocations))
	mux.HandleFunc("POST /api/locations", h.RequireAuth(h.CreateLocation))
	mux.HandleFunc("PUT /api/locations/{id}", h.RequireAuth(h.UpdateLocation))
	mux.HandleFunc("DELETE /api/locations/{id}", h.RequireAuth(h.DeleteLocation))

	// REST API Armada Rental Steger (Kapolding, Siku, Shock)
	mux.HandleFunc("GET /api/fleet", h.RequireAuth(h.GetFleet))
	mux.HandleFunc("PUT /api/fleet/{code}", h.RequireAuth(h.UpdateFleet))

	// REST API Inventaris Toko Bahan Bangunan (Gypsum, Cornis, Hollow, Sekrup, dll)
	mux.HandleFunc("GET /api/items", h.RequireAuth(h.ListItems))
	mux.HandleFunc("POST /api/items", h.RequireAuth(h.CreateItem))
	mux.HandleFunc("GET /api/items/{id}", h.RequireAuth(h.GetItem))
	mux.HandleFunc("PUT /api/items/{id}", h.RequireAuth(h.UpdateItem))
	mux.HandleFunc("DELETE /api/items/{id}", h.RequireAuth(h.DeleteItem))
	mux.HandleFunc("POST /api/items/{id}/movements", h.RequireAuth(h.AddStockMovement))
	mux.HandleFunc("GET /api/items/{id}/movements", h.RequireAuth(h.GetItemMovements))

	// REST API Mutasi & Audit Log
	mux.HandleFunc("GET /api/movements", h.RequireAuth(h.GetAllMovements))
	mux.HandleFunc("GET /api/audit-logs", h.RequireAuth(h.GetAuditLogs))
	mux.HandleFunc("GET /api/stats", h.RequireAuth(h.GetDashboardStats))

	// REST API Rental Steger / Kapolding
	mux.HandleFunc("GET /api/loans", h.RequireAuth(h.ListLoans))
	mux.HandleFunc("POST /api/loans", h.RequireAuth(h.CreateLoan))
	mux.HandleFunc("PUT /api/loans/{id}", h.RequireAuth(h.UpdateLoan))
	mux.HandleFunc("DELETE /api/loans/{id}", h.RequireAuth(h.DeleteLoan))
	mux.HandleFunc("PATCH /api/loans/{id}/toggle", h.RequireAuth(h.ToggleLoanField))
	mux.HandleFunc("GET /api/loans/stats", h.RequireAuth(h.GetMonthlyLoanStats))

	// Static Web Frontend dari Live Directory
	fileServer := http.FileServer(http.Dir("web"))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	serverAddr := ":" + port
	log.Printf("Server Gudang & Rental Kapolding aktif di http://0.0.0.0:%s\n", port)
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("Gagal menjalankan web server: %v", err)
	}
}
