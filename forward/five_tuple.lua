local ffi = require("ffi")
local band = bit.band
local rshift = bit.rshift

local ipv4_five_tuple = ffi.typeof("char[14]")
local ipv4_five_tuple_len = 14
local ipv6_five_tuple = ffi.typeof("char[38]")
local ipv6_five_tuple_len = 38

-- Create a five-tuple
function five_tuple(ip_protocol, src, src_port, dst, dst_port)
   local t, t_len
   if ip_protocol == L3_IPV4 then
      t = ipv4_five_tuple()
      t_len = ipv4_five_tuple_len
      ffi.copy(t + 6, src, 4)
      ffi.copy(t + 10, dst, 4)
   else
      t = ipv6_five_tuple()
      t_len = ipv6_five_tuple_len
      ffi.copy(t + 6, src, 16)
      ffi.copy(t + 22, dst, 16)
   end
   t[0] = band(ip_protocol, 0xff)
   t[1] = band(rshift(ip_protocol, 8), 0xff)
   t[2] = band(src_port, 0xff)
   t[3] = band(rshift(src_port, 8), 0xff)
   t[4] = band(dst_port, 0xff)
   t[5] = band(rshift(dst_port, 8), 0xff)
   return t, t_len
end
