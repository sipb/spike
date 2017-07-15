local P = require("core.packet")
local L = require("core.link")
local GRE = require("lib.protocol.gre")
local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")

local L3_IPV4 = 0x0800
local L3_IPV6 = 0x86DD
local L4_TCP = 0x06
local L4_UDP = 0x11
local L4_GRE = 0x2f

_G._NAME = "rewriting"  -- Snabb requires this for some reason


-- Compute the five-tuple hash of a packet given the five-tuple
local function hash_five_tuple(ip_protocol, src, src_port, dst, dst_port)
   -- TODO implement this for real in go
   --      (currently implements the random oracle model)
   return 4
end


-- Return the backend associated with a hash value
local function get_backend(five_hash)
   -- TODO
   return "\066\066\066\066"
end


local Rewriting = {}

function Rewriting:new()
   return setmetatable({}, {__index = Rewriting})
end

function Rewriting:push()
   local i = assert(self.input.input, "input port not found")
   local o = assert(self.output.output, "output port not found")
   while not L.empty(i) do
      self:process_packet(i, o)
   end
end

function Rewriting:process_packet(i, o)
   -- SECURITY TODO make sure malformed packets can't break parsing
   local p = L.receive(i)
   local datagram = Datagram:new(p, nil, {delayed_commit = true})

   local eth_header = datagram:parse_match(Ethernet)
   if eth_header == nil then
      P.free(p)
      return
   end
   local l3_type = eth_header:type()
   if not (l3_type == L3_IPV4 or l3_type == L3_IPV6) then
      P.free(p)
      return
   end
   -- TODO consider moving packet parsing after ethernet into go
   local ip_class = eth_header:upper_layer()

   -- TODO should we check the length of the packet and discard any
   --      remainder?
   local ip_header = datagram:parse_match(ip_class)
   if ip_header == nil then
      P.free(p)
      return
   end
   local ip_src = ip_header:src()
   local ip_dst = ip_header:dst()
   -- TODO should we decrement the inner TTL too?
   --      (and recalculate the inner checksum)
   local ip_ttl, l4_type
   if l3_type == L3_IPV4 then
      ip_ttl = ip_header:ttl() - 1
      l4_type = ip_header:protocol()
   else
      ip_ttl = ip_header:hop_limit() - 1
      l4_type = ip_header:next_header()
   end
   if ip_ttl <= 0 then
      -- TODO should we send a time-exceeded?
      P.free(p)
      return
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

   local five_hash = hash_five_tuple(l3_type,
                                     ip_src, src_port, ip_dst, dst_port)
   local backend = get_backend(five_hash)

   -- unparse L4 and L3
   datagram:unparse(2)
   -- decapsulate from ethernet
   datagram:pop(1)

   local _, payload_len = datagram:payload()

   local gre_header = GRE:new({protocol = l3_type})
   datagram:push(gre_header)

   -- TODO figure out source IP
   -- TODO should we inherit ecn, id, fragmentation etc from the
   --      underlying packet?
   -- TODO should we emit IPV4 or IPV6?
   local outer_ip_header = IPV4:new({dst = backend,
                                     protocol = L4_GRE,
                                     ttl = ip_ttl})
   outer_ip_header:total_length(
      payload_len + gre_header:sizeof() + outer_ip_header:sizeof())
   datagram:push(outer_ip_header)

   -- TODO figure out destination and source MAC addresses
   local outer_eth_header = Ethernet:new({type = L3_IPV4})
   datagram:push(outer_eth_header)

   datagram:commit()

   link.transmit(o, datagram:packet())
end

return Rewriting
