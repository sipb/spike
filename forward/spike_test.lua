local Rewriting = require("rewriting")
local Datagram = require("lib.protocol.datagram")
local Ethernet = require("lib.protocol.ethernet")
local IPV4 = require("lib.protocol.ipv4")

local L3_IPV4 = 0x0800
local L3_IPV6 = 0x86DD
local L4_TCP = 0x06
local L4_UDP = 0x11
local L4_GRE = 0x2f

function run_spike_tests()
   local src_mac, dst_mac = "01:23:45:67:89:ab", "01:23:45:67:8a:ab"
   local spike_addr = "1.2.3.4"
   local r = Rewriting:new({
      src_mac = src_mac,
      dst_mac = dst_mac,
      ipv4_addr = spike_addr
   })
   local client_addr = "1.1.1.1"
   local ttl = 30
   local datagram = Datagram:new(nil, nil, {delayed_commit = true})
   local ip_header = IPV4:new({
      src = client_addr,
      dst = spike_addr,
      protocol = L4_TCP,
      ttl = ttl
   })
   print(datagram:commit())
end
