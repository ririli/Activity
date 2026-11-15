-- KEYS: generation metadata, board, user metadata, processed draw IDs.
-- ARGV[1]: expected generation.
local function guard()
 if not string.find(redis.call('INFO','replication'),'\nrole:master\r\n',1,true) then return false end
 local types={'hash','zset','hash','set'}
 for i,key in ipairs(KEYS) do
  local typ=redis.call('TYPE',key).ok
  if typ~='none' and typ~=types[i] then return false end
 end
 local meta=redis.call('HMGET',KEYS[1],'generation','run_id','users','draws')
 local run=string.match(redis.call('INFO','server'),'run_id:(%w+)')
 return meta[1]==ARGV[1] and meta[2]==run
  and tonumber(meta[3])==redis.call('ZCARD',KEYS[2])
  and tonumber(meta[3])==redis.call('HLEN',KEYS[3])
  and tonumber(meta[4])==redis.call('SCARD',KEYS[4])
end
