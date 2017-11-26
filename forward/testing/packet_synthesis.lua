local P = require("core.packet")
local GRE = require("lib.protocol.gre")
local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")
local IPV6 = require("lib.protocol.ipv6")
local TCP = require("lib.protocol.tcp")
local ffi = require("ffi")

require("networking_magic_numbers")

local function clone_table(t, changes)
   changes = changes or {}
   local res = {}
   for k, v in pairs(t) do
      res[k] = v
   end
   for k, v in pairs(changes) do
      res[k] = v
   end
   return res
end

local function make_payload(len)
   local buf = ffi.new('char[?]', len)
   for i=0,len-1 do
      buf[i] = (i + 128) % 256
   end
   return ffi.string(buf, len)
end

-- Returns an array of payload fragments.
local function split_payload(payload, payload_len, fragment_len)
   -- add fragment_payload_len-1 to handle case when payload_len is divisible by fragment_payload_len
   -- want to compute (payload_len + fragment_len - 1) // fragment_payload_len but lua5.1 has no integer division
   local dividend = payload_len + fragment_len - 1
   local num_fragments = (dividend - dividend % fragment_len) / fragment_len
   local fragments = {}
   for i=1,num_fragments do
      local curr_fragment_len = fragment_len
      if i == num_fragments then
         curr_fragment_len = payload_len - fragment_len * (num_fragments-1)
      end
      local fragment = string.sub(payload, (i-1) * fragment_len + 1, math.min(i * fragment_len, payload_len))
      fragments[i] = fragment
   end
   return fragments, num_fragments
end

local PacketSynthesisContext = {}

local function parse_mac(mac)
   if mac then return Ethernet:pton(mac) else return nil end
end

local function parse_addr(addr, use_ipv6)
   if addr then
      if use_ipv6 then return IPV6:pton(addr)
      else return IPV4:pton(addr)
      end
   else return nil end
end

-- Arguments (all optional):
-- spike_mac (string) -- MAC address of the spike.
-- router_mac (string) -- MAC address of the router.
-- backend_vip_addr (string) -- Virtual IP address of the backend that
--    the client queries.
-- client_addr (string) -- IP address of the client.
-- spike_internal_addr (string) -- Internal IP address of the spike.
-- other_spike_internal_addr (string) -- Internal IP address of another
--    spike that redirects IP fragments to this spike.
-- backend_vip_port (int) -- Backend port that the client queires.
-- client_port (int) -- Port that the client queries from.
-- ttl (int) -- TTL for IP headers.
-- mtu (int) -- MTU of network.
function PacketSynthesisContext:new(network_config)
   local parsed_network_config = clone_table(network_config, {
      spike_mac = parse_mac(network_config.spike_mac),
      router_mac = parse_mac(network_config.router_mac),
      backend_vip_addr =
         parse_addr(network_config.backend_vip_addr, false),
      client_addr =
         parse_addr(network_config.client_addr, false),
      spike_internal_addr =
         parse_addr(network_config.spike_internal_addr, false),
      other_spike_internal_addr =
         parse_addr(network_config.other_spike_internal_addr, false),
      backend_vip_ipv6_addr =
         parse_addr(network_config.backend_vip_ipv6_addr, true),
      client_ipv6_addr =
         parse_addr(network_config.client_ipv6_addr, true),
      spike_internal_ipv6_addr =
         parse_addr(network_config.spike_internal_ipv6_addr, true),
      other_spike_internal_ipv6_addr =
         parse_addr(network_config.other_spike_internal_ipv6_addr, true)
   })
   return setmetatable({
      network_config = parsed_network_config,
      datagram = nil,
      datagram_len = 0
   }, {
      __index = PacketSynthesisContext
   })
end

function PacketSynthesisContext:new_packet()
   self.datagram = Datagram:new(nil, nil, {delayed_commit = true})
   self.datagram_len = 0
end

-- Arguments:
-- payload (binary) -- Payload data to append.
-- payload_len (int) -- Length of payload.
function PacketSynthesisContext:add_payload(config)
   config = config or {}
   local payload_len = config.payload_len or 100
   local payload = config.payload or make_payload(payload_len)
   self.datagram:payload(payload, payload_len)
   self.datagram_len = self.datagram_len + payload_len
end

-- Arguments:
-- src_addr (binary, default client_addr) -- Source IP address.
-- dst_addr (binary, default backend_vip_addr) -- Destination IP
--    address.
-- inner_prot/l4_prot (int, default L4_TCP) -- Inner protocol.
-- ip_flags (int, default 0) -- Flags.
-- ttl (int, default network_config.ttl or 30) -- TTL.
-- ip_payload_len (int, default datagram_len) -- Length of payload,
--    including L4 header.
function PacketSynthesisContext:make_ip_header(config)
   config = config or {}
   local l3_prot = config.l3_prot or self.network_config.l3_prot or L3_IPV4
   local payload_length = config.ip_payload_len or self.datagram_len
   local ip_header
   if l3_prot == L3_IPV6 then
      ip_header = IPV6:new({
         src = config.src_ipv6_addr or config.src_addr or
            self.network_config.client_ipv6_addr,
         dst = config.dst_ipv6_addr or config.dst_addr or
            self.network_config.backend_vip_ipv6_addr,
         next_header = config.inner_prot or config.l4_prot or L4_TCP,
         hop_limit = config.ttl or self.network_config.ttl or 30
      })
      ip_header:payload_length(payload_length)
   elseif l3_prot == L3_IPV4 then
      ip_header = IPV4:new({
         src = config.src_addr or self.network_config.client_addr,
         dst = config.dst_addr or self.network_config.backend_vip_addr,
         protocol = config.inner_prot or config.l4_prot or L4_TCP,
         flags = config.ip_flags or 0,
         frag_off = config.frag_off or 0,
         ttl = config.ttl or self.network_config.ttl or 30
      })
      local ip_header_size = ip_header:sizeof()
      ip_header:total_length(ip_header_size + payload_length)
      ip_header:checksum()
   else
      assert(false, 'invalid l3_prot')
   end
   return ip_header
end

function PacketSynthesisContext:add_ip_header(config)
   local ip_header = self:make_ip_header(config)
   self.datagram:push(ip_header)
   self.datagram_len = self.datagram_len + ip_header:sizeof()
end

-- TCP checksum requires the IP header, so it is easier to implement
-- adding both at once.
-- Arguments:
-- src_port (int, default client_port) -- Source port.
-- dst_port (int, default backend_vip_port) -- Destination port.
-- tcp_payload (binary, default datagram:data()) -- Payload.
-- tcp_payload_len (int, default length of datagram) -- Length of
--    tcp_payload.
function PacketSynthesisContext:make_tcp_ip_headers(config)
   config = config or {}

   local tcp_header = TCP:new({
      src_port = config.src_port or self.network_config.client_port,
      dst_port = config.dst_port or self.network_config.backend_vip_port
   })
   local tcp_header_size = tcp_header:sizeof()
   -- TCP data offset field is in units of 32-bit words
   tcp_header:offset(tcp_header_size / 4)

   local tcp_payload = config.tcp_payload
   local tcp_payload_len = config.tcp_payload_len
   if not tcp_payload then
      self.datagram:commit()
      tcp_payload, tcp_payload_len = self.datagram:data()
   end
   local ip_header = self:make_ip_header(clone_table(config, {
      ip_payload_len = tcp_payload_len + tcp_header_size
   }))
   tcp_header:checksum(tcp_payload, tcp_payload_len, ip_header)

   return tcp_header, ip_header
end

function PacketSynthesisContext:add_tcp_ip_headers(config)
   local tcp_header, ip_header = self:make_tcp_ip_headers(config)
   self.datagram:push(tcp_header)
   self.datagram:push(ip_header)
   self.datagram_len = self.datagram_len
      + tcp_header:sizeof() + ip_header:sizeof()
end

-- Arguments:
-- inner_prot/l3_prot (int, default L3_IPV4) -- Inner protocol.
function PacketSynthesisContext:add_gre_header(config)
   config = config or {}
   local gre_header = GRE:new({
      protocol = config.inner_prot or config.l3_prot or L3_IPV4
   })
   self.datagram:push(gre_header)
   self.datagram_len = self.datagram_len + gre_header:sizeof()
end

-- Arguments:
-- other_spike_internal_addr (binary,
--    default network_config.other_spike_internal_addr)
--    -- Address of spike sending the packet.
-- spike_internal_addr (binary, default_network_config.spike_internal_addr)
--    -- Address of spike (receiving the packet).
function PacketSynthesisContext:add_spike_to_spike_ip_header(config)
   config = config or {}
   self:add_ip_header(clone_table(config, {
      src_addr = config.other_spike_internal_addr or
         self.network_config.other_spike_internal_addr,
      dst_addr = config.spike_internal_addr or
         self.network_config.spike_internal_addr
   }))
end

-- Arguments:
-- spike_internal_addr (binary, default network_config.spike_internal_addr)
--    -- Address of spike.
-- backend_addr (binary, required) -- Address of backend that the packet
--    is sent to.
function PacketSynthesisContext:add_spike_to_backend_ip_header(config)
   config = config or {}
   self:add_ip_header(clone_table(config, {
      src_addr = config.spike_internal_addr or
         self.network_config.spike_internal_addr,
      src_ipv6_addr = config.spike_internal_ipv6_addr or
         self.network_config.spike_internal_ipv6_addr,
      dst_addr = config.backend_addr
   }))
end

-- Arguments:
-- src_mac (binary, default router_mac) -- Source MAC address.
-- dst_mac (binary, default spike_mac) -- Destination MAC address.
-- inner_prot/l3_prot (int, default L3_IPV4) -- Inner protocol.
function PacketSynthesisContext:add_ethernet_header(config)
   config = config or {}
   local eth_header = Ethernet:new({
      src = config.src_mac or self.network_config.router_mac,
      dst = config.dst_mac or self.network_config.spike_mac,
      type = config.inner_prot or config.l3_prot or L3_IPV4
   })
   self.datagram:push(eth_header)
   self.datagram_len = self.datagram_len + eth_header:sizeof()
end

function PacketSynthesisContext:get_packet()
   self.datagram:commit()
   return self.datagram:packet()
end

-- Returns a packet similar to what Spike would receive in the normal case.
function PacketSynthesisContext:make_in_packet_normal(config)
   config = config or {}
   self:new_packet()
   self:add_payload(config)
   self:add_tcp_ip_headers(config)
   self:add_ethernet_header(config)
   return self:get_packet()
end

-- Returns a packet similar to what Spike would receive in the case when
-- an IPv4 fragment has been redirected to it.
function PacketSynthesisContext:make_in_packet_redirected_ipv4_fragment(
   config
)
   config = config or {}
   self:new_packet()
   self:add_payload(config)
   self:add_ip_header(config)
   self:add_gre_header(config)
   self:add_spike_to_spike_ip_header(clone_table(config,{
      inner_prot = L4_GRE,
      ip_flags = false,
      frag_off = 0
   }))
   self:add_ethernet_header(config)
   return self:get_packet()
end

-- Arguments:
-- mtu (default network_config.mtu) -- MTU of network.
function PacketSynthesisContext:make_ipv4_fragments(config)
   config = config or {}
   self:new_packet()
   self:add_payload(config)

   local tcp_header, ip_header = self:make_tcp_ip_headers(config)
   self.datagram:push(tcp_header)
   self.datagram_len = self.datagram_len + tcp_header:sizeof()

   local p = self:get_packet()
   local ip_payload_len = self.datagram_len
   local ip_payload = ffi.string(p.data, ip_payload_len)

   local mtu = config.mtu or self.network_config.mtu or 100
   local fragment_len = mtu - IP_HEADER_LENGTH
   -- fragment length must be a multiple of 8
   fragment_len = fragment_len - fragment_len % 8
   local fragments, num_fragments = split_payload(
      ip_payload, ip_payload_len, fragment_len
   )
   P.free(p)
   return fragments, num_fragments
end

-- Returns a set of packets similar to what Spike would receive
-- in the case when a set of IPv4 fragments have been redirected to it.
function PacketSynthesisContext:make_in_packets_redirected_ipv4_fragments(
   config
)
   config = config or {}
   local fragments, num_fragments = self:make_ipv4_fragments(config)
   local packets = {}
   local curr_offset = 0
   for i=1,num_fragments do
      local ip_flags = 0
      if i ~= num_fragments then
         ip_flags = IP_MF_FLAG
      end
      -- fragment offset field is units of 8-byte blocks
      local frag_len = string.len(fragments[i])
      local frag_off = curr_offset / 8
      packets[i] = self:make_in_packet_redirected_ipv4_fragment(
         clone_table(config, {
            payload = fragments[i],
            payload_len = frag_len,
            ip_flags = ip_flags,
            frag_off = frag_off
         })
      )
      curr_offset = curr_offset + frag_len
   end
   return packets, num_fragments
end

function PacketSynthesisContext:make_out_packet_normal(config)
   config = config or {}
   self:new_packet()
   self:add_payload(config)
   self:add_tcp_ip_headers(config)
   self:add_gre_header(config)
   local l3_prot = config.outer_l3_prot or config.l3_prot
   self:add_spike_to_backend_ip_header(clone_table(config, {
      l3_prot = l3_prot,
      inner_prot = L4_GRE
   }))
   self:add_ethernet_header(clone_table(config, {
      src_mac = self.network_config.spike_mac,
      dst_mac = self.network_config.router_mac,
      l3_prot = l3_prot
   }))
   return self:get_packet()
end

return PacketSynthesisContext
