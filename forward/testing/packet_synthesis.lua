local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")
local TCP = require("lib.protocol.tcp")
local ffi = require("ffi")

local networking_magic_numbers = require("networking_magic_numbers")

local function make_payload(len)
   local buf = ffi.new('char[?]', len)
   for i=0,len-1 do
      buf[i] = i % 256
   end
   return ffi.string(buf, len)
end

function make_ipv4_packet(config)
   local datagram = Datagram:new(nil, nil, {delayed_commit = true})

   local payload_length = config.payload_length or 100
   local payload = config.payload or
      make_payload(payload_length)
   datagram:payload(payload, payload_length)

   local tcp_header = TCP:new({
   })
   tcp_header:offset(tcp_header:sizeof())
   datagram:push(tcp_header)

   local ip_header = IPV4:new({
      src = config.src_addr,
      dst = config.dst_addr,
      protocol = L4_TCP
   })
   ip_header:total_length(ip_header:sizeof() + tcp_header:sizeof() + payload_length)
   datagram:push(ip_header)

   local eth_header = Ethernet:new({
      src = config.src_mac,
      dst = config.dst_addr,
      type = L3_IPV4
   })
   datagram:push(eth_header)

   datagram:commit()
   return datagram:packet()
end
