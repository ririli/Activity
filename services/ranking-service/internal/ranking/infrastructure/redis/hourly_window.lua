-- KEYS: window metadata, window ZSET. ARGV: operation, lane count, arguments.
local op, lanes = ARGV[1], tonumber(ARGV[2])
local meta, board = KEYS[1], KEYS[2]
local run = string.match(redis.call('INFO', 'server'), 'run_id:(%w+)')
local state = redis.call('HMGET', meta, 'run_id', 'users', 'state')
if state[3] == 'retired' then return {'retired'} end

if op == 'retire' then
  redis.call('UNLINK', board, meta)
  redis.call('HSET', meta, 'state', 'retired')
  return {'retired'}
end

if state[1] ~= run or tonumber(state[2]) ~= redis.call('ZCARD', board) then
  redis.call('UNLINK', board, meta)
  redis.call('HSET', meta, 'run_id', run, 'users', 0, 'state', 'open')
  state[3] = 'open'
end

local function cursor(lane)
  return tonumber(redis.call('HGET', meta, 'seq:' .. lane) or '0')
end

if op == 'state' then
  local result = {state[3]}
  for lane = 0, lanes - 1 do result[#result+1] = string.format('%.0f', cursor(lane)) end
  return result
end

if op == 'freeze' then
  for lane = 0, lanes - 1 do
    if cursor(lane) ~= tonumber(ARGV[3+lane]) then return {'incomplete'} end
  end
  redis.call('HSET', meta, 'state', 'frozen')
  local result = {'frozen'}
  for _, value in ipairs(redis.call('ZRANGE', board, 0, tonumber(ARGV[3+lanes])-1, 'REV', 'WITHSCORES')) do
    result[#result+1] = value
  end
  return result
end

local lane = ARGV[3]
local last = cursor(lane)
for i = 4, #ARGV, 3 do
  local seq = tonumber(ARGV[i])
  if seq > last then
    if seq ~= last + 1 or state[3] == 'frozen' then return {'gap', string.format('%.0f', last)} end
    local member, score = ARGV[i+1], ARGV[i+2]
    local added = redis.call('ZADD', board, 'GT', score, member)
    if added == 1 then redis.call('HINCRBY', meta, 'users', 1) end
    redis.call('HSET', meta, 'seq:' .. lane, ARGV[i])
    last = seq
  end
end
return {'applied', string.format('%.0f', last)}
