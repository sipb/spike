local ffi = require("ffi")

local B = require("apps.basic.basic_apps")
local P = require("apps.pcap.pcap")
local C = require("ffi").C
local IPV4 = require("lib.protocol.ipv4")
local link = require("core.link")
local packet = require("core.packet")

local Rewriting = require("rewriting")
local godefs = require("godefs")

local packet_synthesis = require("testing/packet_synthesis")
local TestStreamApp = require("testing/test_stream_app")

local function runmain()
   godefs.Init()
   godefs.AddBackend("http://cheesy-fries.mit.edu/health",
                     IPV4:pton("1.3.5.7"), 4)
   godefs.AddBackend("http://strawberry-habanero.mit.edu/health",
                     IPV4:pton("2.4.6.8"), 4)
   C.usleep(3000000) -- wait for backends to come up

   local spike_mac = "00:00:00:00:00:00"
   local router_mac = "00:ff:ff:ff:ff:ff"
   local spike_addr = "18.0.0.0"
   local router_addr = "18.255.255.255"
   local client_addr = "1.0.0.0"

   -- local test_input_packet = make_ipv4_packet({
   --    src_mac = router_mac,
   --    dst_mac = spike_mac,
   --    src_addr = IPV4:pton(client_addr),
   --    dst_addr = IPV4:pton(spike_addr)
   -- })

   local packets = make_fragmented_ipv4_packets({
      src_mac = router_mac,
      dst_mac = spike_mac,
      src_addr = IPV4:pton(client_addr),
      dst_addr = IPV4:pton(spike_addr),
      add_ip_gre_layer = true
   })

   local c = config.new()
   config.app(c, "stream", TestStreamApp, {
      packets = packets
      -- packets = {
      --    [1] = test_input_packet
      -- }
   })
   config.app(c, "spike", Rewriting, {
      src_mac = spike_mac,
      dst_mac = router_mac,
      ipv4_addr = spike_addr
   })
   config.app(c, "pcap_writer", P.PcapWriter, "test_out.pcap")
   -- config.link(c, "stream.output -> pcap_writer.input")
   config.link(c, "stream.output -> spike.input")
   config.link(c, "spike.output -> pcap_writer.input")

   engine.configure(c)
   engine.main({duration = 1, report = {showlinks = true}})
end

runmain()
