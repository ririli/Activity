-- ARGV: generation, user ID, reverse-ID member, draw ID (empty for init),
--       delta, business second.
if not guard() then return -1 end
local minSec,endSec=unpack(redis.call('HMGET',KEYS[1],'min','end'))
minSec,endSec=tonumber(minSec),tonumber(endSec)
local at,delta=tonumber(ARGV[6]),tonumber(ARGV[5])
local maxExact=9007199254740991
if not minSec or not endSec or not at or not delta or at<minSec or at>endSec
 or delta<0 or delta>maxExact then return -1 end

local draw,member=ARGV[4],ARGV[3]
if draw~='' and redis.call('SISMEMBER',KEYS[4],draw)==1 then return 0 end
local raw=redis.call('HGET',KEYS[3],member)
local score,reached=0,at
if raw then
 local old=cjson.decode(raw)
 score=tonumber(old.score)
 local prior=tonumber(old.reached)
 if delta==0 then
  if score>0 then return 0 end
  reached=math.min(prior,at)
 elseif score>0 then
  reached=math.max(prior,at)
 end
end
if delta>maxExact-score then return -1 end
score=score+delta
local base=endSec-minSec+1
local offset=endSec-reached
if base>maxExact or score>math.floor((maxExact-offset)/base) then return -1 end
local packed=score*base+offset
if packed>maxExact then return -1 end
local metadata=cjson.encode({user_id=ARGV[2],score=string.format('%.0f',score),reached=string.format('%.0f',reached)})

if draw~='' then
 redis.call('SADD',KEYS[4],draw)
 redis.call('HINCRBY',KEYS[1],'draws',1)
end
redis.call('ZADD',KEYS[2],string.format('%.0f',packed),member)
redis.call('HSET',KEYS[3],member,metadata)
if not raw then redis.call('HINCRBY',KEYS[1],'users',1) end
return 1
