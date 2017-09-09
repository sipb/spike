local P = require("core.packet")
local L = require("core.link")
local GRE = require("lib.protocol.gre")
local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")
local bit = require("bit")
local ffi = require("ffi")
local band = bit.band
local rshift = bit.rshift

local godefs = require("godefs")

local L3_IPV4 = 0x0800
local L3_IPV6 = 0x86DD
local L4_TCP = 0x06
local L4_UDP = 0x11
local L4_GRE = 0x2f

_G._NAME = "rewriting"  -- Snabb requires this for some reason

local ipv4_five_tuple = ffi.typeof("char[14]")
local ipv4_five_tuple_len = 14
local ipv6_five_tuple = ffi.typeof("char[38]")
local ipv6_five_tuple_len = 38


-- Create a five-tuple
local function five_tuple(ip_protocol, src, src_port, dst, dst_port)
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

-- Return the backend associated with a five-tuple
local function get_backend(five_tuple, five_tuple_len)
   return godefs.Lookup(five_tuple, five_tuple_len)
end


local Rewriting = {}

-- Arguments:
-- ipv4_addr or ipv6_addr (string) -- the IP address of the spike
-- src_mac (string) -- the MAC address to send from; defaults to the
--                     destination MAC of incoming packets
-- dst_mac (string) -- the MAC address to send packets to
-- ttl (int, default 30) -- the TTL to set on outgoing packets
function Rewriting:new(opts)
   local ipv4_addr, ipv6_addr, err
   if opts.ipv4_addr and opts.ipv6_addr then
      error("cannot specify both ipv4 and ipv6")
   end
   if not opts.ipv4_addr and not opts.ipv4_addr then
      error("need to specify either ipv4addr or ipv6addr")
   end
   if opts.ipv4_addr then
      ipv4_addr, err = IPV4:pton(opts.ipv4_addr)
      if not ipv4_addr then
         error(err)
      end
   end
   if opts.ipv6_addr then
      ipv6_addr, err = IPV6:pton(opts.ipv6_addr)
      if not ipv6_addr then
         error(err)
      end
      error("ipv6 not yet implemented")
   end
   if not opts.dst_mac then
      error("need to specify dst_mac")
   end
   return setmetatable(
      {ipv4_addr = ipv4_addr,
       ipv6_addr = ipv6_addr,
       src_mac = opts.src_mac and Ethernet:pton(opts.src_mac),
       dst_mac = Ethernet:pton(opts.dst_mac),
       ttl = opts.ttl or 30},
      {__index = Rewriting})
end

function Rewriting:push()
   local i = assert(self.input.input, "input port not found")
   local o = assert(self.output.output, "output port not found")
   while not L.empty(i) do
      local p = L.receive(i)
      L.transmit(o, self:process_packet(p))
   end
end

function Rewriting:process_packet(p)
   local datagram = Datagram:new(p, nil, {delayed_commit = true})

   local eth_header = datagram:parse_match(Ethernet)
   if eth_header == nil then
      P.free(p)
      return
   end
   local eth_dst = eth_header:dst()
   local l3_type = eth_header:type()
   if not (l3_type == L3_IPV4 or l3_type == L3_IPV6) then
      P.free(p)
      return
   end
   -- Maybe consider moving packet parsing after ethernet into go?
   local ip_class = eth_header:upper_layer()

   local ip_header = datagram:parse_match(ip_class)
   if ip_header == nil then
      P.free(p)
      return
   end
   local ip_src = ip_header:src()
   local ip_dst = ip_header:dst()
   local l4_type
   if l3_type == L3_IPV4 then
      l4_type = ip_header:protocol()
   else
      l4_type = ip_header:next_header()
   end
   if not (l4_type == L4_TCP or l4_type == L4_UDP) then
      P.free(p)
      return
   end
   local prot_class = ip_header:upper_layer()

   local prot_header = datagram:parse_match(prot_class)
   if prot_header == nil then
      P.free(p)
      return
   end
   local src_port = prot_header:src_port()
   local dst_port = prot_header:dst_port()

   local t, t_len = five_tuple(l3_type,
                               ip_src, src_port, ip_dst, dst_port)
   local backend, backend_len = get_backend(t, t_len)
   if backend_len == 0 then
      P.free(p)
      return
   end
   if backend_len == 16 then
      error("ipv6 output not implemented")
   elseif backend_len ~= 4 then
      error("backend length must be 4 (for ipv4) or 16 (for ipv6)")
   end

   -- unparse L4 and L3
   datagram:unparse(2)
   -- decapsulate from ethernet
   datagram:pop(1)

   local _, payload_len = datagram:payload()

   local gre_header = GRE:new({protocol = l3_type})
   datagram:push(gre_header)

   local outer_ip_header = IPV4:new({src = self.ipv4_addr,
                                     dst = backend,
                                     protocol = L4_GRE,
                                     ttl = self.ttl})
   outer_ip_header:total_length(
      payload_len + gre_header:sizeof() + outer_ip_header:sizeof())
   datagram:push(outer_ip_header)

   local outer_eth_header = Ethernet:new({src = self.src_mac or eth_dst,
                                          dst = self.dst_mac,
                                          type = L3_IPV4})
   datagram:push(outer_eth_header)

   datagram:commit()

   return datagram:packet()
end

return Rewriting
