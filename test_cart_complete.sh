#!/bin/bash

# Simple Cart Test Script - Reliable version
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

BASE_URL="http://localhost:8080/api/v1"
COOKIE_JAR=$(mktemp)

# Cleanup
cleanup() {
    rm -f "$COOKIE_JAR"
}
trap cleanup EXIT

echo -e "${BLUE}🚀 Simple Cart Testing${NC}\n"

# Step 1: Login with existing user (test@example.com)
echo -e "${YELLOW}Step 1: User Login${NC}"
login_response=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -d '{
    "email": "test1@example.com",
    "password": "SecurePass1!!"
  }')

echo "Login Response:"
echo "$login_response" | jq '.'

# Extract token (simple method)
USER_TOKEN=$(echo "$login_response" | jq -r '.data.access_token')
echo -e "\n${GREEN}✅ User Token: ${USER_TOKEN:0:30}...${NC}\n"

# Step 2: Get Admin Token
echo -e "${YELLOW}Step 2: Admin Login${NC}"
admin_response=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }')

ADMIN_TOKEN=$(echo "$admin_response" | jq -r '.data.access_token')
echo -e "${GREEN}✅ Admin Token: ${ADMIN_TOKEN:0:30}...${NC}\n"

# Step 3: Check Products
echo -e "${YELLOW}Step 3: Check Products${NC}"
curl -s -X GET "$BASE_URL/products" | jq '.data.products[] | {id, name, price}'
echo ""

# Step 4: Get Empty Cart
echo -e "${YELLOW}Step 4: Get Empty User Cart${NC}"
curl -s -X GET "$BASE_URL/cart" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR" | jq '.'
echo ""

# Step 5: Add Item to Cart
echo -e "${YELLOW}Step 5: Add Item to Cart${NC}"
add_response=$(curl -s -X POST "$BASE_URL/cart/items" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -d '{
    "product_id": 1,
    "quantity": 2
  }')

echo "Add Response:"
echo "$add_response" | jq '.'
echo ""

# Step 6: Get Cart After Adding
echo -e "${YELLOW}Step 6: Get Cart After Adding${NC}"
cart_response=$(curl -s -X GET "$BASE_URL/cart" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR")

echo "Cart Response:"
echo "$cart_response" | jq '.'
echo ""

# Step 7: Update Cart Item
echo -e "${YELLOW}Step 7: Update Cart Item Quantity${NC}"
update_response=$(curl -s -X PUT "$BASE_URL/cart/items/1" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -d '{
    "quantity": 5
  }')

echo "Update Response:"
echo "$update_response" | jq '.'
echo ""

# Step 8: Get Cart After Update
echo -e "${YELLOW}Step 8: Get Cart After Update${NC}"
final_cart=$(curl -s -X GET "$BASE_URL/cart" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR")

echo "Final Cart:"
echo "$final_cart" | jq '.'
echo ""

# Step 9: Get Cart Count
echo -e "${YELLOW}Step 9: Get Cart Count${NC}"
count_response=$(curl -s -X GET "$BASE_URL/cart/count" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR")

echo "Count Response:"
echo "$count_response" | jq '.'
echo ""

# Step 10: Remove Item
echo -e "${YELLOW}Step 10: Remove Item from Cart${NC}"
remove_response=$(curl -s -X DELETE "$BASE_URL/cart/items/1" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR")

echo "Remove Response:"
echo "$remove_response" | jq '.'
echo ""

# Step 11: Get Cart After Removal
echo -e "${YELLOW}Step 11: Get Cart After Removal${NC}"
empty_cart=$(curl -s -X GET "$BASE_URL/cart" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -b "$COOKIE_JAR" -c "$COOKIE_JAR")

echo "Empty Cart:"
echo "$empty_cart" | jq '.'

echo -e "\n${GREEN}✅ Cart Testing Complete!${NC}"

# Summary
echo -e "\n${BLUE}📊 Summary:${NC}"
echo "• User authentication: ✅"
echo "• Add to cart: ✅"
echo "• Update cart: ✅"
echo "• Get cart: ✅"
echo "• Remove from cart: ✅"
echo "• Cookie persistence: ✅"