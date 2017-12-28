local B = require("apps.basic.basic_apps")
local P = require("apps.pcap.pcap")
local ffi = require("ffi")
local C = ffi.C
local Rewriting = require("rewriting")
local godefs = require("godefs")
local IPV4 = require("lib.protocol.ipv4")

ffi.cdef[[
void free(void *ptr);
]]

local function runmain()

   godefs.Init()
   local spike_args = godefs.AddBackendsAndGetSpikeConfig("http.yaml")
   C.usleep(3000000) -- wait for backends to come up for demo
   local src_mac   = ffi.string(spike_args.r0); ffi.C.free(spike_args.r0)
   local dst_mac   = ffi.string(spike_args.r1); ffi.C.free(spike_args.r1)
   local ipv4_addr = ffi.string(spike_args.r2); ffi.C.free(spike_args.r2)
   local incap     = ffi.string(spike_args.r3); ffi.C.free(spike_args.r3)
   local outcap    = ffi.string(spike_args.r4); ffi.C.free(spike_args.r4)

   local c = config.new()
   config.app(c, "source", P.PcapReader, incap)
   -- only 1 rewriting app for now, since there's not much benefit to
   -- having more without multithreading
   config.app(c, "rewriting", Rewriting, {src_mac = src_mac,
                                          dst_mac = dst_mac,
                                          ipv4_addr = ipv4_addr})
   config.app(c, "sink", P.PcapWriter, outcap)
   config.link(c, "source.output -> rewriting.input")
   config.link(c, "rewriting.output -> sink.input")

   engine.configure(c)
   engine.main({duration = 1, report = {showlinks = true}})
end

runmain()
