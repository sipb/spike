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

M.HEALTH_CHECK_NONE = 0
M.HEALTH_CHECK_HTTP = 1

function M.Init()
   return golib.Init()
end

function M.LoadConfig(config_file)
   local cfg = golib.LoadConfig(GoString(config_file, #config_file))
   return {
      src_mac = cfg.r0,
      dst_mac = cfg.r1,
      src_ip  = cfg.r2,
   }
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
