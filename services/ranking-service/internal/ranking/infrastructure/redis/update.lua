-- IDs remain strings. Only bounded exact integers are converted to Lua numbers.
-- KEYS: board, users, draw fingerprints, fixed bounds, disabled flag.
-- ARGV: user, reverse-ID member, draw ID (empty for init), fingerprint,
--       delta, business second, minimum second, end second.
local allowed={'zset','hash','hash','hash','string'}
for i,key in ipairs(KEYS) do
 local typ=redis.call('TYPE',key).ok
 if typ~='none' and typ~=allowed[i] then return redis.error_reply('ranking key type mismatch') end
end
if redis.call('EXISTS',KEYS[5])==1 then return -2 end
local minSec,endSec=tonumber(ARGV[7]),tonumber(ARGV[8])
local at,delta=tonumber(ARGV[6]),tonumber(ARGV[5])
if at<minSec or at>endSec then return -3 end
local oldMin=redis.call('HGET',KEYS[4],'min')
local oldEnd=redis.call('HGET',KEYS[4],'end')
if oldMin and (oldMin~=ARGV[7] or oldEnd~=ARGV[8]) then
 redis.call('SET',KEYS[5],'bounds-changed');return -2
end
if ARGV[3]~='' then
 local seen=redis.call('HGET',KEYS[3],ARGV[3])
 if seen then
  if seen~=ARGV[4] then return -1 end
  return 0
 end
end
local raw=redis.call('HGET',KEYS[2],ARGV[1])
local score,reached=0,at
if raw then
 local old=cjson.decode(raw)
 score=tonumber(old.score);reached=tonumber(old.reached)
 if delta==0 then
  if score>0 then return 0 end
  reached=math.min(reached,at)
 elseif score>0 then reached=math.max(reached,at)
 else reached=at end
end
local maxExact=9007199254740991
local base=endSec-minSec+1
-- Check before adding/multiplying; failure never records the draw as processed.
if delta>maxExact-score then redis.call('SET',KEYS[5],'precision');return -2 end
score=score+delta
local offset=endSec-reached
if base>maxExact or score>math.floor((maxExact-offset)/base) then
 redis.call('SET',KEYS[5],'precision');return -2
end
local packed=score*base+offset
if packed>maxExact then redis.call('SET',KEYS[5],'precision');return -2 end
local metadata=cjson.encode({user_id=ARGV[1],score=string.format('%.0f',score),reached=string.format('%.0f',reached)})
redis.call('HSET',KEYS[4],'min',ARGV[7],'end',ARGV[8],ARGV[2],ARGV[1])
redis.call('ZADD',KEYS[1],string.format('%.0f',packed),ARGV[2])
redis.call('HSET',KEYS[2],ARGV[1],metadata)
if ARGV[3]~='' then redis.call('HSET',KEYS[3],ARGV[3],ARGV[4]) end
return 1

