// internal/interfaces/http/handlers/analytics.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/analytics"
	"gorm.io/gorm"
)

// AnalyticsHandler handles analytics endpoints
type AnalyticsHandler struct {
	analyticsService *analytics.Service
	config           *config.Config
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(db *gorm.DB, cfg *config.Config) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analytics.NewService(db, cfg),
		config:           cfg,
	}
}

// GetDashboard handles GET /admin/analytics/dashboard
func (h *AnalyticsHandler) GetDashboard(c *gin.Context) {
	stats, err := h.analyticsService.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve dashboard statistics",
		})
		return
	}

	// Format currency values for display
	formattedStats := map[string]interface{}{
		// Revenue metrics
		"total_revenue":      formatCurrency(stats.TotalRevenue),
		"revenue_today":      formatCurrency(stats.RevenueToday),
		"revenue_this_week":  formatCurrency(stats.RevenueThisWeek),
		"revenue_this_month": formatCurrency(stats.RevenueThisMonth),
		"revenue_growth":     roundFloat(stats.RevenueGrowth, 2),

		// Order metrics
		"total_orders":      stats.TotalOrders,
		"orders_today":      stats.OrdersToday,
		"orders_this_week":  stats.OrdersThisWeek,
		"orders_this_month": stats.OrdersThisMonth,
		"order_growth":      roundFloat(stats.OrderGrowth, 2),

		// User metrics
		"total_users":          stats.TotalUsers,
		"active_users":         stats.ActiveUsers,
		"new_users_today":      stats.NewUsersToday,
		"new_users_this_week":  stats.NewUsersThisWeek,
		"new_users_this_month": stats.NewUsersThisMonth,
		"user_growth":          roundFloat(stats.UserGrowth, 2),

		// Product metrics
		"total_products":        stats.TotalProducts,
		"active_products":       stats.ActiveProducts,
		"out_of_stock_products": stats.OutOfStockProducts,
		"low_stock_products":    stats.LowStockProducts,

		// Conversion metrics
		"conversion_rate":      roundFloat(stats.ConversionRate, 2),
		"avg_order_value":      formatCurrency(stats.AvgOrderValue),
		"repeat_customer_rate": roundFloat(stats.RepeatCustomerRate, 2),

		// Raw values for calculations
		"raw": stats,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Dashboard statistics retrieved successfully",
		"data":    formattedStats,
	})
}

// GetSales handles GET /admin/analytics/sales
func (h *AnalyticsHandler) GetSales(c *gin.Context) {
	// Get days parameter (default 30 days)
	daysParam := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysParam)
	if err != nil || days <= 0 {
		days = 30
	}

	// Limit to reasonable range
	if days > 365 {
		days = 365
	}

	salesData, err := h.analyticsService.GetSalesAnalytics(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve sales analytics",
		})
		return
	}

	// Format currency values
	formattedData := map[string]interface{}{
		"period_days":     days,
		"total_sales":     salesData.TotalSales,
		"total_revenue":   formatCurrency(salesData.TotalRevenue),
		"avg_order_value": formatCurrency(salesData.AvgOrderValue),
		"revenue_growth":  roundFloat(salesData.RevenueGrowth, 2),
		"sales_growth":    roundFloat(salesData.SalesGrowth, 2),
		"daily_revenue":   formatTimeSeriesData(salesData.DailyRevenue),
		"weekly_revenue":  formatTimeSeriesData(salesData.WeeklyRevenue),
		"monthly_revenue": formatTimeSeriesData(salesData.MonthlyRevenue),
		"top_products":    formatProductSalesData(salesData.TopProducts),
		"sales_by_status": salesData.SalesByStatus,
		"raw":             salesData,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sales analytics retrieved successfully",
		"data":    formattedData,
	})
}

// GetProducts handles GET /admin/analytics/products
func (h *AnalyticsHandler) GetProducts(c *gin.Context) {
	productData, err := h.analyticsService.GetProductAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve product analytics",
		})
		return
	}

	// Format currency values
	formattedData := map[string]interface{}{
		"total_products":       productData.TotalProducts,
		"active_products":      productData.ActiveProducts,
		"inventory_value":      formatCurrency(productData.InventoryValue),
		"top_selling_products": formatProductSalesData(productData.TopSellingProducts),
		"category_sales":       formatCategoryData(productData.CategorySales),
		"low_stock_products":   productData.LowStockProducts,
		"product_views":        productData.ProductViews,
		"raw":                  productData,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product analytics retrieved successfully",
		"data":    formattedData,
	})
}

// GetCustomers handles GET /admin/analytics/customers
func (h *AnalyticsHandler) GetCustomers(c *gin.Context) {
	customerData, err := h.analyticsService.GetCustomerAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve customer analytics",
		})
		return
	}

	// Format currency values
	formattedData := map[string]interface{}{
		"total_customers":         customerData.TotalCustomers,
		"active_customers":        customerData.ActiveCustomers,
		"new_customers":           customerData.NewCustomers,
		"repeat_customers":        customerData.RepeatCustomers,
		"customer_growth":         roundFloat(customerData.CustomerGrowth, 2),
		"customer_lifetime_value": formatCurrency(customerData.CustomerLifetimeValue),
		"top_customers":           formatCustomerData(customerData.TopCustomers),
		"customers_by_location":   formatLocationData(customerData.CustomersByLocation),
		"raw":                     customerData,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Customer analytics retrieved successfully",
		"data":    formattedData,
	})
}

// GetRevenue handles GET /admin/analytics/revenue
func (h *AnalyticsHandler) GetRevenue(c *gin.Context) {
	// Get days parameter (default 30 days)
	daysParam := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysParam)
	if err != nil || days <= 0 {
		days = 30
	}

	// Limit to reasonable range
	if days > 365 {
		days = 365
	}

	revenueData, err := h.analyticsService.GetRevenueAnalytics(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve revenue analytics",
		})
		return
	}

	// Format currency values
	formattedData := map[string]interface{}{
		"period_days":         days,
		"total_revenue":       formatCurrency(revenueData.TotalRevenue),
		"revenue_growth":      roundFloat(revenueData.RevenueGrowth, 2),
		"avg_order_value":     formatCurrency(revenueData.AvgOrderValue),
		"revenue_by_period":   formatTimeSeriesData(revenueData.RevenueByPeriod),
		"revenue_by_category": formatCategoryData(revenueData.RevenueByCategory),
		"revenue_by_product":  formatProductSalesData(revenueData.RevenueByProduct),
		"revenue_targets": map[string]interface{}{
			"daily_target":   formatCurrency(revenueData.RevenueTargets.DailyTarget),
			"weekly_target":  formatCurrency(revenueData.RevenueTargets.WeeklyTarget),
			"monthly_target": formatCurrency(revenueData.RevenueTargets.MonthlyTarget),
			"yearly_target":  formatCurrency(revenueData.RevenueTargets.YearlyTarget),
			"achievement":    roundFloat(revenueData.RevenueTargets.Achievement, 2),
		},
		"raw": revenueData,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Revenue analytics retrieved successfully",
		"data":    formattedData,
	})
}

// Helper functions for formatting data

// formatCurrency formats cents to currency string
func formatCurrency(cents int64) string {
	return strconv.FormatFloat(float64(cents)/100, 'f', 2, 64)
}

// roundFloat rounds float to specified decimal places
func roundFloat(val float64, precision int) float64 {
	multiplier := float64(1)
	for i := 0; i < precision; i++ {
		multiplier *= 10
	}
	return float64(int(val*multiplier+0.5)) / multiplier
}

// formatTimeSeriesData formats time series data with currency formatting
func formatTimeSeriesData(data []analytics.TimeSeriesData) []map[string]interface{} {
	var formatted []map[string]interface{}
	for _, item := range data {
		formatted = append(formatted, map[string]interface{}{
			"date":      item.Date,
			"value":     formatCurrency(item.Value),
			"value_raw": item.Value,
			"count":     item.Count,
		})
	}
	return formatted
}

// formatProductSalesData formats product sales data with currency formatting
func formatProductSalesData(data []analytics.ProductSalesData) []map[string]interface{} {
	var formatted []map[string]interface{}
	for _, item := range data {
		formatted = append(formatted, map[string]interface{}{
			"product_id":   item.ProductID,
			"product_name": item.ProductName,
			"sku":          item.SKU,
			"total_sold":   item.TotalSold,
			"revenue":      formatCurrency(item.Revenue),
			"revenue_raw":  item.Revenue,
			"order_count":  item.OrderCount,
		})
	}
	return formatted
}

// formatCategoryData formats category data with currency formatting
func formatCategoryData(data []analytics.CategoryData) []map[string]interface{} {
	var formatted []map[string]interface{}
	for _, item := range data {
		formatted = append(formatted, map[string]interface{}{
			"category_id":   item.CategoryID,
			"category_name": item.CategoryName,
			"revenue":       formatCurrency(item.Revenue),
			"revenue_raw":   item.Revenue,
			"order_count":   item.OrderCount,
			"product_count": item.ProductCount,
		})
	}
	return formatted
}

// formatCustomerData formats customer data with currency formatting
func formatCustomerData(data []analytics.CustomerData) []map[string]interface{} {
	var formatted []map[string]interface{}
	for _, item := range data {
		formatted = append(formatted, map[string]interface{}{
			"user_id":         item.UserID,
			"customer_name":   item.CustomerName,
			"email":           item.Email,
			"total_spent":     formatCurrency(item.TotalSpent),
			"total_spent_raw": item.TotalSpent,
			"order_count":     item.OrderCount,
			"last_order":      item.LastOrder,
		})
	}
	return formatted
}

// formatLocationData formats location data with currency formatting
func formatLocationData(data []analytics.LocationData) []map[string]interface{} {
	var formatted []map[string]interface{}
	for _, item := range data {
		formatted = append(formatted, map[string]interface{}{
			"country":        item.Country,
			"state":          item.State,
			"city":           item.City,
			"customer_count": item.CustomerCount,
			"revenue":        formatCurrency(item.Revenue),
			"revenue_raw":    item.Revenue,
		})
	}
	return formatted
}
