local P = require("core.packet")
local L = require("core.link")
local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")
local IPV6 = require("lib.protocol.ipv6")
local GRE = require("lib.protocol.gre")
local TCP = require("lib.protocol.tcp")
local bit = require("bit")
local ffi = require("ffi")
local band = bit.band
local rshift = bit.rshift

local godefs = require("godefs")

require("networking_magic_numbers")
require("five_tuple")
local IPFragReassembly = require("ip_frag/reassembly")

_G._NAME = "rewriting"  -- Snabb requires this for some reason

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
   if not opts.ipv4_addr and not opts.ipv6_addr then
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
   end
   if not opts.dst_mac then
      error("need to specify dst_mac")
   end
   return setmetatable(
      {ipv4_addr = ipv4_addr,
       ipv6_addr = ipv6_addr,
       src_mac = opts.src_mac and Ethernet:pton(opts.src_mac),
       dst_mac = Ethernet:pton(opts.dst_mac),
       ttl = opts.ttl or 30,
       ip_frag_reassembly = IPFragReassembly:new()},
      {__index = Rewriting})
end

function Rewriting:push()
   local i = assert(self.input.input, "input port not found")
   local o = assert(self.output.output, "output port not found")
   while not L.empty(i) do
      self:process_packet(i, o)
   end
end

-- Parses a datagram to compute the packet's five tuple and the backend
-- pool to forward to packet to. Due to IP fragmentation, the packet to
-- be forwarded may be different from the packet received.
-- See reassembly.lua for an explanation of IP fragmentation handling.
-- Expected input datagram structure:
-- IPv4 fragment --
--    Ethernet (popped) | IPv4 (parsed) | payload
-- Redirected IPv4 fragment --
--    Ethernet (popped) | IPv4 (parsed) | GRE | IPv4 | payload
-- Full packet --
--    Ethernet (popped) | IPv4/IPv6 (parsed) | TCP/UDP | payload
-- Expected output datagram structure:
--    Ethernet (popped/missing) | IPv4 (parsed) | TCP/UDP | payload
-- Arguments:
-- datagram (datagram) -- Datagram to process. Should have Ethernet
--    header popped and IP header parsed.
-- ip_header (header) -- Parsed IP header of datagram.
-- ip_type (int) -- IP header type, can be L3_IPV4 or L3_IPV6
-- Returns:
-- forward_datagram (bool) -- Indicates whether there is a datagram to
--    forward.
-- t (binary) -- Five tuple of datagram.
-- t_len (int) -- Length of t.
-- backend_pool (int) -- Backend pool to forward the packet to.
-- new_datagram (datagram) -- Datagram to forward. Should have Ethernet
--    header popped or missing, and IP header parsed.
-- new_datagram_len (int) -- Length of new_datagram.
function Rewriting:handle_fragmentation_and_get_forwarding_params(datagram, ip_header, ip_type)
   local ip_src = ip_header:src()
   local ip_dst = ip_header:dst()
   local l4_type, ip_total_length
   if ip_type == L3_IPV4 then
      l4_type = ip_header:protocol()
      ip_total_length = ip_header:total_length()
   else
      l4_type = ip_header:next_header()
      ip_total_length = ip_header:payload_length() + ip_header:sizeof()
   end
   local prot_class = ip_header:upper_layer()

   -- TODO: handle IPv6 fragments
   if ip_type == L3_IPV4 then
      local frag_off = ip_header:frag_off()
      local mf = band(ip_header:flags(), IP_MF_FLAG) ~= 0
      if frag_off ~= 0 or mf then
         -- Packet is an IPv4 fragment; redirect to another spike
         -- Set ports to zero to get three-tuple
         local t3, t3_len = five_tuple(ip_type, ip_src, 0, ip_dst, 0)
         -- TODO: Return spike backend pool
         return true, t3, t3_len, nil, datagram, ip_total_length
      end
   end

   if l4_type == L4_GRE then
      -- Packet is a redirected IPv4 fragment
      local new_datagram, new_ip_header = self.ip_frag_reassembly:process_datagram(datagram)
      if not new_datagram then
         return false
      end
      datagram = new_datagram
      datagram:parse_match(IPV4)
      ip_src = new_ip_header:src()
      ip_dst = new_ip_header:dst()
      ip_total_length = new_ip_header:total_length()
      prot_class = new_ip_header:upper_layer()
   elseif not (l4_type == L4_TCP or l4_type == L4_UDP) then
      return false
   end

   local prot_header = datagram:parse_match(prot_class)
   if prot_header == nil then
      return false
   end
   local src_port = prot_header:src_port()
   local dst_port = prot_header:dst_port()

   local t, t_len = five_tuple(ip_type,
                               ip_src, src_port, ip_dst, dst_port)
   -- TODO: Use IP destination to determine backend pool.

   -- unparse L4
   datagram:unparse(1)
   return true, t, t_len, nil, datagram, ip_total_length
end

function Rewriting:process_packet(i, o)
   local p = L.receive(i)
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
   -- decapsulate from ethernet
   datagram:pop(1)

   local ip_header = datagram:parse_match(ip_class)
   if ip_header == nil then
      P.free(p)
      return
   end
   
   local forward_datagram, t, t_len,
      backend_pool, new_datagram, ip_total_length =
         self:handle_fragmentation_and_get_forwarding_params(
            datagram, ip_header, l3_type
         )
   if not forward_datagram then
      P.free(p)
      return
   end
   if datagram ~= new_datagram then
      P.free(p)
      datagram = new_datagram
   end

   local backend, backend_len = get_backend(t, t_len)
   if backend_len == 0 then
      P.free(p)
      return
   end


   -- unparse L4 and L3
   datagram:unparse(2)

   local gre_header = GRE:new({protocol = l3_type})
   datagram:push(gre_header)


   if backend_len == 4 then -- IPv4
      local outer_ip_header = IPV4:new({src = self.ipv4_addr,
                                        dst = backend,
                                        protocol = L4_GRE,
                                        ttl = self.ttl})
      outer_ip_header:total_length(
            ip_total_length + gre_header:sizeof() + outer_ip_header:sizeof())
      -- need to recompute checksum after changing total_length
      outer_ip_header:checksum()
      datagram:push(outer_ip_header)

      local outer_eth_header = Ethernet:new({src = self.src_mac or eth_dst,
                                             dst = self.dst_mac,
                                             type = L3_IPV4})
      datagram:push(outer_eth_header)
   elseif backend_len == 16 then -- IPv6
      local outer_ip_header = IPV6:new({src= self.ipv6_addr,
                                        dst = backend,
                                        next_header = L4_GRE,
                                        hop_limit = self.ttl})
      outer_ip_header:payload_length(ip_total_length + gre_header:sizeof())
      
      datagram:push(outer_ip_header)

      local outer_eth_header = Ethernet:new({src = self.src_mac or eth_dst,
                                             dst = self.dst_mac,
                                             type = L3_IPV6})
      datagram:push(outer_eth_header)
   else
      error("backend length must be 4 (for IPv4) or 16 (for IPv6)")
   end

   datagram:commit()

   link.transmit(o, datagram:packet())
end

return Rewriting
