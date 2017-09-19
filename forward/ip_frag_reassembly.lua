local IPFragReassembly = {}

function IPFragReassembly:new()
   return setmetatable({
      frag_pkts = {}
   }, {
      __index = IPFragReassembly
   })
end

function IPFragReassembly:process_packet(five_tuple_str, offset, mf, payload, payload_len)
   local frag_pkt = self.frag_pkts[five_tuple_str]
   if frag_pkt then
      if frag_pkt.frags[offset] then
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
            [offset] = {
               payload = payload,
               len = payload_len
            }
         },
         timestamp = os.time()
      }
      self.frag_pkts[five_tuple_str] = frag_pkt
   end
   if not mf then
      frag_pkt.total_length = offset * 8 + payload_len
   end
   if frag_pkt.total_length and frag_pkt.curr_frags_length == frag_pkt.total_length then
      print("reassembly ready!")
      -- Reassemble packet
   else
      print(frag_pkt.curr_frags_length)
   end
end

return IPFragReassembly
