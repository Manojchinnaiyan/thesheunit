// internal/interfaces/http/handlers/invoice.go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/order"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"github.com/your-org/ecommerce-backend/internal/pkg/pdf"
	"gorm.io/gorm"
)

// InvoiceHandler handles invoice-related endpoints
type InvoiceHandler struct {
	orderService *order.Service
	pdfService   *pdf.Service
	config       *config.Config
}

// NewInvoiceHandler creates a new invoice handler
func NewInvoiceHandler(db *gorm.DB, cfg *config.Config) *InvoiceHandler {
	orderService := order.NewService(db, cfg, nil)
	pdfService := pdf.NewService(cfg)

	return &InvoiceHandler{
		orderService: orderService,
		pdfService:   pdfService,
		config:       cfg,
	}
}

// GenerateInvoice handles GET /orders/:id/invoice
func (h *InvoiceHandler) GenerateInvoice(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	// Get order
	order, err := h.orderService.GetOrder(uint(orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found",
		})
		return
	}

	// Ensure user can only access their own orders
	if order.UserID == nil || *order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// Generate PDF invoice
	pdfBuffer, err := h.pdfService.GenerateInvoice(order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate invoice",
		})
		return
	}

	// Set headers for PDF download
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=invoice-%s.pdf", order.OrderNumber))
	c.Header("Content-Length", strconv.Itoa(len(pdfBuffer.Bytes())))

	// Send PDF
	c.Data(http.StatusOK, "application/pdf", pdfBuffer.Bytes())
}

// GetInvoiceData handles GET /orders/:id/invoice/data (for frontend preview)
func (h *InvoiceHandler) GetInvoiceData(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	idParam := c.Param("id")
	orderID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order ID",
		})
		return
	}

	// Get order
	order, err := h.orderService.GetOrder(uint(orderID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found",
		})
		return
	}

	// Ensure user can only access their own orders
	if order.UserID == nil || *order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// Prepare invoice data
	invoiceData := map[string]interface{}{
		"invoice_number": fmt.Sprintf("INV-%s", order.OrderNumber),
		"invoice_date":   time.Now().Format("January 2, 2006"),
		"due_date":       time.Now().AddDate(0, 0, 30).Format("January 2, 2006"),
		"order":          order,
		"company": map[string]interface{}{
			"name":    h.config.App.CompanyName,
			"address": h.config.App.CompanyAddress,
			"phone":   h.config.App.CompanyPhone,
			"email":   h.config.App.CompanyEmail,
			"website": h.config.App.CompanyWebsite,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invoice data retrieved successfully",
		"data":    invoiceData,
	})
}
