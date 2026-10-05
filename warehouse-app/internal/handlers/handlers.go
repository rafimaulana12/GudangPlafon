package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"warehouse-app/internal/models"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}

// -------------------------------------------------------------
// Kategori CRUD
// -------------------------------------------------------------
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT c.id, c.name, COUNT(i.id) AS items_count, c.created_at, c.updated_at
		FROM categories c
		LEFT JOIN items i ON i.category = c.name
		GROUP BY c.id, c.name, c.created_at, c.updated_at
		ORDER BY c.name ASC
	`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	cats := make([]models.Category, 0)
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ItemsCount, &c.CreatedAt, &c.UpdatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		cats = append(cats, c)
	}
	jsonResponse(w, http.StatusOK, cats)
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		jsonError(w, http.StatusBadRequest, "Nama kategori wajib diisi")
		return
	}

	res, err := h.db.Exec(`INSERT INTO categories (name) VALUES (?)`, name)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Gagal membuat kategori (mungkin sudah ada): "+err.Error())
		return
	}
	id, _ := res.LastInsertId()
	jsonResponse(w, http.StatusCreated, models.Category{ID: id, Name: name})
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		jsonError(w, http.StatusBadRequest, "Nama kategori wajib diisi")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()



	var oldName string
	err = tx.QueryRow(`SELECT name FROM categories WHERE id = ? FOR UPDATE`, id).Scan(&oldName)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Kategori tidak ditemukan")
		return
	}

	if oldName == "Umum" && name != "Umum" {
		jsonError(w, http.StatusBadRequest, "Kategori default 'Umum' tidak boleh diganti namanya")
		return
	}

	_, err = tx.Exec(`UPDATE categories SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(`UPDATE items SET category = ? WHERE category = ?`, name, oldName)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, models.Category{ID: id, Name: name})
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var catName string
	err = tx.QueryRow(`SELECT name FROM categories WHERE id = ? FOR UPDATE`, id).Scan(&catName)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Kategori tidak ditemukan")
		return
	}

	if catName == "Umum" {
		jsonError(w, http.StatusBadRequest, "Kategori default 'Umum' tidak boleh dihapus")
		return
	}

	_, _ = tx.Exec(`UPDATE items SET category = 'Umum' WHERE category = ?`, catName)
	_, err = tx.Exec(`DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Kategori berhasil dihapus"})
}

// -------------------------------------------------------------
// Lokasi / Rak CRUD
// -------------------------------------------------------------
func (h *Handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT l.id, l.name, COALESCE(l.description, ""), COUNT(i.id) AS items_count, l.created_at, l.updated_at
		FROM locations l
		LEFT JOIN items i ON i.location = l.name
		GROUP BY l.id, l.name, l.description, l.created_at, l.updated_at
		ORDER BY l.name ASC
	`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	locs := make([]models.Location, 0)
	for rows.Next() {
		var l models.Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.ItemsCount, &l.CreatedAt, &l.UpdatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		locs = append(locs, l)
	}
	jsonResponse(w, http.StatusOK, locs)
}

func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		jsonError(w, http.StatusBadRequest, "Nama lokasi/rak wajib diisi")
		return
	}

	res, err := h.db.Exec(`INSERT INTO locations (name, description) VALUES (?, ?)`, name, strings.TrimSpace(req.Description))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Gagal membuat lokasi: "+err.Error())
		return
	}
	id, _ := res.LastInsertId()
	jsonResponse(w, http.StatusCreated, models.Location{ID: id, Name: name, Description: req.Description})
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		jsonError(w, http.StatusBadRequest, "Nama lokasi/rak wajib diisi")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var oldName string
	err = tx.QueryRow(`SELECT name FROM locations WHERE id = ? FOR UPDATE`, id).Scan(&oldName)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Lokasi tidak ditemukan")
		return
	}

	_, err = tx.Exec(`UPDATE locations SET name = ?, description = ? WHERE id = ?`, name, strings.TrimSpace(req.Description), id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(`UPDATE items SET location = ? WHERE location = ?`, name, oldName)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, models.Location{ID: id, Name: name, Description: req.Description})
}

func (h *Handler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var locName string
	err = tx.QueryRow(`SELECT name FROM locations WHERE id = ? FOR UPDATE`, id).Scan(&locName)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Lokasi tidak ditemukan")
		return
	}

	_, _ = tx.Exec(`UPDATE items SET location = "" WHERE location = ?`, locName)
	_, err = tx.Exec(`DELETE FROM locations WHERE id = ?`, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Lokasi berhasil dihapus"})
}


// -------------------------------------------------------------
// Armada Rental (Kapolding, Siku, Shock)
// -------------------------------------------------------------
func (h *Handler) GetFleet(w http.ResponseWriter, r *http.Request) {
	var rentedSets, rentedElbows, rentedShocks int
	_ = h.db.QueryRow(`
		SELECT 
			COALESCE(SUM(quantity), 0),
			COALESCE(SUM(elbow_count), 0),
			COALESCE(SUM(shock_count), 0)
		FROM item_loans
		WHERE status = 'ACTIVE'
	`).Scan(&rentedSets, &rentedElbows, &rentedShocks)

	rows, err := h.db.Query(`SELECT component_code, name, unit, total_owned FROM rental_fleet ORDER BY id ASC`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	fleet := make([]models.FleetComponent, 0)
	for rows.Next() {
		var f models.FleetComponent
		if err := rows.Scan(&f.Code, &f.Name, &f.Unit, &f.TotalOwned); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}

		switch f.Code {
		case "KPD_SET":
			f.CurrentlyRented = rentedSets
		case "KPD_ELBOW":
			f.CurrentlyRented = rentedElbows
		case "KPD_SHOCK":
			f.CurrentlyRented = rentedShocks
		}
		f.AvailableInWarehouse = f.TotalOwned - f.CurrentlyRented
		if f.AvailableInWarehouse < 0 {
			f.AvailableInWarehouse = 0
		}
		fleet = append(fleet, f)
	}

	jsonResponse(w, http.StatusOK, fleet)
}

func (h *Handler) UpdateFleet(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	var req models.UpdateFleetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.TotalOwned < 0 {
		jsonError(w, http.StatusBadRequest, "Jumlah armada tidak boleh negatif")
		return
	}

	var currentlyRented int
	switch code {
	case "KPD_SET":
		_ = h.db.QueryRow(`SELECT COALESCE(SUM(quantity), 0) FROM item_loans WHERE status = 'ACTIVE'`).Scan(&currentlyRented)
	case "KPD_ELBOW":
		_ = h.db.QueryRow(`SELECT COALESCE(SUM(elbow_count), 0) FROM item_loans WHERE status = 'ACTIVE'`).Scan(&currentlyRented)
	case "KPD_SHOCK":
		_ = h.db.QueryRow(`SELECT COALESCE(SUM(shock_count), 0) FROM item_loans WHERE status = 'ACTIVE'`).Scan(&currentlyRented)
	}

	if req.TotalOwned < currentlyRented {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("Total armada baru (%d) tidak boleh lebih kecil dari jumlah yang sedang aktif tersewa di proyek (%d)", req.TotalOwned, currentlyRented))
		return
	}

	res, err := h.db.Exec(`UPDATE rental_fleet SET total_owned = ? WHERE component_code = ?`, req.TotalOwned, code)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		jsonError(w, http.StatusNotFound, "Komponen armada tidak ditemukan")
		return
	}

	h.GetFleet(w, r)
}

// -------------------------------------------------------------
// Toko & Barang Konsumsi (Gypsum, Cornis, Hollow, Sekrup, dll)
// -------------------------------------------------------------
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, sku, name, category, location, quantity, min_threshold, unit, price, description, created_at, updated_at FROM items WHERE 1=1`
	var args []interface{}

	search := r.URL.Query().Get("search")
	if search != "" {
		query += ` AND (name LIKE ? OR sku LIKE ? OR description LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	category := r.URL.Query().Get("category")
	if category != "" {
		query += ` AND category = ?`
		args = append(args, category)
	}

	query += ` ORDER BY name ASC`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := make([]models.Item, 0)
	for rows.Next() {
		var it models.Item
		var desc sql.NullString
		if err := rows.Scan(&it.ID, &it.SKU, &it.Name, &it.Category, &it.Location, &it.Quantity, &it.MinThreshold, &it.Unit, &it.Price, &desc, &it.CreatedAt, &it.UpdatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if desc.Valid {
			it.Description = desc.String
		}
		items = append(items, it)
	}

	jsonResponse(w, http.StatusOK, items)
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var it models.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	it.Name = strings.TrimSpace(it.Name)
	it.SKU = strings.TrimSpace(it.SKU)
	if it.Name == "" || it.SKU == "" {
		jsonError(w, http.StatusBadRequest, "Nama barang dan SKU wajib diisi")
		return
	}
	if it.Category == "" {
		it.Category = "Umum"
	}
	if it.Unit == "" {
		it.Unit = "pcs"
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	if it.Category != "" {
		_, _ = tx.Exec(`INSERT IGNORE INTO categories (name) VALUES (?)`, it.Category)
	}
	if it.Location != "" {
		_, _ = tx.Exec(`INSERT IGNORE INTO locations (name) VALUES (?)`, it.Location)
	}



	res, err := tx.Exec(`
		INSERT INTO items (sku, name, category, location, quantity, min_threshold, unit, price, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, it.SKU, it.Name, it.Category, it.Location, it.Quantity, it.MinThreshold, it.Unit, it.Price, it.Description)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Gagal membuat barang (mungkin SKU duplikat): "+err.Error())
		return
	}

	id, _ := res.LastInsertId()
	it.ID = id

	if it.Quantity > 0 {
		_, _ = tx.Exec(`
			INSERT INTO stock_movements (item_id, type, quantity, reference)
			VALUES (?, 'IN', ?, 'Stok Awal')
		`, id, it.Quantity)
	}

	detailBytes, _ := json.Marshal(it)
	_, _ = tx.Exec(`
		INSERT INTO item_audit_logs (item_id, sku, name, action, details)
		VALUES (?, ?, ?, 'CREATE', ?)
	`, id, it.SKU, it.Name, string(detailBytes))

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, it)
}

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	var it models.Item
	var desc sql.NullString
	err = h.db.QueryRow(`
		SELECT id, sku, name, category, location, quantity, min_threshold, unit, price, description, created_at, updated_at
		FROM items WHERE id = ?
	`, id).Scan(&it.ID, &it.SKU, &it.Name, &it.Category, &it.Location, &it.Quantity, &it.MinThreshold, &it.Unit, &it.Price, &desc, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Barang tidak ditemukan")
		return
	}
	if desc.Valid {
		it.Description = desc.String
	}

	jsonResponse(w, http.StatusOK, it)
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	var it models.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	it.Name = strings.TrimSpace(it.Name)
	it.SKU = strings.TrimSpace(it.SKU)
	if it.Name == "" || it.SKU == "" {
		jsonError(w, http.StatusBadRequest, "Nama barang dan SKU wajib diisi")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var old models.Item
	var desc sql.NullString
	err = tx.QueryRow(`
		SELECT id, sku, name, category, location, quantity, min_threshold, unit, price, description
		FROM items WHERE id = ? FOR UPDATE
	`, id).Scan(&old.ID, &old.SKU, &old.Name, &old.Category, &old.Location, &old.Quantity, &old.MinThreshold, &old.Unit, &old.Price, &desc)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Barang tidak ditemukan")
		return
	}
	if desc.Valid {
		old.Description = desc.String
	}

	if it.Category != "" {
		_, _ = tx.Exec(`INSERT IGNORE INTO categories (name) VALUES (?)`, it.Category)
	}
	if it.Location != "" {
		_, _ = tx.Exec(`INSERT IGNORE INTO locations (name) VALUES (?)`, it.Location)
	}

	_, err = tx.Exec(`
		UPDATE items SET sku = ?, name = ?, category = ?, location = ?, min_threshold = ?, unit = ?, price = ?, description = ?
		WHERE id = ?
	`, it.SKU, it.Name, it.Category, it.Location, it.MinThreshold, it.Unit, it.Price, it.Description, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"old": old,
		"new": it,
	})
	_, _ = tx.Exec(`
		INSERT INTO item_audit_logs (item_id, sku, name, action, details)
		VALUES (?, ?, ?, 'UPDATE', ?)
	`, id, it.SKU, it.Name, string(auditDetails))

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	it.ID = id
	it.Quantity = old.Quantity
	jsonResponse(w, http.StatusOK, it)
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var it models.Item
	err = tx.QueryRow(`SELECT id, sku, name, category, quantity FROM items WHERE id = ? FOR UPDATE`, id).Scan(&it.ID, &it.SKU, &it.Name, &it.Category, &it.Quantity)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Barang tidak ditemukan")
		return
	}

	_, err = tx.Exec(`DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	auditDetails, _ := json.Marshal(it)
	_, _ = tx.Exec(`
		INSERT INTO item_audit_logs (item_id, sku, name, action, details)
		VALUES (?, ?, ?, 'DELETE', ?)
	`, id, it.SKU, it.Name, string(auditDetails))

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Barang berhasil dihapus"})
}

func (h *Handler) AddStockMovement(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	var req models.MovementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid movement payload")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var currentQty int
	var sku, name string
	err = tx.QueryRow(`SELECT quantity, sku, name FROM items WHERE id = ? FOR UPDATE`, id).Scan(&currentQty, &sku, &name)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Barang tidak ditemukan")
		return
	}

	var newQty int
	switch req.Type {
	case "IN":
		if req.Quantity <= 0 {
			jsonError(w, http.StatusBadRequest, "Jumlah mutasi masuk harus lebih dari 0")
			return
		}
		newQty = currentQty + req.Quantity
	case "OUT":
		if req.Quantity <= 0 {
			jsonError(w, http.StatusBadRequest, "Jumlah mutasi keluar harus lebih dari 0")
			return
		}
		if currentQty < req.Quantity {
			jsonError(w, http.StatusBadRequest, "Stok tidak mencukupi untuk dikeluarkan")
			return
		}
		newQty = currentQty - req.Quantity
	case "ADJUSTMENT":
		if req.Quantity < 0 {
			jsonError(w, http.StatusBadRequest, "Jumlah stok penyesuaian tidak boleh negatif")
			return
		}
		newQty = req.Quantity
	default:
		jsonError(w, http.StatusBadRequest, "Tipe mutasi tidak valid (IN, OUT, ADJUSTMENT)")
		return
	}

	_, err = tx.Exec(`UPDATE items SET quantity = ? WHERE id = ?`, newQty, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, err = tx.Exec(`
		INSERT INTO stock_movements (item_id, type, quantity, reference)
		VALUES (?, ?, ?, ?)
	`, id, req.Type, req.Quantity, req.Reference)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	movAuditDetails, _ := json.Marshal(map[string]interface{}{
		"type":         req.Type,
		"quantity":     req.Quantity,
		"previous_qty": currentQty,
		"new_qty":      newQty,
		"reference":    req.Reference,
	})
	_, _ = tx.Exec(`
		INSERT INTO item_audit_logs (item_id, sku, name, action, details)
		VALUES (?, ?, ?, ?, ?)
	`, id, sku, name, "MUTASI_"+req.Type, string(movAuditDetails))

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message":      "Mutasi stok berhasil dicatat",
		"item_id":      id,
		"previous_qty": currentQty,
		"new_qty":      newQty,
	})
}

func (h *Handler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	var s models.DashboardStats

	_ = h.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(quantity), 0), COALESCE(SUM(quantity * price), 0) FROM items`).Scan(&s.TotalItems, &s.TotalUnits, &s.TotalValue)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM items WHERE quantity <= min_threshold`).Scan(&s.LowStockCount)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&s.CategoriesCount)

	jsonResponse(w, http.StatusOK, s)
}

func (h *Handler) GetItemMovements(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	rows, err := h.db.Query(`
		SELECT id, item_id, type, quantity, reference, created_at
		FROM stock_movements
		WHERE item_id = ?
		ORDER BY created_at DESC
		LIMIT 50
	`, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	movements := make([]models.StockMovement, 0)
	for rows.Next() {
		var m models.StockMovement
		if err := rows.Scan(&m.ID, &m.ItemID, &m.Type, &m.Quantity, &m.Reference, &m.CreatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		movements = append(movements, m)
	}

	jsonResponse(w, http.StatusOK, movements)
}

func (h *Handler) GetAllMovements(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT m.id, m.item_id, i.name, m.type, m.quantity, m.reference, m.created_at
		FROM stock_movements m
		JOIN items i ON i.id = m.item_id
		ORDER BY m.created_at DESC
		LIMIT 100
	`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	movements := make([]models.StockMovement, 0)
	for rows.Next() {
		var m models.StockMovement
		if err := rows.Scan(&m.ID, &m.ItemID, &m.ItemName, &m.Type, &m.Quantity, &m.Reference, &m.CreatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		movements = append(movements, m)
	}

	jsonResponse(w, http.StatusOK, movements)
}

func (h *Handler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT id, item_id, sku, name, action, details, created_at
		FROM item_audit_logs
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	logs := make([]models.AuditLog, 0)
	for rows.Next() {
		var l models.AuditLog
		var rawDetails []byte
		if err := rows.Scan(&l.ID, &l.ItemID, &l.SKU, &l.Name, &l.Action, &rawDetails, &l.CreatedAt); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		l.Details = json.RawMessage(rawDetails)
		logs = append(logs, l)
	}

	jsonResponse(w, http.StatusOK, logs)
}

// -------------------------------------------------------------
// Rental Kapolding & Pelunasan Bulanan (Terpisah dari Toko)
// -------------------------------------------------------------
func (h *Handler) ListLoans(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	unpaidBorrower := r.URL.Query().Get("unpaid_borrower")
	unpaidOwner := r.URL.Query().Get("unpaid_owner")
	idHeld := r.URL.Query().Get("id_held")

	baseQuery := `
		SELECT 
			l.id, l.loan_code, l.item_id, l.item_name, l.quantity, l.elbow_count, l.shock_count,
			l.borrower_name, l.borrower_phone, l.project_location, l.id_card_given,
			l.loan_date, l.due_date, l.return_date, l.status,
			COALESCE(s.rental_fee, l.rental_fee) AS period_rental_fee,
			COALESCE(s.owner_cost, l.owner_cost) AS period_owner_cost,
			COALESCE(s.is_paid, FALSE) AS period_is_paid,
			COALESCE(s.is_paid_to_owner, FALSE) AS period_is_paid_owner,
			(DATE_FORMAT(l.loan_date, '%Y-%m') < ?) AS is_rollover,
			l.notes, l.created_at, l.updated_at
		FROM item_loans l
		LEFT JOIN loan_monthly_settlements s ON s.loan_id = l.id AND s.period = ?
		WHERE (
			DATE_FORMAT(l.loan_date, '%Y-%m') = ?
			OR (
				DATE_FORMAT(l.loan_date, '%Y-%m') < ?
				AND (l.status = 'ACTIVE' OR DATE_FORMAT(COALESCE(l.return_date, '9999-12-31'), '%Y-%m') >= ?)
			)
		)
	`
	args := []interface{}{period, period, period, period, period}

	if search != "" {
		baseQuery += ` AND (l.borrower_name LIKE ? OR l.loan_code LIKE ? OR l.project_location LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	if status != "" {
		baseQuery += ` AND l.status = ?`
		args = append(args, status)
	}

	if unpaidBorrower == "true" {
		baseQuery += ` AND COALESCE(s.is_paid, FALSE) = FALSE`
	}

	if unpaidOwner == "true" {
		baseQuery += ` AND COALESCE(s.is_paid_to_owner, FALSE) = FALSE`
	}

	if idHeld == "true" {
		baseQuery += ` AND l.id_card_given = TRUE`
	}

	baseQuery += ` ORDER BY l.id DESC`

	rows, err := h.db.Query(baseQuery, args...)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	loans := make([]models.Loan, 0)
	for rows.Next() {
		var l models.Loan
		var dueDate, returnDate sql.NullString
		if err := rows.Scan(
			&l.ID, &l.LoanCode, &l.ItemID, &l.ItemName, &l.Quantity, &l.ElbowCount, &l.ShockCount,
			&l.BorrowerName, &l.BorrowerPhone, &l.ProjectLocation, &l.IDCardGiven,
			&l.LoanDate, &dueDate, &returnDate, &l.Status,
			&l.RentalFee, &l.OwnerCost, &l.IsPaid, &l.IsPaidToOwner,
			&l.IsRollover, &l.Notes, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if dueDate.Valid {
			l.DueDate = &dueDate.String
		}
		if returnDate.Valid {
			l.ReturnDate = &returnDate.String
		}
		l.Period = period
		loans = append(loans, l)
	}

	jsonResponse(w, http.StatusOK, loans)
}

func (h *Handler) CreateLoan(w http.ResponseWriter, r *http.Request) {
	var req models.Loan
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.BorrowerName = strings.TrimSpace(req.BorrowerName)
	if req.BorrowerName == "" {
		jsonError(w, http.StatusBadRequest, "Nama peminjam wajib diisi")
		return
	}
	if req.Quantity <= 0 {
		jsonError(w, http.StatusBadRequest, "Jumlah set kapolding harus lebih dari 0")
		return
	}
	if req.LoanDate == "" {
		req.LoanDate = time.Now().Format("2006-01-02")
	}
	if req.Period == "" {
		req.Period = req.LoanDate[:7]
	}
	if req.ElbowCount == 0 {
		req.ElbowCount = req.Quantity * 2
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	// Cek ketersediaan di armada sewa
	var totalOwnedSets, rentedSets int
	_ = tx.QueryRow(`SELECT total_owned FROM rental_fleet WHERE component_code = 'KPD_SET' FOR UPDATE`).Scan(&totalOwnedSets)
	_ = tx.QueryRow(`SELECT COALESCE(SUM(quantity), 0) FROM item_loans WHERE status = 'ACTIVE'`).Scan(&rentedSets)
	availSets := totalOwnedSets - rentedSets
	if availSets < req.Quantity {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("Kapolding di gudang tidak mencukupi: tersedia hanya %d set (armada: %d, sedang tersewa: %d)", availSets, totalOwnedSets, rentedSets))
		return
	}

	// Generate kode sewa
	var maxID sql.NullInt64
	_ = tx.QueryRow(`SELECT MAX(id) FROM item_loans`).Scan(&maxID)
	nextID := int64(1)
	if maxID.Valid {
		nextID = maxID.Int64 + 1
	}
	req.LoanCode = fmt.Sprintf("KPD-%04d", nextID)
	req.ItemName = fmt.Sprintf("%d Set Kapolding", req.Quantity)

	res, err := tx.Exec(`
		INSERT INTO item_loans (
			loan_code, item_id, item_name, quantity, elbow_count, shock_count,
			borrower_name, borrower_phone, project_location, id_card_given,
			loan_date, due_date, status, rental_fee, owner_cost, notes
		) VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE', ?, ?, ?)
	`,
		req.LoanCode, req.ItemName, req.Quantity, req.ElbowCount, req.ShockCount,
		req.BorrowerName, req.BorrowerPhone, req.ProjectLocation, req.IDCardGiven,
		req.LoanDate, req.DueDate, req.RentalFee, req.OwnerCost, req.Notes,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Gagal membuat data sewa: "+err.Error())
		return
	}

	loanID, _ := res.LastInsertId()
	req.ID = loanID

	_, err = tx.Exec(`
		INSERT INTO loan_monthly_settlements (loan_id, period, rental_fee, owner_cost, is_paid, is_paid_to_owner)
		VALUES (?, ?, ?, ?, ?, ?)
	`, loanID, req.Period, req.RentalFee, req.OwnerCost, req.IsPaid, req.IsPaidToOwner)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Gagal membuat settlement bulanan: "+err.Error())
		return
	}

	auditDetails, _ := json.Marshal(req)
	_, _ = tx.Exec(`
		INSERT INTO item_audit_logs (item_id, sku, name, action, details)
		VALUES (NULL, ?, ?, 'LOAN_CREATE', ?)
	`, req.LoanCode, req.BorrowerName, string(auditDetails))

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, req)
}

func (h *Handler) UpdateLoan(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid loan ID")
		return
	}

	var req models.Loan
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.BorrowerName = strings.TrimSpace(req.BorrowerName)
	if req.BorrowerName == "" {
		jsonError(w, http.StatusBadRequest, "Nama peminjam wajib diisi")
		return
	}
	if req.Quantity <= 0 {
		jsonError(w, http.StatusBadRequest, "Jumlah set kapolding harus lebih dari 0")
		return
	}
	if req.ElbowCount == 0 {
		req.ElbowCount = req.Quantity * 2
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var oldLoan models.Loan
	err = tx.QueryRow(`
		SELECT id, loan_code, quantity, elbow_count, shock_count, borrower_name, status
		FROM item_loans WHERE id = ? FOR UPDATE
	`, id).Scan(&oldLoan.ID, &oldLoan.LoanCode, &oldLoan.Quantity, &oldLoan.ElbowCount, &oldLoan.ShockCount, &oldLoan.BorrowerName, &oldLoan.Status)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Data sewa tidak ditemukan")
		return
	}

	if oldLoan.Status == "ACTIVE" && req.Quantity > oldLoan.Quantity {
		diff := req.Quantity - oldLoan.Quantity
		var totalOwnedSets, rentedSets int
		_ = tx.QueryRow(`SELECT total_owned FROM rental_fleet WHERE component_code = 'KPD_SET' FOR UPDATE`).Scan(&totalOwnedSets)
		_ = tx.QueryRow(`SELECT COALESCE(SUM(quantity), 0) FROM item_loans WHERE status = 'ACTIVE'`).Scan(&rentedSets)
		availSets := totalOwnedSets - rentedSets
		if availSets < diff {
			jsonError(w, http.StatusBadRequest, fmt.Sprintf("Penambahan sewa melebihi kapasitas gudang: tersedia hanya %d set tambahan (armada: %d, sedang tersewa: %d)", availSets, totalOwnedSets, rentedSets))
			return
		}
	}

	_, err = tx.Exec(`
		UPDATE item_loans SET
			borrower_name = ?, borrower_phone = ?, project_location = ?,
			quantity = ?, elbow_count = ?, shock_count = ?,
			id_card_given = ?, loan_date = ?, due_date = ?,
			rental_fee = ?, owner_cost = ?, notes = ?
		WHERE id = ?
	`,
		req.BorrowerName, req.BorrowerPhone, req.ProjectLocation,
		req.Quantity, req.ElbowCount, req.ShockCount,
		req.IDCardGiven, req.LoanDate, req.DueDate,
		req.RentalFee, req.OwnerCost, req.Notes, id,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if req.Period != "" {
		_, _ = tx.Exec(`
			INSERT INTO loan_monthly_settlements (loan_id, period, rental_fee, owner_cost, is_paid, is_paid_to_owner)
			VALUES (?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE rental_fee = VALUES(rental_fee), owner_cost = VALUES(owner_cost)
		`, id, req.Period, req.RentalFee, req.OwnerCost, req.IsPaid, req.IsPaidToOwner)
	}

	loanAuditDetails, _ := json.Marshal(map[string]interface{}{
		"action":     "LOAN_UPDATE",
		"old_qty":    oldLoan.Quantity,
		"new_qty":    req.Quantity,
		"rental_fee": req.RentalFee,
		"borrower":   req.BorrowerName,
	})
	_, _ = tx.Exec(`
		INSERT INTO item_audit_logs (item_id, sku, name, action, details)
		VALUES (NULL, ?, ?, 'UPDATE', ?)
	`, oldLoan.LoanCode, req.BorrowerName, string(loanAuditDetails))

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	req.ID = id
	req.LoanCode = oldLoan.LoanCode
	jsonResponse(w, http.StatusOK, req)
}

func (h *Handler) ToggleLoanField(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid loan ID")
		return
	}

	var req models.LoanToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	period := req.Period
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var l models.Loan
	err = tx.QueryRow(`
		SELECT id, loan_code, quantity, borrower_name, status, rental_fee, owner_cost
		FROM item_loans WHERE id = ? FOR UPDATE
	`, id).Scan(&l.ID, &l.LoanCode, &l.Quantity, &l.BorrowerName, &l.Status, &l.RentalFee, &l.OwnerCost)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Data sewa tidak ditemukan")
		return
	}

	switch req.Field {
	case "is_paid":
		_, err = tx.Exec(`
			INSERT INTO loan_monthly_settlements (loan_id, period, rental_fee, owner_cost, is_paid, is_paid_to_owner)
			VALUES (?, ?, ?, ?, TRUE, FALSE)
			ON DUPLICATE KEY UPDATE is_paid = NOT is_paid
		`, id, period, l.RentalFee, l.OwnerCost)

	case "is_paid_to_owner":
		_, err = tx.Exec(`
			INSERT INTO loan_monthly_settlements (loan_id, period, rental_fee, owner_cost, is_paid, is_paid_to_owner)
			VALUES (?, ?, ?, ?, FALSE, TRUE)
			ON DUPLICATE KEY UPDATE is_paid_to_owner = NOT is_paid_to_owner
		`, id, period, l.RentalFee, l.OwnerCost)

	case "id_card_given":
		_, err = tx.Exec(`UPDATE item_loans SET id_card_given = NOT id_card_given WHERE id = ?`, id)

	case "return":
		if l.Status == "ACTIVE" {
			today := time.Now().Format("2006-01-02")
			_, err = tx.Exec(`UPDATE item_loans SET status = 'RETURNED', return_date = ? WHERE id = ?`, today, id)
			_, _ = tx.Exec(`
				INSERT INTO item_audit_logs (item_id, sku, name, action, details)
				VALUES (NULL, ?, ?, 'LOAN_RETURN', ?)
			`, l.LoanCode, l.BorrowerName, fmt.Sprintf(`{"status": "RETURNED", "period": "%s", "sets": %d}`, period, l.Quantity))
		} else {
			_, err = tx.Exec(`UPDATE item_loans SET status = 'ACTIVE', return_date = NULL WHERE id = ?`, id)
		}

	default:
		jsonError(w, http.StatusBadRequest, "Field toggle tidak valid")
		return
	}

	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Status berhasil diubah",
		"period":  period,
	})
}

func (h *Handler) DeleteLoan(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid loan ID")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var l models.Loan
	err = tx.QueryRow(`SELECT id, loan_code, borrower_name, status, quantity FROM item_loans WHERE id = ? FOR UPDATE`, id).Scan(&l.ID, &l.LoanCode, &l.BorrowerName, &l.Status, &l.Quantity)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Data sewa tidak ditemukan")
		return
	}

	_, _ = tx.Exec(`DELETE FROM loan_monthly_settlements WHERE loan_id = ?`, id)
	_, err = tx.Exec(`DELETE FROM item_loans WHERE id = ?`, id)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, _ = tx.Exec(`
		INSERT INTO item_audit_logs (item_id, sku, name, action, details)
		VALUES (NULL, ?, ?, 'DELETE', ?)
	`, l.LoanCode, l.BorrowerName, fmt.Sprintf(`{"deleted": true, "code": "%s"}`, l.LoanCode))

	if err := tx.Commit(); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Data sewa berhasil dihapus"})
}

func (h *Handler) GetMonthlyLoanStats(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	var s models.MonthlyLoanStats
	s.Period = period

	_ = h.db.QueryRow(`
		SELECT 
			COUNT(*),
			CAST(COALESCE(SUM(CASE WHEN l.status = 'ACTIVE' THEN l.quantity ELSE 0 END), 0) AS SIGNED),
			CAST(COALESCE(SUM(CASE WHEN l.status = 'RETURNED' THEN 1 ELSE 0 END), 0) AS SIGNED),
			CAST(COALESCE(SUM(CASE WHEN DATE_FORMAT(l.loan_date, '%Y-%m') < ? THEN 1 ELSE 0 END), 0) AS SIGNED),
			CAST(COALESCE(SUM(l.id_card_given), 0) AS SIGNED),
			COALESCE(SUM(COALESCE(s.rental_fee, l.rental_fee)), 0),
			COALESCE(SUM(CASE WHEN s.is_paid = TRUE THEN COALESCE(s.rental_fee, l.rental_fee) ELSE 0 END), 0),
			COALESCE(SUM(COALESCE(s.owner_cost, l.owner_cost)), 0),
			COALESCE(SUM(CASE WHEN s.is_paid_to_owner = TRUE THEN COALESCE(s.owner_cost, l.owner_cost) ELSE 0 END), 0),
			CAST(COALESCE(SUM(CASE WHEN COALESCE(s.is_paid, FALSE) = FALSE THEN 1 ELSE 0 END), 0) AS SIGNED),
			CAST(COALESCE(SUM(CASE WHEN COALESCE(s.is_paid_to_owner, FALSE) = FALSE THEN 1 ELSE 0 END), 0) AS SIGNED)
		FROM item_loans l
		LEFT JOIN loan_monthly_settlements s ON s.loan_id = l.id AND s.period = ?
		WHERE (
			DATE_FORMAT(l.loan_date, '%Y-%m') = ?
			OR (
				DATE_FORMAT(l.loan_date, '%Y-%m') < ?
				AND (l.status = 'ACTIVE' OR DATE_FORMAT(COALESCE(l.return_date, '9999-12-31'), '%Y-%m') >= ?)
			)
		)
	`, period, period, period, period, period).Scan(
		&s.TotalActiveLoans,
		&s.TotalSetsRented,
		&s.ReturnedLoans,
		&s.RolloverLoans,
		&s.IDCardsHeld,
		&s.ExpectedRevenue,
		&s.CollectedRevenue,
		&s.ExpectedCost,
		&s.SettledCost,
		&s.UnpaidBorrower,
		&s.UnpaidOwner,
	)

	jsonResponse(w, http.StatusOK, s)
}

// -------------------------------------------------------------
// Central Dashboard Analytics Overview
// -------------------------------------------------------------
func (h *Handler) GetCentralDashboard(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	var overview models.CentralDashboardOverview
	overview.Rental.Period = period

	// 1. Store Analytics
	_ = h.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(quantity), 0), COALESCE(SUM(quantity * price), 0)
		FROM items
	`).Scan(&overview.Store.TotalItems, &overview.Store.TotalUnits, &overview.Store.TotalValue)

	_ = h.db.QueryRow(`SELECT COUNT(*) FROM items WHERE quantity <= min_threshold`).Scan(&overview.Store.LowStockCount)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&overview.Store.CategoriesCount)

	catRows, err := h.db.Query(`
		SELECT category, COUNT(*), COALESCE(SUM(quantity * price), 0)
		FROM items
		GROUP BY category
		ORDER BY SUM(quantity * price) DESC
	`)
	if err == nil {
		defer catRows.Close()
		for catRows.Next() {
			var cb models.CategoryBreakdown
			if err := catRows.Scan(&cb.Category, &cb.ItemsCount, &cb.TotalValue); err == nil {
				overview.Store.CategoryBreakdown = append(overview.Store.CategoryBreakdown, cb)
			}
		}
	}

	// 2. Rental Fleet & Contract Analytics
	var totalOwnedSets int
	_ = h.db.QueryRow(`SELECT total_owned FROM rental_fleet WHERE component_code = 'KPD_SET'`).Scan(&totalOwnedSets)
	overview.Rental.TotalFleetSets = totalOwnedSets

	err = h.db.QueryRow(`
		SELECT 
			COUNT(*),
			CAST(COALESCE(SUM(CASE WHEN l.status = 'ACTIVE' THEN l.quantity ELSE 0 END), 0) AS SIGNED),
			CAST(COALESCE(SUM(l.id_card_given), 0) AS SIGNED),
			COALESCE(SUM(COALESCE(s.rental_fee, l.rental_fee)), 0),
			COALESCE(SUM(CASE WHEN s.is_paid = TRUE THEN COALESCE(s.rental_fee, l.rental_fee) ELSE 0 END), 0),
			COALESCE(SUM(COALESCE(s.owner_cost, l.owner_cost)), 0),
			COALESCE(SUM(CASE WHEN s.is_paid_to_owner = TRUE THEN COALESCE(s.owner_cost, l.owner_cost) ELSE 0 END), 0),
			CAST(COALESCE(SUM(CASE WHEN COALESCE(s.is_paid, FALSE) = FALSE AND l.status = 'ACTIVE' THEN 1 ELSE 0 END), 0) AS SIGNED),
			CAST(COALESCE(SUM(CASE WHEN COALESCE(s.is_paid_to_owner, FALSE) = FALSE AND l.status = 'ACTIVE' THEN 1 ELSE 0 END), 0) AS SIGNED)
		FROM item_loans l
		LEFT JOIN loan_monthly_settlements s ON s.loan_id = l.id AND s.period = ?
		WHERE (
			DATE_FORMAT(l.loan_date, '%Y-%m') = ?
			OR (
				DATE_FORMAT(l.loan_date, '%Y-%m') < ?
				AND (l.status = 'ACTIVE' OR DATE_FORMAT(COALESCE(l.return_date, '9999-12-31'), '%Y-%m') >= ?)
			)
		)
	`, period, period, period, period).Scan(
		&overview.Rental.ActiveContracts,
		&overview.Rental.RentedSets,
		&overview.Rental.IDCardsHeld,
		&overview.Rental.ExpectedRevenue,
		&overview.Rental.CollectedRevenue,
		&overview.Rental.ExpectedCost,
		&overview.Rental.SettledCost,
		&overview.Rental.UnpaidBorrowers,
		&overview.Rental.UnpaidOwners,
	)

	overview.Rental.AvailableSets = overview.Rental.TotalFleetSets - overview.Rental.RentedSets
	if overview.Rental.AvailableSets < 0 {
		overview.Rental.AvailableSets = 0
	}
	if overview.Rental.TotalFleetSets > 0 {
		overview.Rental.UtilizationRate = (float64(overview.Rental.RentedSets) / float64(overview.Rental.TotalFleetSets)) * 100.0
	}
	overview.Rental.UncollectedRevenue = overview.Rental.ExpectedRevenue - overview.Rental.CollectedRevenue
	overview.Rental.UnsettledCost = overview.Rental.ExpectedCost - overview.Rental.SettledCost
	overview.Rental.NetProfitProjected = overview.Rental.ExpectedRevenue - overview.Rental.ExpectedCost
	overview.Rental.NetProfitCollected = overview.Rental.CollectedRevenue - overview.Rental.SettledCost

	// 3. Recent 5 Activity Logs
	logRows, err := h.db.Query(`
		SELECT id, item_id, sku, name, action, details, created_at
		FROM item_audit_logs
		ORDER BY created_at DESC
		LIMIT 5
	`)
	if err == nil {
		defer logRows.Close()
		for logRows.Next() {
			var l models.AuditLog
			var rawDetails []byte
			if err := logRows.Scan(&l.ID, &l.ItemID, &l.SKU, &l.Name, &l.Action, &rawDetails, &l.CreatedAt); err == nil {
				l.Details = json.RawMessage(rawDetails)
				overview.RecentActivities = append(overview.RecentActivities, l)
			}
		}
	}

	jsonResponse(w, http.StatusOK, overview)
}
