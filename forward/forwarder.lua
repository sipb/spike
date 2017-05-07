module(..., package.seeall)

local raw = require("apps.socket.raw")
local pcap = require("apps.pcap.pcap")
local ipv6 = require("lib.protocol.ipv6")

function run (args)
  local c = config.new()
  config.app(c, "playback", raw.RawSocket, "eth0")
  config.link(c, "playback.tx -> playback.rx") 
  engine.configure(c)
  local link_in = link.new("test_in")
  engine.app_table.playback.input.rx = link_in
  local dg_tx = datagram:new()
  local src = ipv6:pton("2001:4830:2446:b5:5054:ff:fe3f:4c36")
  local dst = ipv6:pton("2001:4830:2446:b5:5054:ff:fe8c:ac0a")
  dg_tx:push(ipv6:new({src = src, dst = dst, next_header=59, hop_limit=1})) 
  link.transmit(link_in, dg_tx:packet())
  engine.app_table.playback:push()
  engine.main({duration=0.01, report={showapps=true,showlinks=true}})
end

	 


