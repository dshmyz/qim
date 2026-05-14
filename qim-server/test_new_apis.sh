#!/bin/bash

BASE_URL="http://localhost:8080"

echo "=== Step 1: Login ==="
TOKEN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123456"}' | python3 -c "import sys, json; print(json.load(sys.stdin).get('data', {}).get('token', ''))")

if [ -z "$TOKEN" ]; then
  echo "Login failed, trying to create user first..."
  curl -s -X POST "$BASE_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123456","nickname":"Test User"}'
  
  TOKEN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser","password":"test123456"}' | python3 -c "import sys, json; print(json.load(sys.stdin).get('data', {}).get('token', ''))")
fi

echo "Token obtained: ${TOKEN:0:30}..."

# Assign admin role if needed
sqlite3 qim.db "INSERT OR IGNORE INTO user_roles (user_id, role, created_at) VALUES ((SELECT id FROM users WHERE username='testuser'), 'system_admin', datetime('now'));" 2>/dev/null

# Re-login to get admin role
TOKEN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123456"}' | python3 -c "import sys, json; print(json.load(sys.stdin).get('data', {}).get('token', ''))")

echo ""
echo "=========================================="
echo "Test 1: MCP Tools List API"
echo "=========================================="
curl -s -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/admin/mcp/tools" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(f'Code: {d[\"code\"]}')
if d['code'] == 200:
    tools = d.get('data', {}).get('tools', [])
    print(f'Total tools: {len(tools)}')
    for t in tools:
        status = '✓' if t.get('enabled') else '✗'
        print(f'  {status} {t[\"name\"]}: {t[\"description\"][:50]}...')
else:
    print(f'Error: {d.get(\"message\", \"Unknown error\")}')
"

echo ""
echo "=========================================="
echo "Test 2: Disable Tool API"
echo "=========================================="
curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "$BASE_URL/api/v1/admin/mcp/tools/server_monitor" \
  -d '{"enabled": false}' | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(f'Code: {d[\"code\"]}')
print(f'Message: {d.get(\"message\", \"\")}')
if d['code'] == 200:
    data = d.get('data', {})
    print(f'Tool: {data.get(\"tool_name\", \"\")}, Enabled: {data.get(\"enabled\", False)}')
"

echo ""
echo "=========================================="
echo "Test 3: Enable Tool API"
echo "=========================================="
curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "$BASE_URL/api/v1/admin/mcp/tools/server_monitor" \
  -d '{"enabled": true}' | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(f'Code: {d[\"code\"]}')
print(f'Message: {d.get(\"message\", \"\")}')
if d['code'] == 200:
    data = d.get('data', {})
    print(f'Tool: {data.get(\"tool_name\", \"\")}, Enabled: {data.get(\"enabled\", False)}')
"

echo ""
echo "=========================================="
echo "Test 4: Knowledge Graph API"
echo "=========================================="
curl -s -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/api/v1/admin/knowledge-graph?collection=group_1&max_nodes=10" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(f'Code: {d[\"code\"]}')
if d['code'] == 200:
    data = d.get('data', {})
    nodes = data.get('nodes', [])
    edges = data.get('edges', [])
    print(f'Nodes: {len(nodes)}, Edges: {len(edges)}')
    for n in nodes:
        print(f'  - {n[\"label\"]} (type: {n[\"type\"]})')
    for e in edges:
        print(f'    -> {e[\"source\"]} --[{e[\"label\"]}]--> {e[\"target\"]}')
else:
    print(f'Error: {d.get(\"message\", \"Unknown error\")}')
"

echo ""
echo "=========================================="
echo "Test 5: Swagger Documentation"
echo "=========================================="
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/swagger/index.html")
if [ "$STATUS" = "200" ]; then
    echo "Swagger UI: ✓ Available (http://localhost:8080/swagger/index.html)"
else
    echo "Swagger UI: ✗ Not available (HTTP $STATUS)"
fi

echo ""
echo "=========================================="
echo "All tests completed!"
echo "=========================================="
