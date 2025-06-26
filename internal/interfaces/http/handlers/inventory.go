// internal/interfaces/http/handlers/inventory.go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/your-org/ecommerce-backend/internal/config"
	"github.com/your-org/ecommerce-backend/internal/domain/inventory"
	"github.com/your-org/ecommerce-backend/internal/interfaces/http/middleware"
	"gorm.io/gorm"
)

// InventoryHandler handles inventory endpoints
type InventoryHandler struct {
	inventoryService *inventory.Service
	config           *config.Config
}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler(db *gorm.DB, cfg *config.Config) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventory.NewService(db, cfg),
		config:           cfg,
	}
}

// WAREHOUSE ENDPOINTS

// CreateWarehouse handles POST /admin/warehouses
func (h *InventoryHandler) CreateWarehouse(c *gin.Context) {
	var req inventory.CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	warehouse, err := h.inventoryService.CreateWarehouse(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Warehouse created successfully",
		"data":    warehouse,
	})
}

// GetWarehouses handles GET /inventory/warehouses
func (h *InventoryHandler) GetWarehouses(c *gin.Context) {
	warehouses, err := h.inventoryService.GetWarehouses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve warehouses",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Warehouses retrieved successfully",
		"data":    warehouses,
	})
}

// GetDefaultWarehouse handles GET /inventory/warehouses/default
func (h *InventoryHandler) GetDefaultWarehouse(c *gin.Context) {
	warehouse, err := h.inventoryService.GetDefaultWarehouse()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Default warehouse retrieved successfully",
		"data":    warehouse,
	})
}

// INVENTORY MANAGEMENT ENDPOINTS

// GetInventoryItem handles GET /admin/inventory/:productId/:warehouseId
func (h *InventoryHandler) GetInventoryItem(c *gin.Context) {
	productIDParam := c.Param("productId")
	warehouseIDParam := c.Param("warehouseId")

	productID, err := strconv.ParseUint(productIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	warehouseID, err := strconv.ParseUint(warehouseIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid warehouse ID",
		})
		return
	}

	item, err := h.inventoryService.GetInventoryItem(uint(productID), uint(warehouseID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Inventory item retrieved successfully",
		"data":    item,
	})
}

// CreateOrUpdateInventoryItem handles POST /admin/inventory
func (h *InventoryHandler) CreateOrUpdateInventoryItem(c *gin.Context) {
	var req struct {
		ProductID       uint   `json:"product_id" binding:"required"`
		WarehouseID     uint   `json:"warehouse_id" binding:"required"`
		SKU             string `json:"sku" binding:"required"`
		InitialQuantity int    `json:"initial_quantity"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	item, err := h.inventoryService.CreateOrUpdateInventoryItem(req.ProductID, req.WarehouseID, req.SKU, req.InitialQuantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Inventory item created/updated successfully",
		"data":    item,
	})
}

// STOCK MOVEMENT ENDPOINTS

// RecordStockMovement handles POST /admin/inventory/movements
func (h *InventoryHandler) RecordStockMovement(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req inventory.StockMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	movement, err := h.inventoryService.RecordStockMovement(&req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Stock movement recorded successfully",
		"data":    movement,
	})
}

// RESERVATION ENDPOINTS

// ReserveStock handles POST /inventory/reserve
func (h *InventoryHandler) ReserveStock(c *gin.Context) {
	var req inventory.ReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	reservation, err := h.inventoryService.ReserveStock(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Stock reserved successfully",
		"data":    reservation,
	})
}

// ReleaseReservation handles POST /inventory/release
func (h *InventoryHandler) ReleaseReservation(c *gin.Context) {
	var req struct {
		OrderID     uint `json:"order_id" binding:"required"`
		OrderItemID uint `json:"order_item_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err := h.inventoryService.ReleaseReservation(req.OrderID, req.OrderItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stock reservation released successfully",
	})
}

// FulfillReservation handles POST /inventory/fulfill
func (h *InventoryHandler) FulfillReservation(c *gin.Context) {
	var req struct {
		OrderID     uint `json:"order_id" binding:"required"`
		OrderItemID uint `json:"order_item_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err := h.inventoryService.FulfillReservation(req.OrderID, req.OrderItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stock reservation fulfilled successfully",
	})
}

// UTILITY ENDPOINTS

// GetStockLevel handles GET /inventory/stock-level/:productId
func (h *InventoryHandler) GetStockLevel(c *gin.Context) {
	productIDParam := c.Param("productId")
	productID, err := strconv.ParseUint(productIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	var warehouseID *uint
	if warehouseIDParam := c.Query("warehouse_id"); warehouseIDParam != "" {
		if whID, err := strconv.ParseUint(warehouseIDParam, 10, 32); err == nil {
			whIDUint := uint(whID)
			warehouseID = &whIDUint
		}
	}

	stockLevel, err := h.inventoryService.GetStockLevel(uint(productID), warehouseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve stock level",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stock level retrieved successfully",
		"data": gin.H{
			"product_id":   productID,
			"warehouse_id": warehouseID,
			"stock_level":  stockLevel,
		},
	})
}
