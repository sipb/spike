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

local packet_synthesis = require("testing/packet_synthesis")
local TestStreamApp = require("testing/test_stream_app")

local function runmain()
   godefs.Init()
   godefs.AddBackend("http://cheesy-fries.mit.edu/health",
                     IPV4:pton("1.3.5.7"), 4)
   godefs.AddBackend("http://strawberry-habanero.mit.edu/health",
                     IPV4:pton("2.4.6.8"), 4)
   C.usleep(3000000) -- wait for backends to come up

   local spike_mac = "38:c3:0d:1d:34:df"
   local router_mac = "ce:d2:85:61:1e:01"
   local backend_vip_addr = "18.0.0.0"
   local router_addr = "18.100.100.100"
   local client_addr = "1.0.0.0"
   local spike_internal_addr = "192.168.1.0"
   local other_spike_internal_addr = "192.168.1.1"

   local test_fragmentation = true
   local debug_bypass_spike = false

   local packets
   if test_fragmentation then
      packets = make_fragmented_ipv4_packets({
         src_mac = Ethernet:pton(router_mac),
         dst_mac = Ethernet:pton(spike_mac),
         src_addr = IPV4:pton(client_addr),
         dst_addr = IPV4:pton(backend_vip_addr),
         outer_src_addr = IPV4:pton(other_spike_internal_addr),
         outer_dst_addr = IPV4:pton(spike_internal_addr),
         add_ip_gre_layer = true
      })
   else
      packets = {
         [1] = make_ipv4_packet({
            src_mac = Ethernet:pton(router_mac),
            dst_mac = Ethernet:pton(spike_mac),
            src_addr = IPV4:pton(client_addr),
            dst_addr = IPV4:pton(backend_vip_addr)
         })
      }
   end

   local c = config.new()
   config.app(c, "stream", TestStreamApp, {
      packets = packets
   })
   config.app(c, "spike", Rewriting, {
      src_mac = spike_mac,
      dst_mac = router_mac,
      ipv4_addr = spike_internal_addr
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
