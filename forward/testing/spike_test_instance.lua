local Rewriting = require("rewriting")

local SpikeTestInstance = {}

function SpikeTestInstance:new(
      spike_mac,
      router_mac,
      spike_addr,
      router_addr
   )
   return setmetatable({
      spike_mac = spike_mac,
      router_mac = router_mac,
      spike_addr = spike_addr,
      router_addr = router_addr,
      rewriter = Rewriting:new({
         src_mac = spike_mac,
         dst_mac = router_mac,
         ipv4_addr = spike_addr
      })
   }, {
         __index = SpikeTestInstance
   })
end

function SpikeTestInstance:process_packet(p)
   return self.rewriter:process_packet(p)
end

return SpikeTestInstance
