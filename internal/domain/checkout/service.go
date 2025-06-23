// internal/domain/checkout/service.go
package checkout

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/cart"
	"github.com/your-org/ecommerce-backend/internal/domain/user"
	"gorm.io/gorm"
)

// Service handles checkout business logic
type Service struct {
	db          *gorm.DB
	redisClient *redis.Client
	config      *config.Config
	cartService *cart.Service
}

// NewService creates a new checkout service
func NewService(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *Service {
	return &Service{
		db:          db,
		redisClient: redisClient,
		config:      cfg,
		cartService: cart.NewService(db, redisClient, cfg),
	}
}

// ShippingMethod represents a shipping option
type ShippingMethod struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Price         int64  `json:"price"` // Price in cents
	EstimatedDays string `json:"estimated_days"`
	Available     bool   `json:"available"`
	Carrier       string `json:"carrier"`
}

// ShippingCalculationRequest represents shipping calculation request
type ShippingCalculationRequest struct {
	ShippingMethodID string `json:"shipping_method_id" binding:"required"`
	AddressID        uint   `json:"address_id" binding:"required"`
}

// ShippingCalculation represents shipping calculation result
type ShippingCalculation struct {
	ShippingMethod ShippingMethod `json:"shipping_method"`
	Cost           int64          `json:"cost"`
	TaxAmount      int64          `json:"tax_amount"`
	TotalCost      int64          `json:"total_cost"`
	EstimatedDays  string         `json:"estimated_days"`
}

// TaxCalculationRequest represents tax calculation request
type TaxCalculationRequest struct {
	AddressID uint   `json:"address_id" binding:"required"`
	Subtotal  *int64 `json:"subtotal,omitempty"`
}

// TaxCalculation represents tax calculation result
type TaxCalculation struct {
	TaxRate       float64        `json:"tax_rate"`       // Tax rate as percentage
	TaxAmount     int64          `json:"tax_amount"`     // Tax amount in cents
	TaxableAmount int64          `json:"taxable_amount"` // Amount subject to tax
	TaxType       string         `json:"tax_type"`       // GST, VAT, Sales Tax, etc.
	Breakdown     []TaxBreakdown `json:"breakdown,omitempty"`
}

// TaxBreakdown represents detailed tax breakdown
type TaxBreakdown struct {
	Type        string  `json:"type"` // CGST, SGST, IGST, etc.
	Rate        float64 `json:"rate"`
	Amount      int64   `json:"amount"`
	Description string  `json:"description"`
}

// CouponApplication represents applied coupon details
type CouponApplication struct {
	CouponCode        string     `json:"coupon_code"`
	DiscountType      string     `json:"discount_type"`       // percentage, fixed_amount
	DiscountValue     float64    `json:"discount_value"`      // Percentage or amount
	DiscountAmount    int64      `json:"discount_amount"`     // Actual discount in cents
	MinOrderAmount    int64      `json:"min_order_amount"`    // Minimum order required
	MaxDiscountAmount int64      `json:"max_discount_amount"` // Maximum discount allowed
	ValidUntil        *time.Time `json:"valid_until,omitempty"`
	Applied           bool       `json:"applied"`
	Message           string     `json:"message,omitempty"`
}

// CheckoutSummary represents complete checkout summary
type CheckoutSummary struct {
	Cart            *cart.CartResponse `json:"cart"`
	ShippingAddress *user.Address      `json:"shipping_address,omitempty"`
	BillingAddress  *user.Address      `json:"billing_address,omitempty"`
	ShippingMethod  *ShippingMethod    `json:"shipping_method,omitempty"`
	Pricing         CheckoutPricing    `json:"pricing"`
	AppliedCoupon   *CouponApplication `json:"applied_coupon,omitempty"`
	PaymentMethods  []PaymentMethod    `json:"payment_methods"`
}

// CheckoutPricing represents pricing breakdown
type CheckoutPricing struct {
	Subtotal       int64 `json:"subtotal"`
	ShippingCost   int64 `json:"shipping_cost"`
	TaxAmount      int64 `json:"tax_amount"`
	DiscountAmount int64 `json:"discount_amount"`
	TotalAmount    int64 `json:"total_amount"`
}

// PaymentMethod represents available payment methods
type PaymentMethod struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	Logo        string `json:"logo,omitempty"`
}

// CheckoutValidationRequest represents checkout validation request
type CheckoutValidationRequest struct {
	ShippingAddressID uint   `json:"shipping_address_id" binding:"required"`
	BillingAddressID  *uint  `json:"billing_address_id,omitempty"`
	ShippingMethodID  string `json:"shipping_method_id" binding:"required"`
	PaymentMethodID   string `json:"payment_method_id" binding:"required"`
	CouponCode        string `json:"coupon_code,omitempty"`
}

// CheckoutValidation represents checkout validation result
type CheckoutValidation struct {
	IsValid        bool             `json:"is_valid"`
	Errors         []string         `json:"errors,omitempty"`
	Warnings       []string         `json:"warnings,omitempty"`
	Summary        *CheckoutSummary `json:"summary,omitempty"`
	EstimatedTotal int64            `json:"estimated_total"`
}

// GetShippingMethods retrieves available shipping methods
func (s *Service) GetShippingMethods(userID uint, addressID *uint, country, state, city *string) ([]ShippingMethod, error) {
	// Get shipping address
	var shippingAddress *user.Address
	var err error

	if addressID != nil {
		addressService := user.NewAddressService(s.db, s.config)
		shippingAddress, err = addressService.GetAddress(userID, *addressID)
		if err != nil {
			return nil, fmt.Errorf("failed to get shipping address: %w", err)
		}
	} else if country != nil && state != nil && city != nil {
		// Use provided location info
		shippingAddress = &user.Address{
			Country: *country,
			State:   *state,
			City:    *city,
		}
	} else {
		// Get user's default shipping address
		addressService := user.NewAddressService(s.db, s.config)
		shippingAddress, err = addressService.GetDefaultAddress(userID, "shipping")
		if err != nil {
			return nil, fmt.Errorf("no shipping address found: %w", err)
		}
	}

	// Get cart to calculate shipping
	userIDPtr := &userID
	cartResponse, err := s.cartService.GetCart(userIDPtr, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(cartResponse.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Calculate shipping methods based on location and cart
	return s.calculateShippingMethods(shippingAddress, cartResponse), nil
}

// CalculateShipping calculates shipping cost for specific method
func (s *Service) CalculateShipping(userID uint, req *ShippingCalculationRequest) (*ShippingCalculation, error) {
	// Get shipping address
	addressService := user.NewAddressService(s.db, s.config)
	address, err := addressService.GetAddress(userID, req.AddressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	// Get shipping method
	methods := s.calculateShippingMethods(address, nil)
	var selectedMethod *ShippingMethod
	for _, method := range methods {
		if method.ID == req.ShippingMethodID {
			selectedMethod = &method
			break
		}
	}

	if selectedMethod == nil {
		return nil, fmt.Errorf("shipping method not found or not available")
	}

	// Calculate tax on shipping if applicable
	taxAmount := s.calculateShippingTax(selectedMethod.Price, address)

	return &ShippingCalculation{
		ShippingMethod: *selectedMethod,
		Cost:           selectedMethod.Price,
		TaxAmount:      taxAmount,
		TotalCost:      selectedMethod.Price + taxAmount,
		EstimatedDays:  selectedMethod.EstimatedDays,
	}, nil
}

// ApplyCoupon applies a coupon code
func (s *Service) ApplyCoupon(userID uint, couponCode string) (*CouponApplication, error) {
	// Get user's cart
	userIDPtr := &userID
	cartResponse, err := s.cartService.GetCart(userIDPtr, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(cartResponse.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Validate and apply coupon
	coupon := s.validateCoupon(couponCode, cartResponse.Totals.SubTotal)
	if !coupon.Applied {
		return coupon, nil
	}

	// Store applied coupon in Redis
	ctx := context.Background()
	couponKey := fmt.Sprintf("applied_coupon:%d", userID)
	couponData, _ := json.Marshal(coupon)
	s.redisClient.Set(ctx, couponKey, couponData, 24*time.Hour)

	return coupon, nil
}

// RemoveCoupon removes applied coupon
func (s *Service) RemoveCoupon(userID uint) error {
	ctx := context.Background()
	couponKey := fmt.Sprintf("applied_coupon:%d", userID)
	return s.redisClient.Del(ctx, couponKey).Err()
}

// CalculateTax calculates tax for the order
func (s *Service) CalculateTax(userID uint, req *TaxCalculationRequest) (*TaxCalculation, error) {
	// Get address for tax calculation
	addressService := user.NewAddressService(s.db, s.config)
	address, err := addressService.GetAddress(userID, req.AddressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	var subtotal int64
	if req.Subtotal != nil {
		subtotal = *req.Subtotal
	} else {
		// Get cart subtotal
		userIDPtr := &userID
		cartResponse, err := s.cartService.GetCart(userIDPtr, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get cart: %w", err)
		}
		subtotal = cartResponse.Totals.SubTotal
	}

	return s.calculateTaxForLocation(subtotal, address), nil
}

// GetCheckoutSummary gets complete checkout summary
func (s *Service) GetCheckoutSummary(userID uint, addressID *uint, shippingMethodID, couponCode string) (*CheckoutSummary, error) {
	// Get cart
	userIDPtr := &userID
	cartResponse, err := s.cartService.GetCart(userIDPtr, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if len(cartResponse.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	summary := &CheckoutSummary{
		Cart: cartResponse,
		Pricing: CheckoutPricing{
			Subtotal: cartResponse.Totals.SubTotal,
		},
		PaymentMethods: s.getAvailablePaymentMethods(),
	}

	// Get addresses
	addressService := user.NewAddressService(s.db, s.config)
	if addressID != nil {
		summary.ShippingAddress, _ = addressService.GetAddress(userID, *addressID)
		summary.BillingAddress = summary.ShippingAddress // Default to same
	} else {
		summary.ShippingAddress, _ = addressService.GetDefaultAddress(userID, "shipping")
		summary.BillingAddress, _ = addressService.GetDefaultAddress(userID, "billing")
	}

	// Calculate shipping
	if shippingMethodID != "" && summary.ShippingAddress != nil {
		methods := s.calculateShippingMethods(summary.ShippingAddress, cartResponse)
		for _, method := range methods {
			if method.ID == shippingMethodID {
				summary.ShippingMethod = &method
				summary.Pricing.ShippingCost = method.Price
				break
			}
		}
	}

	// Calculate tax
	if summary.ShippingAddress != nil {
		taxCalc := s.calculateTaxForLocation(summary.Pricing.Subtotal, summary.ShippingAddress)
		summary.Pricing.TaxAmount = taxCalc.TaxAmount
	}

	// Apply coupon
	if couponCode != "" {
		coupon := s.validateCoupon(couponCode, summary.Pricing.Subtotal)
		if coupon.Applied {
			summary.AppliedCoupon = coupon
			summary.Pricing.DiscountAmount = coupon.DiscountAmount
		}
	} else {
		// Check for stored coupon
		summary.AppliedCoupon = s.getStoredCoupon(userID)
		if summary.AppliedCoupon != nil {
			summary.Pricing.DiscountAmount = summary.AppliedCoupon.DiscountAmount
		}
	}

	// Calculate total
	summary.Pricing.TotalAmount = summary.Pricing.Subtotal +
		summary.Pricing.ShippingCost +
		summary.Pricing.TaxAmount -
		summary.Pricing.DiscountAmount

	return summary, nil
}

// ValidateCheckout validates checkout data
func (s *Service) ValidateCheckout(userID uint, req *CheckoutValidationRequest) (*CheckoutValidation, error) {
	validation := &CheckoutValidation{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Get checkout summary
	summary, err := s.GetCheckoutSummary(userID, &req.ShippingAddressID, req.ShippingMethodID, req.CouponCode)
	if err != nil {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, err.Error())
		return validation, nil
	}

	validation.Summary = summary
	validation.EstimatedTotal = summary.Pricing.TotalAmount

	// Validate addresses
	if summary.ShippingAddress == nil {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, "shipping address is required")
	}

	// Validate shipping method
	if summary.ShippingMethod == nil {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, "shipping method is required")
	} else if !summary.ShippingMethod.Available {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, "selected shipping method is not available")
	}

	// Validate payment method
	paymentMethodValid := false
	for _, pm := range summary.PaymentMethods {
		if pm.ID == req.PaymentMethodID && pm.Available {
			paymentMethodValid = true
			break
		}
	}
	if !paymentMethodValid {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, "invalid or unavailable payment method")
	}

	// Validate cart
	if len(summary.Cart.Items) == 0 {
		validation.IsValid = false
		validation.Errors = append(validation.Errors, "cart is empty")
	}

	// Validate inventory
	for _, item := range summary.Cart.Items {
		if item.Product != nil && item.Product.TrackQuantity {
			availableQuantity := item.Product.Quantity
			if item.ProductVariant != nil {
				availableQuantity = item.ProductVariant.Quantity
			}
			if availableQuantity < item.Quantity {
				validation.Warnings = append(validation.Warnings,
					fmt.Sprintf("Limited stock for %s. Available: %d", item.Product.Name, availableQuantity))
			}
		}
	}

	return validation, nil
}

// Private helper methods

func (s *Service) calculateShippingMethods(address *user.Address, cartResponse *cart.CartResponse) []ShippingMethod {
	methods := []ShippingMethod{
		{
			ID:            "standard",
			Name:          "Standard Shipping",
			Description:   "Regular delivery in 5-7 business days",
			Price:         999, // ₹9.99
			EstimatedDays: "5-7 business days",
			Available:     true,
			Carrier:       "India Post",
		},
		{
			ID:            "express",
			Name:          "Express Shipping",
			Description:   "Fast delivery in 2-3 business days",
			Price:         1999, // ₹19.99
			EstimatedDays: "2-3 business days",
			Available:     true,
			Carrier:       "BlueDart",
		},
	}

	// Add premium options for certain locations
	if address.Country == "IN" && (address.State == "Maharashtra" || address.State == "Delhi" || address.State == "Karnataka") {
		methods = append(methods, ShippingMethod{
			ID:            "same_day",
			Name:          "Same Day Delivery",
			Description:   "Delivery within 24 hours",
			Price:         2999, // ₹29.99
			EstimatedDays: "Same day",
			Available:     true,
			Carrier:       "Dunzo",
		})
	}

	// Free shipping for orders above threshold
	if cartResponse != nil && cartResponse.Totals.SubTotal >= 299900 { // ₹2999
		for i := range methods {
			if methods[i].ID == "standard" {
				methods[i].Price = 0
				methods[i].Description = "Free standard shipping on orders over ₹2999"
			}
		}
	}

	return methods
}

func (s *Service) calculateShippingTax(shippingCost int64, address *user.Address) int64 {
	// Simple tax calculation - in India, shipping is generally taxable
	if address.Country == "IN" {
		return int64(float64(shippingCost) * 0.18) // 18% GST
	}
	return 0
}

func (s *Service) validateCoupon(couponCode string, subtotal int64) *CouponApplication {
	// Mock coupon validation - replace with actual coupon system
	coupons := map[string]CouponApplication{
		"SAVE10": {
			CouponCode:        "SAVE10",
			DiscountType:      "percentage",
			DiscountValue:     10.0,
			MinOrderAmount:    199900, // ₹1999
			MaxDiscountAmount: 149900, // ₹1499
			Applied:           false,
		},
		"FLAT500": {
			CouponCode:     "FLAT500",
			DiscountType:   "fixed_amount",
			DiscountValue:  50000,  // ₹500
			MinOrderAmount: 299900, // ₹2999
			Applied:        false,
		},
		"WELCOME20": {
			CouponCode:        "WELCOME20",
			DiscountType:      "percentage",
			DiscountValue:     20.0,
			MinOrderAmount:    99900,  // ₹999
			MaxDiscountAmount: 199900, // ₹1999
			Applied:           false,
		},
	}

	coupon, exists := coupons[couponCode]
	if !exists {
		return &CouponApplication{
			CouponCode: couponCode,
			Applied:    false,
			Message:    "Invalid coupon code",
		}
	}

	// Check minimum order amount
	if subtotal < coupon.MinOrderAmount {
		coupon.Message = fmt.Sprintf("Minimum order amount of ₹%.2f required", float64(coupon.MinOrderAmount)/100)
		return &coupon
	}

	// Calculate discount
	if coupon.DiscountType == "percentage" {
		coupon.DiscountAmount = int64(float64(subtotal) * coupon.DiscountValue / 100)
		if coupon.MaxDiscountAmount > 0 && coupon.DiscountAmount > coupon.MaxDiscountAmount {
			coupon.DiscountAmount = coupon.MaxDiscountAmount
		}
	} else {
		coupon.DiscountAmount = int64(coupon.DiscountValue)
	}

	coupon.Applied = true
	coupon.Message = fmt.Sprintf("Coupon applied! You saved ₹%.2f", float64(coupon.DiscountAmount)/100)
	return &coupon
}

func (s *Service) calculateTaxForLocation(subtotal int64, address *user.Address) *TaxCalculation {
	// Tax calculation based on Indian GST system
	if address.Country != "IN" {
		return &TaxCalculation{
			TaxRate:       0,
			TaxAmount:     0,
			TaxableAmount: subtotal,
			TaxType:       "No Tax",
		}
	}

	// Standard GST rate for most products
	taxRate := 18.0 // 18% GST
	taxAmount := int64(float64(subtotal) * taxRate / 100)

	breakdown := []TaxBreakdown{
		{
			Type:        "CGST",
			Rate:        9.0,
			Amount:      taxAmount / 2,
			Description: "Central Goods and Services Tax",
		},
		{
			Type:        "SGST",
			Rate:        9.0,
			Amount:      taxAmount / 2,
			Description: "State Goods and Services Tax",
		},
	}

	return &TaxCalculation{
		TaxRate:       taxRate,
		TaxAmount:     taxAmount,
		TaxableAmount: subtotal,
		TaxType:       "GST",
		Breakdown:     breakdown,
	}
}

func (s *Service) getAvailablePaymentMethods() []PaymentMethod {
	return []PaymentMethod{
		{
			ID:          "razorpay",
			Name:        "Razorpay",
			Description: "Pay using Credit Card, Debit Card, NetBanking, UPI, or Wallets",
			Available:   s.config.External.Razorpay.KeyID != "",
			Logo:        "/images/razorpay-logo.png",
		},
		{
			ID:          "cod",
			Name:        "Cash on Delivery",
			Description: "Pay cash when your order is delivered",
			Available:   true,
			Logo:        "/images/cod-logo.png",
		},
		{
			ID:          "wallet",
			Name:        "Digital Wallet",
			Description: "Pay using Paytm, PhonePe, Google Pay",
			Available:   true,
			Logo:        "/images/wallet-logo.png",
		},
	}
}

func (s *Service) getStoredCoupon(userID uint) *CouponApplication {
	ctx := context.Background()
	couponKey := fmt.Sprintf("applied_coupon:%d", userID)

	couponData, err := s.redisClient.Get(ctx, couponKey).Result()
	if err != nil {
		return nil
	}

	var coupon CouponApplication
	if err := json.Unmarshal([]byte(couponData), &coupon); err != nil {
		return nil
	}

	return &coupon
}
