local godefs = require("godefs")

local UnitTests = require("testing/unit_tests")

require("networking_magic_numbers")

local function runmain()
   godefs.Init()

   local network_config = {
      spike_mac = "38:c3:0d:1d:34:df",
      router_mac = "ce:d2:85:61:1e:01",
      backend_vip_addr = "18.0.0.0",
      client_addr = "1.0.0.0",
      spike_internal_addr = "192.168.1.0",
      other_spike_internal_addr = "192.168.1.1",
      backend_vip_ipv6_addr = "0:0:0:0:0:ffff:1200:0",
      client_ipv6_addr = "0:0:0:0:0:ffff:100:0",
      spike_internal_ipv6_addr = "0:0:0:0:0:ffff:c0a8:100",
      other_spike_internal_ipv6_addr = "0:0:0:0:0:ffff:c0a8:101",
      backend_vip_port = 80,
      client_port = 12345,
   }

   local unit_tests = UnitTests:new(network_config)
   unit_tests:run()
end

runmain()
