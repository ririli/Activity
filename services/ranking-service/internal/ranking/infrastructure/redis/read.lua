if redis.call('EXISTS',KEYS[5])==1 or redis.call('EXISTS',KEYS[4])==0 then return {} end
local count=redis.call('ZCARD',KEYS[1])
if count==0 or count~=redis.call('HLEN',KEYS[2]) then return {} end
local result={tostring(count),'',''}
local mine=redis.call('HGET',KEYS[2],ARGV[1])
local rank=redis.call('ZREVRANK',KEYS[1],ARGV[2])
if mine and rank then result[2]=mine;result[3]=tostring(rank+1)
elseif mine or rank then return {} end
local members=redis.call('ZREVRANGE',KEYS[1],0,19)
-- Reverse-ID members map to decimal-string IDs without Lua BIGINT conversion.
for _,member in ipairs(members) do
 local uid=redis.call('HGET',KEYS[4],member)
 if not uid then return {} end
 local raw=redis.call('HGET',KEYS[2],uid)
 if not raw then return {} end
 table.insert(result,raw)
end
return result
