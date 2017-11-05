local ffi = require("ffi")
require("networking_magic_numbers")

-- Create a five-tuple
function five_tuple(src_ip, dst_ip, src_port, dst_port, protocol_num)
   local src_ip_new = ip_addr[protocol_num]()
   local dst_ip_new = ip_addr[protocol_num]()
   ffi.copy(src_ip_new, src_ip, ip_addr_len[protocol_num])
   ffi.copy(dst_ip_new, dst_ip, ip_addr_len[protocol_num])
   return {src_ip = src_ip_new,
           dst_ip = dst_ip_new,
           src_port = src_port,
           dst_port = dst_port,
           protocol_num = protocol_num}
end
