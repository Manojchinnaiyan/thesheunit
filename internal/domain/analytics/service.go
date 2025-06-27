// internal/domain/analytics/service.go
package analytics

import (
	"fmt"
	"time"

	"github.com/your-org/ecommerce-backend/internal/config"
	"gorm.io/gorm"
)

// Service handles analytics business logic
type Service struct {
	db     *gorm.DB
	config *config.Config
}

// NewService creates a new analytics service
func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		config: cfg,
	}
}

// DashboardStats represents overall dashboard statistics
type DashboardStats struct {
	// Sales metrics
	TotalRevenue     int64   `json:"total_revenue"`      // In cents
	RevenueToday     int64   `json:"revenue_today"`      // In cents
	RevenueThisWeek  int64   `json:"revenue_this_week"`  // In cents
	RevenueThisMonth int64   `json:"revenue_this_month"` // In cents
	RevenueGrowth    float64 `json:"revenue_growth"`     // Percentage

	// Order metrics
	TotalOrders     int64   `json:"total_orders"`
	OrdersToday     int64   `json:"orders_today"`
	OrdersThisWeek  int64   `json:"orders_this_week"`
	OrdersThisMonth int64   `json:"orders_this_month"`
	OrderGrowth     float64 `json:"order_growth"` // Percentage

	// User metrics
	TotalUsers        int64   `json:"total_users"`
	ActiveUsers       int64   `json:"active_users"`
	NewUsersToday     int64   `json:"new_users_today"`
	NewUsersThisWeek  int64   `json:"new_users_this_week"`
	NewUsersThisMonth int64   `json:"new_users_this_month"`
	UserGrowth        float64 `json:"user_growth"` // Percentage

	// Product metrics
	TotalProducts      int64 `json:"total_products"`
	ActiveProducts     int64 `json:"active_products"`
	OutOfStockProducts int64 `json:"out_of_stock_products"`
	LowStockProducts   int64 `json:"low_stock_products"`

	// Conversion metrics
	ConversionRate     float64 `json:"conversion_rate"`      // Percentage
	AvgOrderValue      int64   `json:"avg_order_value"`      // In cents
	RepeatCustomerRate float64 `json:"repeat_customer_rate"` // Percentage
}

// SalesAnalytics represents sales analytics data
type SalesAnalytics struct {
	// Time-based revenue
	DailyRevenue   []TimeSeriesData `json:"daily_revenue"`
	WeeklyRevenue  []TimeSeriesData `json:"weekly_revenue"`
	MonthlyRevenue []TimeSeriesData `json:"monthly_revenue"`

	// Sales summary
	TotalSales    int64              `json:"total_sales"`
	TotalRevenue  int64              `json:"total_revenue"`
	AvgOrderValue int64              `json:"avg_order_value"`
	TopProducts   []ProductSalesData `json:"top_products"`
	SalesByStatus []StatusData       `json:"sales_by_status"`

	// Growth metrics
	RevenueGrowth float64 `json:"revenue_growth"`
	SalesGrowth   float64 `json:"sales_growth"`
}

// ProductAnalytics represents product analytics data
type ProductAnalytics struct {
	TotalProducts      int64              `json:"total_products"`
	ActiveProducts     int64              `json:"active_products"`
	TopSellingProducts []ProductSalesData `json:"top_selling_products"`
	CategorySales      []CategoryData     `json:"category_sales"`
	LowStockProducts   []LowStockData     `json:"low_stock_products"`
	ProductViews       []ProductViewData  `json:"product_views"`
	InventoryValue     int64              `json:"inventory_value"`
}

// CustomerAnalytics represents customer analytics data
type CustomerAnalytics struct {
	TotalCustomers        int64          `json:"total_customers"`
	ActiveCustomers       int64          `json:"active_customers"`
	NewCustomers          int64          `json:"new_customers"`
	RepeatCustomers       int64          `json:"repeat_customers"`
	CustomerGrowth        float64        `json:"customer_growth"`
	TopCustomers          []CustomerData `json:"top_customers"`
	CustomersByLocation   []LocationData `json:"customers_by_location"`
	CustomerLifetimeValue int64          `json:"customer_lifetime_value"`
}

// RevenueAnalytics represents revenue analytics data
type RevenueAnalytics struct {
	TotalRevenue      int64              `json:"total_revenue"`
	RevenueGrowth     float64            `json:"revenue_growth"`
	RevenueByPeriod   []TimeSeriesData   `json:"revenue_by_period"`
	RevenueByCategory []CategoryData     `json:"revenue_by_category"`
	RevenueByProduct  []ProductSalesData `json:"revenue_by_product"`
	AvgOrderValue     int64              `json:"avg_order_value"`
	RevenueTargets    RevenueTargets     `json:"revenue_targets"`
}

// Supporting data structures
type TimeSeriesData struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
	Count int64  `json:"count,omitempty"`
}

type ProductSalesData struct {
	ProductID   uint   `json:"product_id"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku"`
	TotalSold   int64  `json:"total_sold"`
	Revenue     int64  `json:"revenue"`
	OrderCount  int64  `json:"order_count"`
}

type CategoryData struct {
	CategoryID   uint   `json:"category_id"`
	CategoryName string `json:"category_name"`
	Revenue      int64  `json:"revenue"`
	OrderCount   int64  `json:"order_count"`
	ProductCount int64  `json:"product_count"`
}

type StatusData struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
	Value  int64  `json:"value"`
}

type LowStockData struct {
	ProductID     uint   `json:"product_id"`
	ProductName   string `json:"product_name"`
	SKU           string `json:"sku"`
	CurrentStock  int    `json:"current_stock"`
	ReorderLevel  int    `json:"reorder_level"`
	WarehouseName string `json:"warehouse_name"`
}

type ProductViewData struct {
	ProductID   uint   `json:"product_id"`
	ProductName string `json:"product_name"`
	ViewCount   int64  `json:"view_count"`
	Revenue     int64  `json:"revenue"`
}

type CustomerData struct {
	UserID       uint       `json:"user_id"`
	CustomerName string     `json:"customer_name"`
	Email        string     `json:"email"`
	TotalSpent   int64      `json:"total_spent"`
	OrderCount   int64      `json:"order_count"`
	LastOrder    *time.Time `json:"last_order"`
}

type LocationData struct {
	Country       string `json:"country"`
	State         string `json:"state"`
	City          string `json:"city"`
	CustomerCount int64  `json:"customer_count"`
	Revenue       int64  `json:"revenue"`
}

type RevenueTargets struct {
	DailyTarget   int64   `json:"daily_target"`
	WeeklyTarget  int64   `json:"weekly_target"`
	MonthlyTarget int64   `json:"monthly_target"`
	YearlyTarget  int64   `json:"yearly_target"`
	Achievement   float64 `json:"achievement"` // Percentage of monthly target achieved
}

// GetDashboardStats retrieves overall dashboard statistics
func (s *Service) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}
	now := time.Now()

	// Define time periods
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thisWeek := today.AddDate(0, 0, -int(today.Weekday()))
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonth := thisMonth.AddDate(0, -1, 0)

	// Revenue metrics
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status NOT IN ('cancelled', 'failed')").Scan(&stats.TotalRevenue)
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status NOT IN ('cancelled', 'failed') AND created_at >= ?", today).Scan(&stats.RevenueToday)
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status NOT IN ('cancelled', 'failed') AND created_at >= ?", thisWeek).Scan(&stats.RevenueThisWeek)
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status NOT IN ('cancelled', 'failed') AND created_at >= ?", thisMonth).Scan(&stats.RevenueThisMonth)

	// Calculate revenue growth (current month vs last month)
	var lastMonthRevenue int64
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status NOT IN ('cancelled', 'failed') AND created_at >= ? AND created_at < ?", lastMonth, thisMonth).Scan(&lastMonthRevenue)
	if lastMonthRevenue > 0 {
		stats.RevenueGrowth = float64(stats.RevenueThisMonth-lastMonthRevenue) / float64(lastMonthRevenue) * 100
	}

	// Order metrics
	s.db.Raw("SELECT COUNT(*) FROM orders").Scan(&stats.TotalOrders)
	s.db.Raw("SELECT COUNT(*) FROM orders WHERE created_at >= ?", today).Scan(&stats.OrdersToday)
	s.db.Raw("SELECT COUNT(*) FROM orders WHERE created_at >= ?", thisWeek).Scan(&stats.OrdersThisWeek)
	s.db.Raw("SELECT COUNT(*) FROM orders WHERE created_at >= ?", thisMonth).Scan(&stats.OrdersThisMonth)

	// Calculate order growth
	var lastMonthOrders int64
	s.db.Raw("SELECT COUNT(*) FROM orders WHERE created_at >= ? AND created_at < ?", lastMonth, thisMonth).Scan(&lastMonthOrders)
	if lastMonthOrders > 0 {
		stats.OrderGrowth = float64(stats.OrdersThisMonth-lastMonthOrders) / float64(lastMonthOrders) * 100
	}

	// User metrics
	s.db.Raw("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	s.db.Raw("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&stats.ActiveUsers)
	s.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ?", today).Scan(&stats.NewUsersToday)
	s.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ?", thisWeek).Scan(&stats.NewUsersThisWeek)
	s.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ?", thisMonth).Scan(&stats.NewUsersThisMonth)

	// Calculate user growth
	var lastMonthUsers int64
	s.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ? AND created_at < ?", lastMonth, thisMonth).Scan(&lastMonthUsers)
	if lastMonthUsers > 0 {
		stats.UserGrowth = float64(stats.NewUsersThisMonth-lastMonthUsers) / float64(lastMonthUsers) * 100
	}

	// Product metrics
	s.db.Raw("SELECT COUNT(*) FROM products").Scan(&stats.TotalProducts)
	s.db.Raw("SELECT COUNT(*) FROM products WHERE is_active = true").Scan(&stats.ActiveProducts)

	// Inventory metrics (if inventory system exists)
	s.db.Raw("SELECT COUNT(*) FROM inventory_items WHERE available_quantity <= 0").Scan(&stats.OutOfStockProducts)
	s.db.Raw("SELECT COUNT(*) FROM inventory_items WHERE available_quantity <= reorder_level AND available_quantity > 0").Scan(&stats.LowStockProducts)

	// Conversion metrics
	if stats.TotalUsers > 0 {
		stats.ConversionRate = float64(stats.TotalOrders) / float64(stats.TotalUsers) * 100
	}

	if stats.TotalOrders > 0 {
		stats.AvgOrderValue = stats.TotalRevenue / stats.TotalOrders
	}

	// Repeat customer rate
	var repeatCustomers int64
	s.db.Raw("SELECT COUNT(DISTINCT user_id) FROM orders WHERE user_id IN (SELECT user_id FROM orders GROUP BY user_id HAVING COUNT(*) > 1)").Scan(&repeatCustomers)
	if stats.TotalUsers > 0 {
		stats.RepeatCustomerRate = float64(repeatCustomers) / float64(stats.TotalUsers) * 100
	}

	return stats, nil
}

// GetSalesAnalytics retrieves sales analytics data
func (s *Service) GetSalesAnalytics(days int) (*SalesAnalytics, error) {
	analytics := &SalesAnalytics{}

	// Default to 30 days if not specified
	if days <= 0 {
		days = 30
	}

	startDate := time.Now().AddDate(0, 0, -days)

	// Get daily revenue for the period
	rows, err := s.db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COALESCE(SUM(total_amount), 0) as revenue,
			COUNT(*) as order_count
		FROM orders 
		WHERE created_at >= ? AND status NOT IN ('cancelled', 'failed')
		GROUP BY DATE(created_at)
		ORDER BY date
	`, startDate).Rows()

	if err != nil {
		return nil, fmt.Errorf("failed to get daily revenue: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data TimeSeriesData
		if err := rows.Scan(&data.Date, &data.Value, &data.Count); err != nil {
			continue
		}
		analytics.DailyRevenue = append(analytics.DailyRevenue, data)
	}

	// Get summary metrics
	s.db.Raw("SELECT COUNT(*) FROM orders WHERE created_at >= ? AND status NOT IN ('cancelled', 'failed')", startDate).Scan(&analytics.TotalSales)
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE created_at >= ? AND status NOT IN ('cancelled', 'failed')", startDate).Scan(&analytics.TotalRevenue)

	if analytics.TotalSales > 0 {
		analytics.AvgOrderValue = analytics.TotalRevenue / analytics.TotalSales
	}

	// Get top products
	productRows, err := s.db.Raw(`
		SELECT 
			p.id,
			p.name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) as total_sold,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			COUNT(DISTINCT o.id) as order_count
		FROM products p
		LEFT JOIN order_items oi ON p.id = oi.product_id
		LEFT JOIN orders o ON oi.order_id = o.id
		WHERE o.created_at >= ? AND o.status NOT IN ('cancelled', 'failed')
		GROUP BY p.id, p.name, p.sku
		ORDER BY revenue DESC
		LIMIT 10
	`, startDate).Rows()

	if err == nil {
		defer productRows.Close()
		for productRows.Next() {
			var product ProductSalesData
			if err := productRows.Scan(&product.ProductID, &product.ProductName, &product.SKU, &product.TotalSold, &product.Revenue, &product.OrderCount); err != nil {
				continue
			}
			analytics.TopProducts = append(analytics.TopProducts, product)
		}
	}

	// Get sales by status
	statusRows, err := s.db.Raw(`
		SELECT 
			status,
			COUNT(*) as count,
			COALESCE(SUM(total_amount), 0) as value
		FROM orders 
		WHERE created_at >= ?
		GROUP BY status
		ORDER BY count DESC
	`, startDate).Rows()

	if err == nil {
		defer statusRows.Close()
		for statusRows.Next() {
			var status StatusData
			if err := statusRows.Scan(&status.Status, &status.Count, &status.Value); err != nil {
				continue
			}
			analytics.SalesByStatus = append(analytics.SalesByStatus, status)
		}
	}

	return analytics, nil
}

// Add these methods to the end of your analytics service.go file:

// GetProductAnalytics retrieves product analytics data
func (s *Service) GetProductAnalytics() (*ProductAnalytics, error) {
	analytics := &ProductAnalytics{}

	// Basic product counts
	s.db.Raw("SELECT COUNT(*) FROM products").Scan(&analytics.TotalProducts)
	s.db.Raw("SELECT COUNT(*) FROM products WHERE is_active = true").Scan(&analytics.ActiveProducts)

	// Top selling products (last 30 days)
	productRows, err := s.db.Raw(`
		SELECT 
			p.id,
			p.name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) as total_sold,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			COUNT(DISTINCT o.id) as order_count
		FROM products p
		LEFT JOIN order_items oi ON p.id = oi.product_id
		LEFT JOIN orders o ON oi.order_id = o.id
		WHERE o.created_at >= ? AND o.status NOT IN ('cancelled', 'failed')
		GROUP BY p.id, p.name, p.sku
		ORDER BY total_sold DESC
		LIMIT 10
	`, time.Now().AddDate(0, 0, -30)).Rows()

	if err == nil {
		defer productRows.Close()
		for productRows.Next() {
			var product ProductSalesData
			if err := productRows.Scan(&product.ProductID, &product.ProductName, &product.SKU, &product.TotalSold, &product.Revenue, &product.OrderCount); err != nil {
				continue
			}
			analytics.TopSellingProducts = append(analytics.TopSellingProducts, product)
		}
	}

	// Category sales
	categoryRows, err := s.db.Raw(`
		SELECT 
			c.id,
			c.name,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			COUNT(DISTINCT o.id) as order_count,
			COUNT(DISTINCT p.id) as product_count
		FROM categories c
		LEFT JOIN products p ON c.id = p.category_id
		LEFT JOIN order_items oi ON p.id = oi.product_id
		LEFT JOIN orders o ON oi.order_id = o.id
		WHERE o.created_at >= ? AND o.status NOT IN ('cancelled', 'failed')
		GROUP BY c.id, c.name
		ORDER BY revenue DESC
	`, time.Now().AddDate(0, 0, -30)).Rows()

	if err == nil {
		defer categoryRows.Close()
		for categoryRows.Next() {
			var category CategoryData
			if err := categoryRows.Scan(&category.CategoryID, &category.CategoryName, &category.Revenue, &category.OrderCount, &category.ProductCount); err != nil {
				continue
			}
			analytics.CategorySales = append(analytics.CategorySales, category)
		}
	}

	// Low stock products (if inventory system exists)
	lowStockRows, err := s.db.Raw(`
		SELECT 
			p.id,
			p.name,
			p.sku,
			COALESCE(ii.available_quantity, 0) as current_stock,
			COALESCE(ii.reorder_level, 0) as reorder_level,
			COALESCE(w.name, 'Unknown') as warehouse_name
		FROM products p
		LEFT JOIN inventory_items ii ON p.id = ii.product_id
		LEFT JOIN warehouses w ON ii.warehouse_id = w.id
		WHERE ii.available_quantity <= ii.reorder_level AND ii.available_quantity >= 0
		ORDER BY ii.available_quantity ASC
		LIMIT 20
	`).Rows()

	if err == nil {
		defer lowStockRows.Close()
		for lowStockRows.Next() {
			var lowStock LowStockData
			if err := lowStockRows.Scan(&lowStock.ProductID, &lowStock.ProductName, &lowStock.SKU, &lowStock.CurrentStock, &lowStock.ReorderLevel, &lowStock.WarehouseName); err != nil {
				continue
			}
			analytics.LowStockProducts = append(analytics.LowStockProducts, lowStock)
		}
	}

	// Calculate inventory value (if inventory system exists)
	s.db.Raw("SELECT COALESCE(SUM(quantity * cost_price), 0) FROM inventory_items WHERE status = 'active'").Scan(&analytics.InventoryValue)

	return analytics, nil
}

// GetCustomerAnalytics retrieves customer analytics data
func (s *Service) GetCustomerAnalytics() (*CustomerAnalytics, error) {
	analytics := &CustomerAnalytics{}
	now := time.Now()
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastMonth := thisMonth.AddDate(0, -1, 0)

	// Basic customer counts
	s.db.Raw("SELECT COUNT(*) FROM users").Scan(&analytics.TotalCustomers)
	s.db.Raw("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&analytics.ActiveCustomers)
	s.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ?", thisMonth).Scan(&analytics.NewCustomers)

	// Repeat customers (users with more than 1 order)
	s.db.Raw("SELECT COUNT(DISTINCT user_id) FROM orders WHERE user_id IN (SELECT user_id FROM orders GROUP BY user_id HAVING COUNT(*) > 1)").Scan(&analytics.RepeatCustomers)

	// Customer growth
	var lastMonthCustomers int64
	s.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ? AND created_at < ?", lastMonth, thisMonth).Scan(&lastMonthCustomers)
	if lastMonthCustomers > 0 {
		analytics.CustomerGrowth = float64(analytics.NewCustomers-lastMonthCustomers) / float64(lastMonthCustomers) * 100
	}

	// Top customers by total spent
	customerRows, err := s.db.Raw(`
		SELECT 
			u.id,
			CONCAT(u.first_name, ' ', u.last_name) as customer_name,
			u.email,
			COALESCE(SUM(o.total_amount), 0) as total_spent,
			COUNT(o.id) as order_count,
			MAX(o.created_at) as last_order
		FROM users u
		LEFT JOIN orders o ON u.id = o.user_id
		WHERE o.status NOT IN ('cancelled', 'failed')
		GROUP BY u.id, u.first_name, u.last_name, u.email
		ORDER BY total_spent DESC
		LIMIT 10
	`).Rows()

	if err == nil {
		defer customerRows.Close()
		for customerRows.Next() {
			var customer CustomerData
			if err := customerRows.Scan(&customer.UserID, &customer.CustomerName, &customer.Email, &customer.TotalSpent, &customer.OrderCount, &customer.LastOrder); err != nil {
				continue
			}
			analytics.TopCustomers = append(analytics.TopCustomers, customer)
		}
	}

	// Customers by location (using shipping addresses)
	locationRows, err := s.db.Raw(`
		SELECT 
			COALESCE(country, 'Unknown') as country,
			COALESCE(state, 'Unknown') as state,
			COALESCE(city, 'Unknown') as city,
			COUNT(DISTINCT user_id) as customer_count,
			COALESCE(SUM(total_amount), 0) as revenue
		FROM orders o
		WHERE o.status NOT IN ('cancelled', 'failed')
		GROUP BY country, state, city
		ORDER BY customer_count DESC
		LIMIT 20
	`).Rows()

	if err == nil {
		defer locationRows.Close()
		for locationRows.Next() {
			var location LocationData
			if err := locationRows.Scan(&location.Country, &location.State, &location.City, &location.CustomerCount, &location.Revenue); err != nil {
				continue
			}
			analytics.CustomersByLocation = append(analytics.CustomersByLocation, location)
		}
	}

	// Customer lifetime value (average)
	if analytics.TotalCustomers > 0 {
		var totalRevenue int64
		s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status NOT IN ('cancelled', 'failed')").Scan(&totalRevenue)
		analytics.CustomerLifetimeValue = totalRevenue / analytics.TotalCustomers
	}

	return analytics, nil
}

// GetRevenueAnalytics retrieves revenue analytics data
func (s *Service) GetRevenueAnalytics(days int) (*RevenueAnalytics, error) {
	analytics := &RevenueAnalytics{}

	// Default to 30 days if not specified
	if days <= 0 {
		days = 30
	}

	startDate := time.Now().AddDate(0, 0, -days)
	thisMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())
	lastMonth := thisMonth.AddDate(0, -1, 0)

	// Total revenue
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status NOT IN ('cancelled', 'failed')").Scan(&analytics.TotalRevenue)

	// Revenue growth (this month vs last month)
	var thisMonthRevenue, lastMonthRevenue int64
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE created_at >= ? AND status NOT IN ('cancelled', 'failed')", thisMonth).Scan(&thisMonthRevenue)
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE created_at >= ? AND created_at < ? AND status NOT IN ('cancelled', 'failed')", lastMonth, thisMonth).Scan(&lastMonthRevenue)

	if lastMonthRevenue > 0 {
		analytics.RevenueGrowth = float64(thisMonthRevenue-lastMonthRevenue) / float64(lastMonthRevenue) * 100
	}

	// Revenue by period (daily for the specified period)
	revenueRows, err := s.db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COALESCE(SUM(total_amount), 0) as value
		FROM orders 
		WHERE created_at >= ? AND status NOT IN ('cancelled', 'failed')
		GROUP BY DATE(created_at)
		ORDER BY date
	`, startDate).Rows()

	if err == nil {
		defer revenueRows.Close()
		for revenueRows.Next() {
			var data TimeSeriesData
			if err := revenueRows.Scan(&data.Date, &data.Value); err != nil {
				continue
			}
			analytics.RevenueByPeriod = append(analytics.RevenueByPeriod, data)
		}
	}

	// Revenue by category
	categoryRows, err := s.db.Raw(`
		SELECT 
			c.id,
			c.name,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			COUNT(DISTINCT o.id) as order_count,
			COUNT(DISTINCT p.id) as product_count
		FROM categories c
		LEFT JOIN products p ON c.id = p.category_id
		LEFT JOIN order_items oi ON p.id = oi.product_id
		LEFT JOIN orders o ON oi.order_id = o.id
		WHERE o.created_at >= ? AND o.status NOT IN ('cancelled', 'failed')
		GROUP BY c.id, c.name
		ORDER BY revenue DESC
	`, startDate).Rows()

	if err == nil {
		defer categoryRows.Close()
		for categoryRows.Next() {
			var category CategoryData
			if err := categoryRows.Scan(&category.CategoryID, &category.CategoryName, &category.Revenue, &category.OrderCount, &category.ProductCount); err != nil {
				continue
			}
			analytics.RevenueByCategory = append(analytics.RevenueByCategory, category)
		}
	}

	// Revenue by product (top 10)
	productRows, err := s.db.Raw(`
		SELECT 
			p.id,
			p.name,
			p.sku,
			COALESCE(SUM(oi.quantity), 0) as total_sold,
			COALESCE(SUM(oi.total_price), 0) as revenue,
			COUNT(DISTINCT o.id) as order_count
		FROM products p
		LEFT JOIN order_items oi ON p.id = oi.product_id
		LEFT JOIN orders o ON oi.order_id = o.id
		WHERE o.created_at >= ? AND o.status NOT IN ('cancelled', 'failed')
		GROUP BY p.id, p.name, p.sku
		ORDER BY revenue DESC
		LIMIT 10
	`, startDate).Rows()

	if err == nil {
		defer productRows.Close()
		for productRows.Next() {
			var product ProductSalesData
			if err := productRows.Scan(&product.ProductID, &product.ProductName, &product.SKU, &product.TotalSold, &product.Revenue, &product.OrderCount); err != nil {
				continue
			}
			analytics.RevenueByProduct = append(analytics.RevenueByProduct, product)
		}
	}

	// Average order value
	var totalOrders int64
	s.db.Raw("SELECT COUNT(*) FROM orders WHERE created_at >= ? AND status NOT IN ('cancelled', 'failed')", startDate).Scan(&totalOrders)
	var periodRevenue int64
	s.db.Raw("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE created_at >= ? AND status NOT IN ('cancelled', 'failed')", startDate).Scan(&periodRevenue)

	if totalOrders > 0 {
		analytics.AvgOrderValue = periodRevenue / totalOrders
	}

	// Revenue targets (you can customize these based on your business goals)
	analytics.RevenueTargets = RevenueTargets{
		DailyTarget:   100000,   // $1000 in cents
		WeeklyTarget:  700000,   // $7000 in cents
		MonthlyTarget: 3000000,  // $30000 in cents
		YearlyTarget:  36000000, // $360000 in cents
	}

	// Calculate achievement percentage (current month vs monthly target)
	if analytics.RevenueTargets.MonthlyTarget > 0 {
		analytics.RevenueTargets.Achievement = float64(thisMonthRevenue) / float64(analytics.RevenueTargets.MonthlyTarget) * 100
	}

	return analytics, nil
}
