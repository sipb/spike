local Rewriting = require("rewriting")
local godefs = require("godefs")
local C = require("ffi").C
local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")
local TCP = require("lib.protocol.tcp")

local P = require("apps.pcap.pcap")
local link = require("core.link")
local pcap = require("testing/pcap")
local packet = require("core.packet")

local ffi = require("ffi")

local networking_magic_numbers = require("networking_magic_numbers")

function run_spike_tests()
   godefs.Init()
   godefs.AddBackend("http://cheesy-fries.mit.edu/health",
                     IPV4:pton("1.3.5.7"), 4)
   godefs.AddBackend("http://strawberry-habanero.mit.edu/health",
                     IPV4:pton("2.4.6.8"), 4)
   C.usleep(3000000) -- wait for backends to come up

   local client_mac = "00:11:22:33:44:55"
   local spike_mac, router_mac = "01:23:45:67:89:ab", "01:23:45:67:8a:ab"
   local spike_addr = "1.2.3.4"
   local r = Rewriting:new({
      src_mac = spike_mac,
      dst_mac = router_mac,
      ipv4_addr = spike_addr
   })
   local client_addr = "1.1.1.1"
   local ttl = 30
   local datagram = Datagram:new(nil, nil, {delayed_commit = true})
   local tcp_header = TCP:new({})
   datagram:push(tcp_header)
   local ip_header = IPV4:new({
      src = client_addr,
      dst = spike_addr,
      protocol = L4_TCP,
      ttl = ttl
   })
   ip_header:total_length(ip_header:sizeof())
   datagram:push(ip_header)
   local eth_header = Ethernet:new({
      src = client_mac,
      dst = spike_mac,
      type = L3_IPV4
   })
   datagram:push(eth_header)
   datagram:commit()
   local test_input_packet = datagram:packet()
   local output_packet = r:process_packet(test_input_packet)
   assert(output_packet)

   local output_file = assert(io.open("test_out.pcap", "w"))
   pcap.write_file_header(output_file)
   pcap.write_record_header(output_file, output_packet.length)
   output_file:write(ffi.string(output_packet.data, output_packet.length))
   output_file:flush()
   packet.free(output_packet)

   -- for k,v in pairs(p) do
   --    print('['..k..'] = '..tostring(v))
   -- end
end
