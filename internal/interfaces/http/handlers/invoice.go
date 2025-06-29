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
	"gorm.io/gorm"
)

// InvoiceHandler handles invoice-related endpoints
type InvoiceHandler struct {
	orderService *order.Service
	config       *config.Config
	db           *gorm.DB
}

// NewInvoiceHandler creates a new invoice handler
func NewInvoiceHandler(db *gorm.DB, cfg *config.Config) *InvoiceHandler {
	return &InvoiceHandler{
		orderService: order.NewService(db, cfg, nil),
		config:       cfg,
		db:           db,
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

	// Get order with all relationships
	var orderRecord order.Order
	result := h.db.Preload("Items").Where("id = ?", orderID).First(&orderRecord)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found",
		})
		return
	}

	// Ensure user can only access their own orders
	if orderRecord.UserID == nil || *orderRecord.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// Generate simple HTML invoice
	htmlContent := h.generateSimpleInvoice(&orderRecord)

	// Set proper headers for HTML content
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache")

	// Send HTML directly (don't suggest download)
	c.String(http.StatusOK, htmlContent)
}

func (h *InvoiceHandler) generateSimpleInvoice(orderRecord *order.Order) string {
	// Calculate totals
	subtotal := float64(orderRecord.SubtotalAmount) / 100
	tax := float64(orderRecord.TaxAmount) / 100
	shipping := float64(orderRecord.ShippingAmount) / 100
	discount := float64(orderRecord.DiscountAmount) / 100
	total := float64(orderRecord.TotalAmount) / 100

	// Generate simple HTML
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Invoice - %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: Arial, sans-serif; 
            line-height: 1.6; 
            color: #333; 
            max-width: 800px; 
            margin: 0 auto; 
            padding: 20px;
        }
        .header { 
            text-align: center; 
            border-bottom: 3px solid #007bff; 
            padding-bottom: 20px; 
            margin-bottom: 30px; 
        }
        .company-name { 
            font-size: 32px; 
            font-weight: bold; 
            color: #007bff; 
            margin-bottom: 10px; 
        }
        .invoice-title { 
            font-size: 24px; 
            color: #666; 
        }
        .invoice-info { 
            display: flex; 
            justify-content: space-between; 
            margin-bottom: 30px; 
        }
        .invoice-details, .customer-details { 
            flex: 1; 
        }
        .customer-details { 
            margin-left: 40px; 
        }
        .section-title { 
            font-weight: bold; 
            font-size: 18px; 
            margin-bottom: 10px; 
            color: #007bff; 
        }
        .items-table { 
            width: 100%%; 
            border-collapse: collapse; 
            margin: 30px 0; 
        }
        .items-table th, .items-table td { 
            border: 1px solid #ddd; 
            padding: 12px; 
            text-align: left; 
        }
        .items-table th { 
            background-color: #f8f9fa; 
            font-weight: bold; 
        }
        .items-table .text-right { 
            text-align: right; 
        }
        .totals { 
            float: right; 
            width: 300px; 
            margin-top: 20px; 
        }
        .totals table { 
            width: 100%%; 
            border-collapse: collapse; 
        }
        .totals td { 
            padding: 8px; 
            border-bottom: 1px solid #eee; 
        }
        .totals .total-row { 
            font-weight: bold; 
            font-size: 18px; 
            border-top: 2px solid #007bff; 
            background-color: #f8f9fa; 
        }
        .footer { 
            clear: both; 
            text-align: center; 
            margin-top: 50px; 
            padding-top: 20px; 
            border-top: 1px solid #eee; 
            color: #666; 
        }
        .print-btn { 
            background: #007bff; 
            color: white; 
            border: none; 
            padding: 10px 20px; 
            border-radius: 5px; 
            cursor: pointer; 
            margin-top: 20px; 
        }
        .print-btn:hover { 
            background: #0056b3; 
        }
        @media print { 
            .print-btn { 
                display: none; 
            } 
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="company-name">%s</div>
        <div class="invoice-title">INVOICE</div>
    </div>

    <div class="invoice-info">
        <div class="invoice-details">
            <div class="section-title">Invoice Details</div>
            <p><strong>Invoice #:</strong> INV-%s</p>
            <p><strong>Order #:</strong> %s</p>
            <p><strong>Date:</strong> %s</p>
            <p><strong>Status:</strong> %s</p>
        </div>
        <div class="customer-details">
            <div class="section-title">Bill To</div>
            <p><strong>%s %s</strong></p>
            <p>%s</p>
            %s
            <p>%s, %s %s</p>
            <p>%s</p>
            <p>Email: %s</p>
        </div>
    </div>

    <table class="items-table">
        <thead>
            <tr>
                <th>Item</th>
                <th>SKU</th>
                <th class="text-right">Qty</th>
                <th class="text-right">Price</th>
                <th class="text-right">Total</th>
            </tr>
        </thead>
        <tbody>`,
		orderRecord.OrderNumber,
		h.getCompanyName(),
		orderRecord.OrderNumber,
		orderRecord.OrderNumber,
		time.Now().Format("January 2, 2006"),
		string(orderRecord.Status),
		orderRecord.BillingAddress.FirstName,
		orderRecord.BillingAddress.LastName,
		orderRecord.BillingAddress.AddressLine1,
		func() string {
			if orderRecord.BillingAddress.AddressLine2 != "" {
				return fmt.Sprintf("<p>%s</p>", orderRecord.BillingAddress.AddressLine2)
			}
			return ""
		}(),
		orderRecord.BillingAddress.City,
		orderRecord.BillingAddress.State,
		orderRecord.BillingAddress.PostalCode,
		orderRecord.BillingAddress.Country,
		orderRecord.Email,
	)

	// Add items
	for _, item := range orderRecord.Items {
		itemPrice := float64(item.Price) / 100
		itemTotal := float64(item.TotalPrice) / 100

		variantInfo := ""
		if item.VariantTitle != "" {
			variantInfo = fmt.Sprintf("<br><small>%s</small>", item.VariantTitle)
		}

		html += fmt.Sprintf(`
            <tr>
                <td><strong>%s</strong>%s</td>
                <td>%s</td>
                <td class="text-right">%d</td>
                <td class="text-right">$%.2f</td>
                <td class="text-right">$%.2f</td>
            </tr>`,
			item.Name,
			variantInfo,
			item.SKU,
			item.Quantity,
			itemPrice,
			itemTotal,
		)
	}

	// Close items table and add totals
	html += `
        </tbody>
    </table>

    <div class="totals">
        <table>
            <tr>
                <td>Subtotal:</td>
                <td class="text-right">$` + fmt.Sprintf("%.2f", subtotal) + `</td>
            </tr>`

	// Add discount if exists
	if discount > 0 {
		html += `
            <tr>
                <td>Discount:</td>
                <td class="text-right">-$` + fmt.Sprintf("%.2f", discount) + `</td>
            </tr>`
	}

	html += `
            <tr>
                <td>Shipping:</td>
                <td class="text-right">$` + fmt.Sprintf("%.2f", shipping) + `</td>
            </tr>
            <tr>
                <td>Tax:</td>
                <td class="text-right">$` + fmt.Sprintf("%.2f", tax) + `</td>
            </tr>
            <tr class="total-row">
                <td><strong>Total:</strong></td>
                <td class="text-right"><strong>$` + fmt.Sprintf("%.2f", total) + `</strong></td>
            </tr>
        </table>
    </div>

    <div class="footer">
        <p>Thank you for your business!</p>
        <p>Questions? Contact us at info@yourcompany.com</p>
        <button class="print-btn" onclick="window.print()">Print Invoice</button>
    </div>
</body>
</html>`

	return html
}

func (h *InvoiceHandler) getCompanyName() string {
	if h.config != nil && h.config.App.Name != "" {
		return h.config.App.Name
	}
	return "Your Company Name"
}
