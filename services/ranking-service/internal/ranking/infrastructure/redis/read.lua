-- ARGV[2]: reverse-ID member for the requesting user.
if not guard() then return {} end
local result={'',''}
local mine=redis.call('HGET',KEYS[3],ARGV[2])
local rank=redis.call('ZREVRANK',KEYS[2],ARGV[2])
if mine and rank then result={mine,tostring(rank+1)}
elseif mine or rank then return {} end
for _,member in ipairs(redis.call('ZRANGE',KEYS[2],0,19,'REV')) do
 local raw=redis.call('HGET',KEYS[3],member)
 if not raw then return {} end
 table.insert(result,raw)
end
return result
