local ffi = require("ffi")

local C = require("ffi").C
local P = require("apps.pcap.pcap")
local IPV4 = require("lib.protocol.ipv4")
local link = require("core.link")
local pcap = require("testing/pcap")
local packet = require("core.packet")

local Rewriting = require("rewriting")
local godefs = require("godefs")

local packet_synthesis = require("testing/packet_synthesis")
local SpikeTestInstance = require("testing/spike_test_instance")

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
   local test_instance = SpikeTestInstance:new(
      spike_mac, router_mac, spike_addr, router_addr
   )

   local client_addr = "1.0.0.0"

   local test_input_packet = make_ipv4_packet({
      src_mac = router_mac,
      dst_mac = spike_mac,
      src_addr = client_addr,
      dst_addr = spike_addr
   })
   
   local output_packet = test_instance:process_packet(test_input_packet)
   -- local output_packet = test_input_packet
   assert(output_packet)

   local output_file = assert(io.open("test_out.pcap", "w"))
   pcap.write_file_header(output_file)
   pcap.write_record_header(output_file, output_packet.length)
   output_file:write(ffi.string(output_packet.data, output_packet.length))
   output_file:flush()
   packet.free(output_packet)
end

runmain()
