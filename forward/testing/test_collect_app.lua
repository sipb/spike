local L = require("core.link")

local TestCollectApp = {}

function TestCollectApp:new(opts)
   return setmetatable({
      num_packets_received = 0,
      packets = {}
   }, {
      __index = TestCollectApp
   })
end

function TestCollectApp:push()
   local i = assert(self.input.input, "input port not found")
   while not L.empty(i) do
      local p = L.receive(i)
      table.insert(self.packets, p)
   end
end

return TestCollectApp
