--[[
Sliding window rate limit script.

NOTE: This file must be kept in sync with .devcontainer/infra/redis/rate_limit.lua
Both versions should have identical logic to ensure consistency between
the Go binary embedded script and the DevContainer Redis preloaded script.

Arguments:
  ARGV[1] = namespace (string, e.g. "gmail")
  ARGV[2] = bucket (string, e.g. "global" or "user:123")
  ARGV[3] = current Unix timestamp in seconds
  ARGV[4] = number of window definitions (N)
  ARGV[5..] = pairs of <window_size_seconds, limit> repeated N times

The script increments the counter for the current second and rejects the call
if any configured window would exceed its limit. Keys follow the format
ratelimit:{namespace}:{bucket}:{timestamp}.

Return:
  {allowed_flag, window_size_rejected_or_0, limit_for_window, running_total}
]]

local namespace = ARGV[1]
local bucket = ARGV[2]
local now = tonumber(ARGV[3])
local windowCount = tonumber(ARGV[4])

if namespace == nil or bucket == nil or now == nil or windowCount == nil then
  return {0, 0, 0, 0}
end

local largestWindow = 0
local windows = {}
local argIndex = 5

for i = 1, windowCount do
  local windowSize = tonumber(ARGV[argIndex])
  local limit = tonumber(ARGV[argIndex + 1])
  argIndex = argIndex + 2

  if windowSize == nil or limit == nil then
    return {0, 0, 0, 0}
  end

  windows[i] = {size = windowSize, limit = limit}
  if windowSize > largestWindow then
    largestWindow = windowSize
  end
end

local baseKey = "ratelimit:" .. namespace .. ":" .. bucket
local currentKey = baseKey .. ":" .. now
local currentCount = redis.call("INCR", currentKey)
redis.call("EXPIRE", currentKey, largestWindow + 1)

local function sumWindow(windowSize)
  local total = 0
  for offset = 0, windowSize - 1 do
    local key = baseKey .. ":" .. (now - offset)
    local value = redis.call("GET", key)
    if value then
      total = total + tonumber(value)
    end
  end
  return total
end

for _, window in ipairs(windows) do
  local total = sumWindow(window.size)
  if total > window.limit then
    return {0, window.size, window.limit, total}
  end
end

return {1, 0, 0, currentCount}
