local B = require("apps.basic.basic_apps")
local P = require("apps.pcap.pcap")
local G = require("grewrap")

local c = config.new()

ETHERNET = 0x6558
IPV4 = 0x0800
IPV6 = 0x86DD

config.app(c, "source", B.Source)
config.app(c, "gre", G.GRE_wrap, IPV6)
config.app(c, "sink", P.PcapWriter, "out.pcap")
config.link(c, "source.tx -> gre.input")
config.link(c, "gre.output -> sink.input")

engine.configure(c)
engine.main({duration = 1, report = {showlinks = true}})
