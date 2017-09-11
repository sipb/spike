local B = require("apps.basic.basic_apps")
local P = require("apps.pcap.pcap")
local C = require("ffi").C
local Rewriting = require("rewriting")
local godefs = require("godefs")
local IPV4 = require("lib.protocol.ipv4")
local cmdline = require("cmdline")
local spike_test = require("testing/spike_test")

local function runmain()
   local params = getparam(main.parameters)
   if params["t"] then
      run_spike_tests()
      return nil
   end
   if #params ~= 5 then
      print("Usage: spike src_mac dst_mac ipv4_addr in.pcap out.pcap")
      os.exit(1)
   end

   godefs.Init()
   godefs.AddBackend("http://cheesy-fries.mit.edu/health",
                     IPV4:pton("1.3.5.7"), 4)
   godefs.AddBackend("http://strawberry-habanero.mit.edu/health",
                     IPV4:pton("2.4.6.8"), 4)
   C.usleep(3000000) -- wait for backends to come up for demo
   local src_mac, dst_mac, ipv4_addr, incap, outcap = unpack(main.parameters)

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
