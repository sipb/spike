local ffi = require("ffi")

local function read_all(filename)
    local f = io.open(filename)
    local r = f:read("*a")
    f:close()
    return r
end

ffi.cdef(read_all(os.getenv("LOOKUP_H")))
local golib = ffi.load(os.getenv("LOOKUP_SO"))

local GoString = ffi.typeof("GoString")
local GoSlice = ffi.typeof("GoSlice")
local GoInt = ffi.typeof("GoInt")

local M = {}

M.HEALTH_CHECK_NONE = 0
M.HEALTH_CHECK_HTTP = 1

function M.Init()
   return golib.Init()
end

function M.AddBackend(service, ip, ip_len, health_check_type)
   return golib.AddBackend(GoString(service, #service),
                           GoSlice(ip, ip_len, ip_len),
                           health_check_type)
end

function M.AddBackendsFromConfig(config_file)
   return golib.AddBackendsFromConfigVoid(GoString(config_file, #config_file))
end

function M.AddBackendsAndGetSpikeConfig(config_file)
   return golib.AddBackendsAndGetSpikeConfig(GoString(config_file, #config_file))
end

function M.RemoveBackend(service)
   return golib.RemoveBackend(GoString(service, #service))
end

local ip_addr_array = ffi.typeof("char[16]")
function M.Lookup(x, x_len)
   local output = GoSlice(ip_addr_array(), 16, 16)
   local n = golib.Lookup(GoSlice(x, x_len, x_len), output)
   return output.data, n
end

return M
