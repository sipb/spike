local ffi = require("ffi")

local B = require("apps.basic.basic_apps")
local P = require("apps.pcap.pcap")
local C = require("ffi").C
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")
local link = require("core.link")
local packet = require("core.packet")

local Rewriting = require("rewriting")
local godefs = require("godefs")

local PacketSynthesisContext = require("testing/packet_synthesis")
local TestStreamApp = require("testing/test_stream_app")

local function runmain()
   godefs.Init()
   godefs.AddBackend("http://cheesy-fries.mit.edu/health",
                     IPV4:pton("1.3.5.7"), 4)
   godefs.AddBackend("http://strawberry-habanero.mit.edu/health",
                     IPV4:pton("2.4.6.8"), 4)
   C.usleep(3000000) -- wait for backends to come up

   local network_config = {
      spike_mac = "38:c3:0d:1d:34:df",
      router_mac = "ce:d2:85:61:1e:01",
      backend_vip_addr = "18.0.0.0",
      client_addr = "1.0.0.0",
      spike_internal_addr = "192.168.1.0",
      other_spike_internal_addr = "192.168.1.1",
      backend_vip_port = 80,
      client_port = 12345
   }

   local synthesis = PacketSynthesisContext:new(network_config)

   local test_fragmentation = true
   local debug_bypass_spike = false

   local packets
   if test_fragmentation then
      packets = synthesis:make_redirected_ipv4_fragment_packets()
   else
      packets = {
         [1] = synthesis:make_ipv4_packet()
      }
   end

   local c = config.new()
   config.app(c, "stream", TestStreamApp, {
      packets = packets
   })
   config.app(c, "spike", Rewriting, {
      src_mac = network_config.spike_mac,
      dst_mac = network_config.router_mac,
      ipv4_addr = network_config.spike_internal_addr
   })
   config.app(c, "pcap_writer", P.PcapWriter, "test_out.pcap")
   if debug_bypass_spike then
      config.link(c, "stream.output -> pcap_writer.input")
   else
      config.link(c, "stream.output -> spike.input")
      config.link(c, "spike.output -> pcap_writer.input")
   end

   engine.configure(c)
   engine.main({duration = 1, report = {showlinks = true}})
end

runmain()
