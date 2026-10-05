package models

import "encoding/json"

type Category struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ItemsCount int    `json:"items_count"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type Location struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ItemsCount  int    `json:"items_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}


type Item struct {
	ID           int64   `json:"id"`
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	Location     string  `json:"location"`
	Quantity     int     `json:"quantity"`
	MinThreshold int     `json:"min_threshold"`
	Unit         string  `json:"unit"`
	Price        float64 `json:"price"`
	Description  string  `json:"description"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type StockMovement struct {
	ID        int64  `json:"id"`
	ItemID    int64  `json:"item_id"`
	ItemName  string `json:"item_name,omitempty"`
	Type      string `json:"type"` // IN, OUT, ADJUSTMENT
	Quantity  int    `json:"quantity"`
	Reference string `json:"reference"`
	CreatedAt string `json:"created_at"`
}

type MovementRequest struct {
	Type      string `json:"type"` // IN, OUT, ADJUSTMENT
	Quantity  int    `json:"quantity"`
	Reference string `json:"reference"`
}

type AuditLog struct {
	ID        int64           `json:"id"`
	ItemID    *int64          `json:"item_id"`
	SKU       string          `json:"sku"`
	Name      string          `json:"name"`
	Action    string          `json:"action"` // CREATE, UPDATE, DELETE, LOAN_CREATE, LOAN_RETURN
	Details   json.RawMessage `json:"details"`
	CreatedAt string          `json:"created_at"`
}

type DashboardStats struct {
	TotalItems      int     `json:"total_items"`
	TotalUnits      int     `json:"total_units"`
	TotalValue      float64 `json:"total_value"`
	LowStockCount   int     `json:"low_stock_count"`
	CategoriesCount int     `json:"categories_count"`
}

type FleetComponent struct {
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Unit                 string `json:"unit"`
	TotalOwned           int    `json:"total_owned"`
	CurrentlyRented      int    `json:"currently_rented"`
	AvailableInWarehouse int    `json:"available_in_warehouse"`
}

type UpdateFleetRequest struct {
	TotalOwned int `json:"total_owned"`
}

type Loan struct {
	ID              int64   `json:"id"`
	LoanCode        string  `json:"loan_code"`
	ItemID          *int64  `json:"item_id"`
	ItemName        string  `json:"item_name"`
	Quantity        int     `json:"quantity"`
	ElbowCount      int     `json:"elbow_count"`
	ShockCount      int     `json:"shock_count"`
	BorrowerName    string  `json:"borrower_name"`
	BorrowerPhone   string  `json:"borrower_phone"`
	ProjectLocation string  `json:"project_location"`
	IDCardGiven     bool    `json:"id_card_given"`
	LoanDate        string  `json:"loan_date"`
	DueDate         *string `json:"due_date"`
	ReturnDate      *string `json:"return_date"`
	Status          string  `json:"status"` // ACTIVE, RETURNED
	Period          string  `json:"period"` // YYYY-MM
	RentalFee       float64 `json:"rental_fee"`
	OwnerCost       float64 `json:"owner_cost"`
	IsPaid          bool    `json:"is_paid"`
	IsPaidToOwner   bool    `json:"is_paid_to_owner"`
	IsRollover      bool    `json:"is_rollover"`
	Notes           string  `json:"notes"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type LoanToggleRequest struct {
	Field  string `json:"field"`
	Period string `json:"period"`
}

type MonthlyLoanStats struct {
	Period           string  `json:"period"`
	TotalActiveLoans int     `json:"total_active_loans"`
	TotalSetsRented  int     `json:"total_sets_rented"`
	ReturnedLoans    int     `json:"returned_loans"`
	RolloverLoans    int     `json:"rollover_loans"`
	ExpectedRevenue  float64 `json:"expected_revenue"`
	CollectedRevenue float64 `json:"collected_revenue"`
	ExpectedCost     float64 `json:"expected_cost"`
	SettledCost      float64 `json:"settled_cost"`
	UnpaidBorrower   int     `json:"unpaid_borrower"`
	UnpaidOwner      int     `json:"unpaid_owner"`
	IDCardsHeld      int     `json:"id_cards_held"`
}

type CategoryBreakdown struct {
	Category   string  `json:"category"`
	ItemsCount int     `json:"items_count"`
	TotalValue float64 `json:"total_value"`
}

type StoreOverview struct {
	TotalItems        int                 `json:"total_items"`
	TotalUnits        int                 `json:"total_units"`
	TotalValue        float64             `json:"total_value"`
	LowStockCount     int                 `json:"low_stock_count"`
	CategoriesCount   int                 `json:"categories_count"`
	CategoryBreakdown []CategoryBreakdown `json:"category_breakdown"`
}

type RentalOverview struct {
	Period             string  `json:"period"`
	TotalFleetSets     int     `json:"total_fleet_sets"`
	RentedSets         int     `json:"rented_sets"`
	AvailableSets      int     `json:"available_sets"`
	UtilizationRate    float64 `json:"utilization_rate"`
	ActiveContracts    int     `json:"active_contracts"`
	IDCardsHeld        int     `json:"id_cards_held"`
	ExpectedRevenue    float64 `json:"expected_revenue"`
	CollectedRevenue   float64 `json:"collected_revenue"`
	UncollectedRevenue float64 `json:"uncollected_revenue"`
	ExpectedCost       float64 `json:"expected_cost"`
	SettledCost        float64 `json:"settled_cost"`
	UnsettledCost      float64 `json:"unsettled_cost"`
	NetProfitCollected float64 `json:"net_profit_collected"`
	NetProfitProjected float64 `json:"net_profit_projected"`
	UnpaidBorrowers    int     `json:"unpaid_borrowers_count"`
	UnpaidOwners       int     `json:"unpaid_owners_count"`
}

type CentralDashboardOverview struct {
	Store            StoreOverview   `json:"store"`
	Rental           RentalOverview  `json:"rental"`
	RecentActivities []AuditLog      `json:"recent_activities"`
}
