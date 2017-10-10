-- IP fragmentation reassembly
-- The IPv4 protocol allows for packet payloads to be fragmented,
-- i.e. split across multiple IPv4 packets. This payload includes
-- the L4 protocol header, which spike needs in order to decide
-- which backend to redirect the packet to. This requires spike
-- to reassemble fragmented packets.
-- (See the Google Maglev paper, Section 4.3 for details)
--
-- Life of an IP fragment
-- ======================
-- The fragment arrives in
-- Rewriting:handle_fragmentation_and_get_forwarding_params
-- with the following structure:
-- Ethernet (popped) | IPv4 (parsed) | payload
--
-- Spike recognizes that it is a fragment because the fragment offset
-- or MF field is non-zero (RFC791), and redirects the fragment into
-- the spike backend pool, using the 3-tuple in place of the 5-tuple
-- (Maglev paper).
--
-- The fragment then arrives in
-- Rewriting:handle_fragmentation_and_get_forwarding_params
-- of the new spike, now with the following structure:
-- Ethernet (popped) | IPv4 (parsed) | GRE | IPv4 | payload
--
-- Spike recognizes that the fragment has been forwarded once due to
-- the presence of the GRE header and sends the packet to
-- IPFragReassembly:process_datagram, which extracts the payload and
-- other fragment data and sends it to IPFragReassembly:process_frag.
--
-- Here, fragments are grouped according to source and destination
-- IP address, inner protocol ID and identification field (RFC791).
--
-- When a fragment set is complete, IPFragReassembly:process_datagram
-- creates a new datagram with the reassembled payload and a synthetic
-- IP header, with the following structure:
-- IPv4 | TCP/UDP | payload (nothing parsed)
--
-- It is then treated like an unfragmented packet, eventually exiting
-- spike with the following structure:
-- Ethernet | IPv4 | GRE | IPv4 | TCP/UDP | payload
--
-- TODO: Limit the size of fragment table, or make it fixed-size somehow.
--    (as recommended in Maglev paper)

local Datagram = require("lib.protocol.datagram")
local IPV4 = require("lib.protocol.ipv4")
local GRE = require("lib.protocol.gre")
local ffi = require("ffi")
local band = bit.band
local rshift = bit.rshift

require("networking_magic_numbers")

local function create_fragment_set_id(src, dst, next_prot, ident)
   local id = ffi.new("char[12]")
   local id_len = 12
   id[0] = band(next_prot, 0xff)
   id[1] = band(rshift(next_prot, 8), 0xff)
   id[2] = band(ident, 0xff)
   id[3] = band(rshift(ident, 8), 0xff)
   ffi.copy(id + 4, src, 4)
   ffi.copy(id + 8, dst, 4)
   return id, id_len
end

local IPFragReassembly = {}

function IPFragReassembly:new()
   return setmetatable({
      frag_sets = {}
   }, {
      __index = IPFragReassembly
   })
end

-- Adds a fragment to the fragment buffer. If all the fragments of a
-- set have been received, returns the reassembled data in binary
-- format.
-- Arguments:
-- id (string): Identifier for the fragment set containing the IP
--    protocol, source and destination IP addresses, and the fragment
--    identification field.
-- offset (int): IP fragment offset.
-- mf (bool): IP fragment mf (more fragments) flag.
-- payload (binary): IP fragment payload.
-- payload_len (int): Length of payload.
function IPFragReassembly:process_frag(id, offset, mf, payload, payload_len)
   -- fragment offset field is in units of 8-byte blocks
   offset = offset * 8
   local frag_set = self.frag_sets[id]
   if frag_set then
      if frag_set.frags[offset] then
         -- Received same fragment twice
         return
      else
         frag_set.frags[offset] = {
            payload = payload,
            len = payload_len
         }
         frag_set.curr_frags_length = frag_set.curr_frags_length + payload_len
      end
   else
      frag_set = {
         curr_frags_length = payload_len,
         frags = {
            [offset] = {
               payload = payload,
               len = payload_len
            }
         },
         timestamp = os.time()
         -- TODO: Implement packet expiry (with a linked list or queue).
         --    RFC1122 recommends a value between 60 and 120 seconds.
         --    On timeout, an ICMP Time Exceeded must be sent to the
         --    client (also RFC1122).
      }
      self.frag_sets[id] = frag_set
   end
   -- TODO: Ensure packet fragment sizes and total packet data storage
   --    does not get too large
   if not mf then
      frag_set.total_length = offset + payload_len
   end
   if frag_set.total_length and frag_set.curr_frags_length == frag_set.total_length then
      local success = true
      local reassembled = ffi.new('char[?]', frag_set.total_length)
      local curr_offset = 0
      while curr_offset ~= frag_set.total_length do
         local frag = frag_set.frags[curr_offset]
         if not frag or curr_offset + frag.len > frag_set.total_length then
            success = false
            break
         end
         ffi.copy(reassembled + curr_offset, frag.payload, frag.len)
         curr_offset = curr_offset + frag.len
      end
      self.frag_sets[id] = nil
      if success then
         return reassembled, frag_set.total_length
      end
   end
   return
end

-- Processes a datagram containing a redirected IPv4 fragment and
-- adds the fragment to the reassembly table, returning a datagram
-- containing the reassembled data when a fragment set is complete.
-- Expected input datagram structure:
--    Ethernet (popped) | IPv4 (parsed) | GRE | IPv4 | payload
-- Expected output datagram structure:
--    Ethernet (popped/missing) | IPv4 (parsed) | TCP/UDP | payload
-- Arguments:
-- datagram (datagram): Datagram containing IPv4 fragment. Ethernet
--    should be popped and outer IPv4 should be parsed.
-- Returns:
-- datagram (datagram): If a packet set is complete, this will be
--    the datagram containing the reassembled packet. Otherwise,
--    this will be nil.
function IPFragReassembly:process_datagram(datagram)
   -- This function should only be called if the topmost header of
   -- the datagram is a GRE header
   local gre_header = datagram:parse_match(GRE)
   if gre_header == nil then
      return
   end

   -- Checking the recursion control bit seems unnecessary here since
   -- there is no other reason we would encounter a GRE header...?
   if gre_header:protocol() ~= L3_IPV4 then
      return
   end

   -- Snabb thinks that IPv4 cannot be an upper layer for GRE;
   -- so just supply the upper layer class directly.
   -- PR submitted to Snabb, waiting for merge.
   -- TODO: handle IPv4 fragments
   -- local inner_ip_class = gre_header:upper_layer()
   local inner_ip_class = IPV4
   local inner_ip_header = datagram:parse_match(inner_ip_class)
   if inner_ip_header == nil then
      return
   end

   -- Use these four fields to identify fragment sets as per RFC791
   local src = inner_ip_header:src()
   local dst = inner_ip_header:dst()
   local next_prot = inner_ip_header:protocol()
   local ident = inner_ip_header:id()

   local frag_off = inner_ip_header:frag_off()
   local mf = band(inner_ip_header:flags(), IP_MF_FLAG) ~= 0

   local payload, payload_len = datagram:payload()
   local id, id_len = create_fragment_set_id(src, dst, next_prot, ident)
   local id_str = ffi.string(id, id_len)
   local reassembled_pkt, reassembled_pkt_len = self:process_frag(id_str, frag_off, mf, payload, payload_len)
   if not reassembled_pkt then
      return
   end

   local datagram = Datagram:new(nil, nil, {delayed_commit = true})
   datagram:payload(reassembled_pkt, reassembled_pkt_len)

   local ip_header = IPV4:new({
      src = src,
      dst = dst,
      protocol = next_prot,
      ttl = inner_ip_header:ttl()
   })
   local ip_total_length = ip_header:sizeof() + reassembled_pkt_len
   ip_header:total_length(ip_total_length)
   ip_header:checksum()
   datagram:push(ip_header)
   return datagram, ip_header
end

return IPFragReassembly
