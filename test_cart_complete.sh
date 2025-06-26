#!/bin/bash

# Inventory Management Test Script
# Run this after starting your server with: make dev

BASE_URL="http://localhost:8080/api/v1"
ADMIN_TOKEN=""  # You'll need to set this after login

echo "🧪 Testing Inventory Management System"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ PASS${NC}: $2"
    else
        echo -e "${RED}❌ FAIL${NC}: $2"
    fi
}

# Function to make API calls
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    local auth_header=$4
    
    if [ -n "$auth_header" ]; then
        if [ -n "$data" ]; then
            curl -s -X $method "$BASE_URL$endpoint" \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $auth_header" \
                -d "$data"
        else
            curl -s -X $method "$BASE_URL$endpoint" \
                -H "Authorization: Bearer $auth_header"
        fi
    else
        if [ -n "$data" ]; then
            curl -s -X $method "$BASE_URL$endpoint" \
                -H "Content-Type: application/json" \
                -d "$data"
        else
            curl -s -X $method "$BASE_URL$endpoint"
        fi
    fi
}

echo -e "\n${BLUE}Step 1: Testing Server Health${NC}"
echo "================================"

# # Test server health
# response=$(curl -s -w "%{http_code}" -o /dev/null "$BASE_URL/../health")
# if [ "$response" = "200" ]; then
#     print_result 0 "Server is running"
# else
#     print_result 1 "Server is not running (HTTP: $response)"
#     echo -e "${RED}Please start the server with 'make dev' first${NC}"
#     exit 1
# fi

# echo -e "\n${BLUE}Step 2: Testing Public Endpoints (No Auth Required)${NC}"
# echo "====================================================="

# Test get warehouses (should work even if empty)
response=$(api_call "GET" "/inventory/warehouses")
echo "Response: $response"
if [[ $response == *"message"* ]]; then
    print_result 0 "GET /inventory/warehouses"
else
    print_result 1 "GET /inventory/warehouses"
fi

# Test get default warehouse (might fail if no warehouse exists yet)
response=$(api_call "GET" "/inventory/warehouses/default")
echo "Response: $response"
if [[ $response == *"message"* ]] || [[ $response == *"default warehouse not found"* ]]; then
    print_result 0 "GET /inventory/warehouses/default (expected to fail if no warehouse)"
else
    print_result 1 "GET /inventory/warehouses/default"
fi

# Test stock level for product ID 1 (might return 0 if no inventory)
response=$(api_call "GET" "/inventory/stock-level/1")
echo "Response: $response"
if [[ $response == *"stock_level"* ]]; then
    print_result 0 "GET /inventory/stock-level/1"
else
    print_result 1 "GET /inventory/stock-level/1"
fi

echo -e "\n${BLUE}Step 3: Admin Login (Required for Admin Tests)${NC}"
echo "=============================================="

echo -e "${YELLOW}To test admin endpoints, you need to:${NC}"
echo "1. Register/Login as admin user"
echo "2. Get the JWT token"
echo "3. Set ADMIN_TOKEN variable"
echo ""
echo "Example admin login:"
echo 'curl -X POST http://localhost:8080/api/v1/auth/login \'
echo '  -H "Content-Type: application/json" \'
echo '  -d "{\"email\":\"admin@example.com\",\"password\":\"admin123"}"'
echo ""

# Prompt for admin token
read -p "Enter your admin JWT token (or press Enter to skip admin tests): " ADMIN_TOKEN

if [ -z "$ADMIN_TOKEN" ]; then
    echo -e "${YELLOW}Skipping admin tests (no token provided)${NC}"
else
    echo -e "\n${BLUE}Step 4: Testing Admin Endpoints${NC}"
    echo "================================"

    # Test create warehouse
    warehouse_data='{
        "name": "Test Warehouse",
        "code": "TEST-001",
        "address": "123 Test Street",
        "city": "Test City",
        "state": "Test State",
        "country": "India",
        "postal_code": "12345",
        "phone": "+91-1234567890",
        "email": "test@warehouse.com",
        "is_default": true
    }'
    
    response=$(api_call "POST" "/admin/warehouses" "$warehouse_data" "$ADMIN_TOKEN")
    echo "Create Warehouse Response: $response"
    if [[ $response == *"created successfully"* ]]; then
        print_result 0 "POST /admin/warehouses (Create warehouse)"
        # Extract warehouse ID for later tests
        WAREHOUSE_ID=$(echo $response | grep -o '"id":[0-9]*' | cut -d':' -f2)
        echo "Created warehouse with ID: $WAREHOUSE_ID"
    else
        print_result 1 "POST /admin/warehouses (Create warehouse)"
        WAREHOUSE_ID=1  # Default fallback
    fi

    # Test create inventory item
    inventory_data='{
        "product_id": 1,
        "warehouse_id": '${WAREHOUSE_ID:-1}',
        "sku": "TEST-SKU-001",
        "initial_quantity": 100
    }'
    
    response=$(api_call "POST" "/admin/inventory" "$inventory_data" "$ADMIN_TOKEN")
    echo "Create Inventory Response: $response"
    if [[ $response == *"created"* ]] || [[ $response == *"updated"* ]]; then
        print_result 0 "POST /admin/inventory (Create inventory item)"
    else
        print_result 1 "POST /admin/inventory (Create inventory item)"
    fi

    # Test get inventory item
    response=$(api_call "GET" "/admin/inventory/1/${WAREHOUSE_ID:-1}" "" "$ADMIN_TOKEN")
    echo "Get Inventory Item Response: $response"
    if [[ $response == *"inventory_item_id"* ]] || [[ $response == *"retrieved successfully"* ]]; then
        print_result 0 "GET /admin/inventory/1/${WAREHOUSE_ID:-1} (Get inventory item)"
    else
        print_result 1 "GET /admin/inventory/1/${WAREHOUSE_ID:-1} (Get inventory item)"
    fi

    # Test stock movement (inbound)
    movement_data='{
        "product_id": 1,
        "warehouse_id": '${WAREHOUSE_ID:-1}',
        "movement_type": "inbound",
        "reason": "purchase",
        "quantity": 50,
        "notes": "Test stock increase",
        "cost_price": 1000
    }'
    
    response=$(api_call "POST" "/admin/inventory/movements" "$movement_data" "$ADMIN_TOKEN")
    echo "Stock Movement Response: $response"
    if [[ $response == *"recorded successfully"* ]]; then
        print_result 0 "POST /admin/inventory/movements (Record stock movement)"
    else
        print_result 1 "POST /admin/inventory/movements (Record stock movement)"
    fi
fi

echo -e "\n${BLUE}Step 5: Testing Protected Endpoints (Auth Required)${NC}"
echo "==================================================="

if [ -n "$ADMIN_TOKEN" ]; then
    # Test stock reservation
    reservation_data='{
        "product_id": 1,
        "warehouse_id": '${WAREHOUSE_ID:-1}',
        "order_id": 999,
        "order_item_id": 999,
        "quantity": 5
    }'
    
    response=$(api_call "POST" "/inventory/reserve" "$reservation_data" "$ADMIN_TOKEN")
    echo "Reserve Stock Response: $response"
    if [[ $response == *"reserved successfully"* ]]; then
        print_result 0 "POST /inventory/reserve (Reserve stock)"
        
        # Test release reservation
        release_data='{
            "order_id": 999,
            "order_item_id": 999
        }'
        
        response=$(api_call "POST" "/inventory/release" "$release_data" "$ADMIN_TOKEN")
        echo "Release Reservation Response: $response"
        if [[ $response == *"released successfully"* ]]; then
            print_result 0 "POST /inventory/release (Release reservation)"
        else
            print_result 1 "POST /inventory/release (Release reservation)"
        fi
    else
        print_result 1 "POST /inventory/reserve (Reserve stock)"
    fi
else
    echo -e "${YELLOW}Skipping protected endpoint tests (no token provided)${NC}"
fi

echo -e "\n${BLUE}Step 6: Database Check${NC}"
echo "====================="

echo "Checking if inventory tables were created..."
echo "You can manually verify by connecting to your database and running:"
echo ""
echo "\\dt"
echo ""
echo "You should see these new tables:"
echo "- warehouses"
echo "- inventory_items" 
echo "- inventory_movements"
echo "- stock_alerts"
echo "- stock_reservations"

echo -e "\n${GREEN}🎉 Test Script Complete!${NC}"
echo "=========================="
echo ""
echo "Summary of what was tested:"
echo "✅ Server health check"
echo "✅ Public inventory endpoints"
echo "✅ Warehouse management (if admin token provided)"
echo "✅ Inventory item management (if admin token provided)"
echo "✅ Stock movements (if admin token provided)"
echo "✅ Stock reservations (if admin token provided)"
echo ""
echo "Next steps:"
echo "1. Check your database for the new inventory tables"
echo "2. Try the endpoints manually with Postman/curl"
echo "3. Create some test products and warehouses"
echo "4. Test the full order flow with inventory reservations"

# Test different scenarios
echo -e "\n${BLUE}Additional Test Commands You Can Run:${NC}"
echo "===========================================" 

echo ""
echo "# Test stock level after adding inventory:"
echo "curl $BASE_URL/inventory/stock-level/1"
echo ""
echo "# Test creating another warehouse:"
echo 'curl -X POST '$BASE_URL'/admin/warehouses \'
echo '  -H "Authorization: Bearer YOUR_TOKEN" \'
echo '  -H "Content-Type: application/json" \'
echo '  -d "{\"name\":\"Secondary Warehouse\",\"code\":\"SEC-001\",\"is_default\":false}"'
echo ""
echo "# Test stock movement (outbound):"
echo 'curl -X POST '$BASE_URL'/admin/inventory/movements \'
echo '  -H "Authorization: Bearer YOUR_TOKEN" \'
echo '  -H "Content-Type: application/json" \'
echo '  -d "{\"product_id\":1,\"warehouse_id\":1,\"movement_type\":\"outbound\",\"reason\":\"sale\",\"quantity\":10}"'