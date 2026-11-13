-- ARGV[2]: reverse-ID member. ARGV[3]: packed score from the home shard, or empty.
if not guard() then return {} end
local result={'','0',''}
if ARGV[3] then
 if ARGV[3] ~= '' then
  local higher=redis.call('ZCOUNT',KEYS[2],'(' .. ARGV[3],'+inf')
  local equal=redis.call('ZCOUNT',KEYS[2],ARGV[3],ARGV[3])
  local lo,hi=higher,higher+equal
  while lo<hi do
   local mid=math.floor((lo+hi)/2)
   local member=redis.call('ZRANGE',KEYS[2],mid,mid,'REV')[1]
   if member>ARGV[2] then lo=mid+1 else hi=mid end
  end
  result[2]=tostring(lo)
 end
else
 local mine=redis.call('HGET',KEYS[3],ARGV[2])
 local rank=redis.call('ZREVRANK',KEYS[2],ARGV[2])
 if mine and rank then result={mine,tostring(rank+1),redis.call('ZSCORE',KEYS[2],ARGV[2])}
 elseif mine or rank then return {} end
end
for _,member in ipairs(redis.call('ZRANGE',KEYS[2],0,19,'REV')) do
 local raw=redis.call('HGET',KEYS[3],member)
 if not raw then return {} end
 table.insert(result,raw)
end
return result
