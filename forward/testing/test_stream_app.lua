local L = require("core.link")

local TestStreamApp = {}

function TestStreamApp:new(opts)
   return setmetatable({
      curr_packet_index = 1,
      packets = opts.packets
   }, {
      __index = TestStreamApp
   })
end

function TestStreamApp:pull()
   if self.packets[self.curr_packet_index] then
      local o = assert(self.output.output, "output port not found")
      L.transmit(o, self.packets[self.curr_packet_index])
      self.curr_packet_index = self.curr_packet_index + 1
   end
end

function TestStreamApp:init(packets)
   self.curr_packet_index = 1
   self.packets = packets
end

return TestStreamApp
