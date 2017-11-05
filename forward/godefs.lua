local ffi = require("ffi")
require("networking_magic_numbers")

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

function M.Init()
   return golib.Init()
end

function M.AddBackend(service, ip, ip_len)
   return golib.AddBackend(GoString(service, #service),
                           GoSlice(ip, ip_len, ip_len))
end

function M.RemoveBackend(service)
   return golib.RemoveBackend(GoString(service, #service))
end

local ip_addr_array = ffi.typeof("unsigned char[16]")
function M.Lookup(src_ip, dst_ip, src_port, dst_port, protocol_num)
   local addr_len = ip_addr_len[protocol_num]
   local output = GoSlice(ip_addr_array(), 16, 16)
   local n = golib.Lookup(GoSlice(src_ip, addr_len, addr_len),
                          GoSlice(dst_ip, addr_len, addr_len),
                          src_port, dst_port,
                          protocol_num,
                          output)
   return output.data, n
end

return M
