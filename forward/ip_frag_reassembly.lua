local IPFragReassembly = {};

function IPFragReassembly:new()
   return setmetatable({
      frag_pkts = {}
   }, {
      __index = IPFragReassembly
   })
end

function IPFragReassembly:process_packet(five_tuple, offset, mf, payload_len, payload)
   local frag_pkt = frag_pkts[five_tuple]
   if frag_pkt then
      if frag_pkt.frags[offset] then
         return
      else
         frag_pkt.frags[offset] = {
            payload = payload,
            len = payload_len
         }
         frag_pkt.curr_frags_length = frag_pkt.curr_frags_length + payload_len
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
      frag_pkts[five_tuple] = frag_pkt
   end
   if ~mf then
      frag_pkt.total_length = offset + payload_len
   end
   if frag_pkt.total_length and frag_pkt.curr_frags_length == frag_pkt.total_length then
      -- Reassemble packet
   end
end
