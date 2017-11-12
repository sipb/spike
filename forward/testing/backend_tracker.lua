local ffi = require("ffi")
local C = ffi.C
local godefs = require("godefs")

local BackendTracker = {}

function BackendTracker:new()
   return setmetatable({
      backends = {},
   }, {
      __index = BackendTracker
   })
end

function BackendTracker:add_backends(backends)
   for _, b in ipairs(backends) do
      godefs.AddBackend(b.name, b.addr, b.addr_len, b.health_check_type)
      table.insert(self.backends, b)
   end
   C.usleep(1000)
end

function BackendTracker:clear_backends()
   for _, b in ipairs(self.backends) do
      godefs.RemoveBackend(b.name)
   end
   C.usleep(10000)
   self.backends = {}
end

function BackendTracker:set_backends(backends)
   self:clear_backends()
   self:add_backends(backends)
end

function BackendTracker:get_name_by_addr(addr, addr_len)
   for _, addr in ipairs(self.backends) do
      if b.addr_len == addr_len and
         ffi.string(b.addr, b.addr_len) == ffi.string(addr, addr_len) then
         return b.name
      end
   end
end

return BackendTracker
