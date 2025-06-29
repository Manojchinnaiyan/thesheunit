// internal/pkg/pdf/service.go
package pdf

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/order"
)

// Service handles PDF generation
type Service struct {
	config *config.Config
}

// NewService creates a new PDF service
func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
	}
}

// GenerateInvoice generates a PDF invoice for an order
func (s *Service) GenerateInvoice(order *order.Order) (*bytes.Buffer, error) {
	// Prepare template data
	data := InvoiceData{
		InvoiceNumber: fmt.Sprintf("INV-%s", order.OrderNumber),
		InvoiceDate:   time.Now().Format("January 2, 2006"),
		DueDate:       time.Now().AddDate(0, 0, 30).Format("January 2, 2006"),
		Order:         order,
		Company: CompanyInfo{
			Name:    s.config.App.CompanyName,
			Address: s.config.App.CompanyAddress,
			Phone:   s.config.App.CompanyPhone,
			Email:   s.config.App.CompanyEmail,
			Website: s.config.App.CompanyWebsite,
		},
	}

	// Generate HTML from template
	htmlContent, err := s.generateHTML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to generate HTML: %w", err)
	}

	// Convert HTML to PDF
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF generator: %w", err)
	}

	// Set PDF options
	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.Grayscale.Set(false)

	// Add page from HTML content
	page := wkhtmltopdf.NewPageReader(bytes.NewReader([]byte(htmlContent)))
	page.FooterRight.Set("[page]")
	page.FooterFontSize.Set(9)
	page.Zoom.Set(0.95)

	pdfg.AddPage(page)

	// Create PDF
	err = pdfg.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF: %w", err)
	}

	return bytes.NewBuffer(pdfg.Bytes()), nil
}

// generateHTML generates HTML content from template
func (s *Service) generateHTML(data InvoiceData) (string, error) {
	tmpl := template.Must(template.New("invoice").Parse(invoiceTemplate))

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// InvoiceData represents the data passed to the invoice template
type InvoiceData struct {
	InvoiceNumber string       `json:"invoice_number"`
	InvoiceDate   string       `json:"invoice_date"`
	DueDate       string       `json:"due_date"`
	Order         *order.Order `json:"order"`
	Company       CompanyInfo  `json:"company"`
}

// CompanyInfo represents company information
type CompanyInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Website string `json:"website"`
}

// Invoice HTML template
const invoiceTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Invoice {{.InvoiceNumber}}</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            color: #333;
        }
        .header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 30px;
            border-bottom: 2px solid #eee;
            padding-bottom: 20px;
        }
        .company-info {
            flex: 1;
        }
        .invoice-info {
            text-align: right;
            flex: 1;
        }
        .invoice-title {
            font-size: 28px;
            font-weight: bold;
            color: #2563eb;
            margin-bottom: 10px;
        }
        .invoice-details {
            margin-bottom: 30px;
        }
        .invoice-details table {
            width: 100%;
        }
        .invoice-details td {
            padding: 5px 0;
            vertical-align: top;
        }
        .invoice-details .label {
            font-weight: bold;
            width: 150px;
        }
        .billing-shipping {
            display: flex;
            justify-content: space-between;
            margin-bottom: 30px;
        }
        .billing-info, .shipping-info {
            flex: 1;
            margin-right: 20px;
        }
        .section-title {
            font-size: 16px;
            font-weight: bold;
            margin-bottom: 10px;
            color: #374151;
        }
        .items-table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 30px;
        }
        .items-table th,
        .items-table td {
            border: 1px solid #ddd;
            padding: 12px 8px;
            text-align: left;
        }
        .items-table th {
            background-color: #f8f9fa;
            font-weight: bold;
        }
        .items-table .qty-col,
        .items-table .price-col,
        .items-table .total-col {
            text-align: right;
            width: 80px;
        }
        .totals {
            float: right;
            width: 300px;
        }
        .totals table {
            width: 100%;
            border-collapse: collapse;
        }
        .totals td {
            padding: 8px;
            border-bottom: 1px solid #eee;
        }
        .totals .label {
            text-align: right;
            font-weight: bold;
        }
        .totals .amount {
            text-align: right;
            width: 100px;
        }
        .total-row {
            font-size: 18px;
            font-weight: bold;
            border-top: 2px solid #333 !important;
        }
        .footer {
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            text-align: center;
            color: #666;
            font-size: 12px;
        }
        .status-badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: bold;
            text-transform: uppercase;
        }
        .status-paid {
            background-color: #dcfce7;
            color: #166534;
        }
        .status-pending {
            background-color: #fef3c7;
            color: #92400e;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="company-info">
            <h1>{{.Company.Name}}</h1>
            <p>{{.Company.Address}}</p>
            <p>Phone: {{.Company.Phone}}</p>
            <p>Email: {{.Company.Email}}</p>
            <p>{{.Company.Website}}</p>
        </div>
        <div class="invoice-info">
            <div class="invoice-title">INVOICE</div>
            <p><strong>Invoice #:</strong> {{.InvoiceNumber}}</p>
            <p><strong>Invoice Date:</strong> {{.InvoiceDate}}</p>
            <p><strong>Due Date:</strong> {{.DueDate}}</p>
            <p><strong>Order #:</strong> {{.Order.OrderNumber}}</p>
        </div>
    </div>

    <div class="invoice-details">
        <table>
            <tr>
                <td class="label">Order Date:</td>
                <td>{{.Order.CreatedAt.Format "January 2, 2006"}}</td>
                <td class="label" style="text-align: right;">Payment Status:</td>
                <td style="text-align: right;">
                    <span class="status-badge {{if eq .Order.PaymentStatus "paid"}}status-paid{{else}}status-pending{{end}}">
                        {{.Order.PaymentStatus}}
                    </span>
                </td>
            </tr>
            <tr>
                <td class="label">Order Status:</td>
                <td>{{.Order.Status}}</td>
                <td class="label" style="text-align: right;">Currency:</td>
                <td style="text-align: right;">{{.Order.Currency}}</td>
            </tr>
        </table>
    </div>

    <div class="billing-shipping">
        <div class="billing-info">
            <div class="section-title">Bill To:</div>
            <p><strong>{{.Order.BillingAddress.FirstName}} {{.Order.BillingAddress.LastName}}</strong></p>
            {{if .Order.BillingAddress.Company}}<p>{{.Order.BillingAddress.Company}}</p>{{end}}
            <p>{{.Order.BillingAddress.AddressLine1}}</p>
            {{if .Order.BillingAddress.AddressLine2}}<p>{{.Order.BillingAddress.AddressLine2}}</p>{{end}}
            <p>{{.Order.BillingAddress.City}}, {{.Order.BillingAddress.State}} {{.Order.BillingAddress.PostalCode}}</p>
            <p>{{.Order.BillingAddress.Country}}</p>
            {{if .Order.BillingAddress.Phone}}<p>Phone: {{.Order.BillingAddress.Phone}}</p>{{end}}
            <p>Email: {{.Order.Email}}</p>
        </div>
        <div class="shipping-info">
            <div class="section-title">Ship To:</div>
            <p><strong>{{.Order.ShippingAddress.FirstName}} {{.Order.ShippingAddress.LastName}}</strong></p>
            {{if .Order.ShippingAddress.Company}}<p>{{.Order.ShippingAddress.Company}}</p>{{end}}
            <p>{{.Order.ShippingAddress.AddressLine1}}</p>
            {{if .Order.ShippingAddress.AddressLine2}}<p>{{.Order.ShippingAddress.AddressLine2}}</p>{{end}}
            <p>{{.Order.ShippingAddress.City}}, {{.Order.ShippingAddress.State}} {{.Order.ShippingAddress.PostalCode}}</p>
            <p>{{.Order.ShippingAddress.Country}}</p>
            {{if .Order.ShippingAddress.Phone}}<p>Phone: {{.Order.ShippingAddress.Phone}}</p>{{end}}
        </div>
    </div>

    <table class="items-table">
        <thead>
            <tr>
                <th>Item</th>
                <th>SKU</th>
                <th class="qty-col">Qty</th>
                <th class="price-col">Price</th>
                <th class="total-col">Total</th>
            </tr>
        </thead>
        <tbody>
            {{range .Order.Items}}
            <tr>
                <td>
                    <strong>{{.Name}}</strong>
                    {{if .VariantTitle}}<br><small>{{.VariantTitle}}</small>{{end}}
                </td>
                <td>{{.SKU}}</td>
                <td class="qty-col">{{.Quantity}}</td>
                <td class="price-col">${{printf "%.2f" (div (float64 .Price) 100)}}</td>
                <td class="total-col">${{printf "%.2f" (div (float64 .TotalPrice) 100)}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <div class="totals">
        <table>
            <tr>
                <td class="label">Subtotal:</td>
                <td class="amount">${{printf "%.2f" (div (float64 .Order.SubtotalAmount) 100)}}</td>
            </tr>
            {{if gt .Order.DiscountAmount 0}}
            <tr>
                <td class="label">Discount:</td>
                <td class="amount">-${{printf "%.2f" (div (float64 .Order.DiscountAmount) 100)}}</td>
            </tr>
            {{end}}
            <tr>
                <td class="label">Shipping:</td>
                <td class="amount">${{printf "%.2f" (div (float64 .Order.ShippingAmount) 100)}}</td>
            </tr>
            <tr>
                <td class="label">Tax:</td>
                <td class="amount">${{printf "%.2f" (div (float64 .Order.TaxAmount) 100)}}</td>
            </tr>
            <tr class="total-row">
                <td class="label">Total:</td>
                <td class="amount">${{printf "%.2f" (div (float64 .Order.TotalAmount) 100)}}</td>
            </tr>
        </table>
    </div>

    <div style="clear: both;"></div>

    <div class="footer">
        <p>Thank you for your business!</p>
        <p>If you have any questions about this invoice, please contact us at {{.Company.Email}} or {{.Company.Phone}}</p>
    </div>
</body>
</html>
`
