#!/bin/bash
# Integration test for Claude Agent Manager
# Tests the full flow: create project → PM spawns → agents work → conversation → close
# Captures evidence of each step
set -e

REPORT="/c/tmp/test_report.md"
EVIDENCE="/c/tmp/test_evidence"
mkdir -p "$EVIDENCE"

echo "# Integration Test Report" > "$REPORT"
echo "Date: $(date)" >> "$REPORT"
echo "" >> "$REPORT"

KEY=$(curl -s http://localhost:9222/api/auth/key | docker run --rm -i reach/doc-reader python3 -c "import json,sys; print(json.load(sys.stdin)['apiKey'])")

pass=0
fail=0

check() {
  local name="$1"
  local result="$2"
  if [ "$result" = "PASS" ]; then
    echo "✅ $name" | tee -a "$REPORT"
    pass=$((pass + 1))
  else
    echo "❌ $name: $result" | tee -a "$REPORT"
    fail=$((fail + 1))
  fi
}

echo "## 1. API Health" >> "$REPORT"
HEALTH=$(curl -s http://localhost:9222/api/health)
[ "$HEALTH" = '{"status":"ok"}' ] && check "Backend API healthy" "PASS" || check "Backend API healthy" "$HEALTH"

FRONTEND=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:4173)
[ "$FRONTEND" = "200" ] && check "Frontend serving" "PASS" || check "Frontend serving" "HTTP $FRONTEND"

echo "" >> "$REPORT"
echo "## 2. Roles API" >> "$REPORT"
ROLES=$(curl -s -H "Authorization: Bearer $KEY" http://localhost:9222/api/roles/stats)
ROLE_COUNT=$(echo "$ROLES" | docker run --rm -i reach/doc-reader python3 -c "import json,sys; print(json.load(sys.stdin).get('total_roles',0))" 2>/dev/null)
[ "$ROLE_COUNT" = "162" ] && check "162 roles loaded" "PASS" || check "162 roles loaded" "Got $ROLE_COUNT"

SEARCH=$(curl -s -H "Authorization: Bearer $KEY" "http://localhost:9222/api/roles?q=security" | docker run --rm -i reach/doc-reader python3 -c "import json,sys; print(len(json.load(sys.stdin)))" 2>/dev/null)
[ "$SEARCH" -gt 0 ] 2>/dev/null && check "Role search works" "PASS" || check "Role search works" "Got $SEARCH results"

echo "" >> "$REPORT"
echo "## 3. Project Creation" >> "$REPORT"
# Clean up any test projects first
docker ps --filter "name=cam-agent" -q | xargs -r docker stop 2>/dev/null
docker ps --filter "name=cam-agent" -q -a | xargs -r docker rm 2>/dev/null
rm -rf /c/Users/chris/Projects/integration-test
mkdir -p /c/Users/chris/Projects/integration-test/output
echo "Write a 1-sentence summary of Docker containers." > /c/Users/chris/Projects/integration-test/task.md

RESULT=$(curl -s -X POST http://localhost:9222/api/projects \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
  -d '{"name":"Integration Test","description":"Read /workspace/task.md. Spawn one agent to write it. Have a brief conversation. Close them.","folder_path":"integration-test","max_concurrent":3}')
PID=$(echo "$RESULT" | docker run --rm -i reach/doc-reader python3 -c "import json,sys; print(json.load(sys.stdin).get('id','FAIL'))" 2>/dev/null)
[ "$PID" != "FAIL" ] && [ -n "$PID" ] && check "Project created" "PASS" || check "Project created" "No ID returned"
echo "$RESULT" > "$EVIDENCE/project_created.json"

# Check auto-generated folder
FOLDER=$(echo "$RESULT" | docker run --rm -i reach/doc-reader python3 -c "import json,sys; print(json.load(sys.stdin).get('folder_path',''))" 2>/dev/null)
[ "$FOLDER" = "integration-test" ] && check "Folder path set" "PASS" || check "Folder path set" "Got: $FOLDER"

echo "" >> "$REPORT"
echo "## 4. Project Start + PM Spawn" >> "$REPORT"
curl -s -X POST "http://localhost:9222/api/projects/$PID/start" \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" -d '{}' > /dev/null

sleep 15
CONTAINERS=$(docker ps --filter "name=cam-agent" --format "{{.Names}}" | wc -l)
[ "$CONTAINERS" -ge 1 ] && check "PM container spawned" "PASS" || check "PM container spawned" "$CONTAINERS containers"

# Check PM is restricted
PM_LOG=$(docker logs $(docker ps --filter "name=cam-agent" -q | head -1) 2>&1 | grep "PM mode" | head -1)
echo "$PM_LOG" | grep -q "restricted" && check "PM tool restriction active" "PASS" || check "PM tool restriction active" "No restriction log"

echo "" >> "$REPORT"
echo "## 5. Agent Spawn + Workspace Mount" >> "$REPORT"
sleep 30
CONTAINERS=$(docker ps --filter "name=cam-agent" --format "{{.Names}}" | wc -l)
[ "$CONTAINERS" -ge 2 ] && check "Sub-agent spawned" "PASS" || check "Sub-agent spawned" "$CONTAINERS containers"

# Check mount path
MOUNT=$(docker logs $(docker ps --filter "name=cam-agent" -q | head -1) 2>&1 | head -5)
echo "$MOUNT" > "$EVIDENCE/pm_startup.txt"

# Check workspace mount is correct
SPAWN_LOG=$(docker logs claude_agent_manager-cam-1 2>&1 | grep "Spawning" | tail -1)
echo "$SPAWN_LOG" | grep -q "integration-test" && check "Workspace mount correct" "PASS" || check "Workspace mount correct" "Wrong mount"
echo "$SPAWN_LOG" > "$EVIDENCE/spawn_log.txt"

echo "" >> "$REPORT"
echo "## 6. Agent Activity Streaming" >> "$REPORT"
sleep 30
PM_CONTAINER=$(docker ps --filter "name=cam-agent" -q | head -1)

# Check for tool streaming
TOOL_COUNT=$(docker logs $PM_CONTAINER 2>&1 | grep -c "Streaming tool:" 2>/dev/null || echo 0)
[ "$TOOL_COUNT" -gt 0 ] && check "Tool calls streaming ($TOOL_COUNT)" "PASS" || check "Tool calls streaming" "No streaming"

# Check for thinking streaming
THINK_COUNT=$(docker logs $PM_CONTAINER 2>&1 | grep -c "thinking" 2>/dev/null || echo 0)

# Check for message injection
INJECT_COUNT=$(docker logs $PM_CONTAINER 2>&1 | grep -c "Injecting" 2>/dev/null || echo 0)
[ "$INJECT_COUNT" -gt 0 ] && check "Message injection between turns ($INJECT_COUNT)" "PASS" || check "Message injection" "No injection yet"

echo "" >> "$REPORT"
echo "## 7. PM Conversation + Agent Close" >> "$REPORT"
sleep 30

# Check for relay messages
RELAY_COUNT=$(docker logs $PM_CONTAINER 2>&1 | grep -c "relay_message" 2>/dev/null || echo 0)
[ "$RELAY_COUNT" -gt 0 ] && check "PM used relay_message ($RELAY_COUNT)" "PASS" || check "PM relay_message" "No relay"

# Check for close_agent
CLOSE_COUNT=$(docker logs $PM_CONTAINER 2>&1 | grep -c "close_agent" 2>/dev/null || echo 0)
[ "$CLOSE_COUNT" -gt 0 ] && check "PM closed agents ($CLOSE_COUNT)" "PASS" || check "PM close_agent" "No close"

# Check for BLOCKED (PM tried to use work tool)
BLOCKED_COUNT=$(docker logs $PM_CONTAINER 2>&1 | grep -c "BLOCKED" 2>/dev/null || echo 0)
[ "$BLOCKED_COUNT" -gt 0 ] && check "PM tool enforcement blocked ($BLOCKED_COUNT)" "PASS" || check "PM tool enforcement" "No blocks (may be fine)"

echo "" >> "$REPORT"
echo "## 8. Output Files" >> "$REPORT"
FILE_COUNT=$(ls /c/Users/chris/Projects/integration-test/output/ 2>/dev/null | wc -l)
[ "$FILE_COUNT" -gt 0 ] && check "Output files produced ($FILE_COUNT)" "PASS" || check "Output files" "No files"
ls -lh /c/Users/chris/Projects/integration-test/output/ >> "$REPORT" 2>/dev/null

echo "" >> "$REPORT"
echo "## 9. Pause + Resume" >> "$REPORT"
curl -s -X POST "http://localhost:9222/api/projects/$PID/pause" \
  -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" -d '{}' > /dev/null
sleep 3

# Check agents archived
ACTIVE=$(curl -s -H "Authorization: Bearer $KEY" "http://localhost:9222/api/projects/$PID/agents" | docker run --rm -i reach/doc-reader python3 -c "
import json, sys
agents = json.load(sys.stdin)
active = [a for a in agents if a['status'] not in ('archived','completed')]
print(len(active))
" 2>/dev/null)
[ "$ACTIVE" = "0" ] && check "Pause archived all agents" "PASS" || check "Pause archived agents" "$ACTIVE still active"

echo "" >> "$REPORT"
echo "## Summary" >> "$REPORT"
echo "" >> "$REPORT"
echo "**Passed: $pass**" >> "$REPORT"
echo "**Failed: $fail**" >> "$REPORT"
echo "**Total: $((pass + fail))**" >> "$REPORT"

# Cleanup
docker ps --filter "name=cam-agent" -q | xargs -r docker stop 2>/dev/null
docker ps --filter "name=cam-agent" -q -a | xargs -r docker rm 2>/dev/null
curl -s -X POST "http://localhost:9222/api/projects/$PID/complete" -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" -d '{}' > /dev/null
curl -s -X DELETE "http://localhost:9222/api/projects/$PID" -H "Authorization: Bearer $KEY" > /dev/null
rm -rf /c/Users/chris/Projects/integration-test

echo ""
echo "Report: $REPORT"
echo "Evidence: $EVIDENCE/"
cat "$REPORT"
