package db

import (
	"database/sql"
	"fmt"
	"time"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
)

func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(64) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS user_sessions (
			session_token VARCHAR(64) PRIMARY KEY,
			user_id BIGINT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_expires (expires_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS categories (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS items (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			sku VARCHAR(64) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			category VARCHAR(100) NOT NULL DEFAULT 'Umum',
			location VARCHAR(100) NOT NULL DEFAULT '',
			quantity INT NOT NULL DEFAULT 0,
			min_threshold INT NOT NULL DEFAULT 5,
			unit VARCHAR(32) NOT NULL DEFAULT 'pcs',
			price DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_category (category),
			INDEX idx_location (location)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS stock_movements (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			item_id BIGINT NOT NULL,
			type ENUM('IN', 'OUT', 'ADJUSTMENT') NOT NULL,
			quantity INT NOT NULL,
			reference VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
			INDEX idx_item_created (item_id, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS item_audit_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			item_id BIGINT NULL,
			sku VARCHAR(64) NOT NULL,
			name VARCHAR(255) NOT NULL,
			action VARCHAR(32) NOT NULL,
			details JSON NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_item_id (item_id),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS rental_fleet (
			id INT AUTO_INCREMENT PRIMARY KEY,
			component_code VARCHAR(32) NOT NULL UNIQUE,
			name VARCHAR(100) NOT NULL,
			unit VARCHAR(16) NOT NULL DEFAULT 'pcs',
			total_owned INT NOT NULL DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS item_loans (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			loan_code VARCHAR(32) NOT NULL UNIQUE,
			item_id BIGINT NULL,
			item_name VARCHAR(255) NOT NULL,
			quantity INT NOT NULL DEFAULT 1,
			elbow_count INT NOT NULL DEFAULT 0,
			shock_count INT NOT NULL DEFAULT 0,
			borrower_name VARCHAR(255) NOT NULL,
			borrower_phone VARCHAR(64) NOT NULL DEFAULT '',
			project_location VARCHAR(255) NOT NULL DEFAULT '',
			id_card_given BOOLEAN NOT NULL DEFAULT FALSE,
			loan_date DATE NOT NULL,
			due_date DATE NULL,
			return_date DATE NULL,
			status ENUM('ACTIVE', 'RETURNED') NOT NULL DEFAULT 'ACTIVE',
			rental_fee DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
			owner_cost DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
			is_paid BOOLEAN NOT NULL DEFAULT FALSE,
			is_paid_to_owner BOOLEAN NOT NULL DEFAULT FALSE,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_borrower (borrower_name),
			INDEX idx_status (status),
			INDEX idx_loan_date (loan_date)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS loan_monthly_settlements (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			loan_id BIGINT NOT NULL,
			period VARCHAR(7) NOT NULL,
			rental_fee DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
			owner_cost DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
			is_paid BOOLEAN NOT NULL DEFAULT FALSE,
			is_paid_to_owner BOOLEAN NOT NULL DEFAULT FALSE,
			notes VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (loan_id) REFERENCES item_loans(id) ON DELETE CASCADE,
			UNIQUE KEY uk_loan_period (loan_id, period),
			INDEX idx_period (period)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	// Inisialisasi armada sewa jika belum ada
	var fleetCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM rental_fleet`).Scan(&fleetCount)
	if fleetCount == 0 {
		_, _ = db.Exec(`
			INSERT INTO rental_fleet (component_code, name, unit, total_owned) VALUES
			('KPD_SET', 'Kapolding / Steger Set 170cm', 'set', 47),
			('KPD_ELBOW', 'Siku / Cross Brace Steger 220cm', 'pcs', 94),
			('KPD_SHOCK', 'Shock Sambungan / Joint Pin', 'pcs', 99);
		`)
	}

		// Inisialisasi akun admin jika belum ada
	var userCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount)
	if userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err == nil {
			_, _ = db.Exec(`INSERT INTO users (username, password_hash) VALUES (?, ?)`, "admin", string(hash))
		}
	}

	return nil
}