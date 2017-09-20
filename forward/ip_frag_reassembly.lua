local ffi = require("ffi")

local IPFragReassembly = {}

function IPFragReassembly:new()
   return setmetatable({
      frag_pkts = {}
   }, {
      __index = IPFragReassembly
   })
end

-- Returns a binary containing the reassembled packet data
function IPFragReassembly:process_packet(five_tuple_str, offset, mf, payload, payload_len)
   local frag_pkt = self.frag_pkts[five_tuple_str]
   if frag_pkt then
      if frag_pkt.frags[offset] then
         -- Received same fragment twice
         return
      else
         frag_pkt.frags[offset] = {
            payload = payload,
            len = payload_len
         }
         frag_pkt.curr_frags_length = frag_pkt.curr_frags_length + payload_len
      end
   else
      frag_pkt = {
         curr_frags_length = payload_len,
         frags = {
            -- currently using offset instead of header id field as key
            [offset] = {
               payload = payload,
               len = payload_len
            }
         },
         timestamp = os.time()
         -- TODO: Implement packet expiry (with a linked list or queue)
      }
      self.frag_pkts[five_tuple_str] = frag_pkt
   end
   if not mf then
      -- fragment offset field is in units of 8-byte blocks
      frag_pkt.total_length = offset * 8 + payload_len
   end
   if frag_pkt.total_length and frag_pkt.curr_frags_length == frag_pkt.total_length then
      local reassembled = ffi.new('char[?]', frag_pkt.total_length)
      -- currently not checking for holes
      -- can do so by starting with the offset = 0 fragment
      -- then adding frag.len to get the offset of the next fragment
      -- instead of just looping through fragments in table order
      local success = true
      for offset, frag in pairs(frag_pkt.frags) do
         if offset + frag.len >= frag_pkt.total_length then
            success = false
            break
         end
         -- fragment offset field is in units of 8-byte blocks
         ffi.copy(reassembled + offset * 8, frag.payload, frag.len)
      end
      self.frag_pkts[five_tuple_str] = nil
      if success then
         return reassembled, frag_pkt.total_length
      end
   end
end

return IPFragReassembly
