local ffi = require("ffi")

local B = require("apps.basic.basic_apps")
local P = require("apps.pcap.pcap")
local Match = require("apps.test.match")
local Datagram = require("lib.protocol.datagram")

local Rewriting = require("rewriting")
local godefs = require("godefs")

local PacketSynthesisContext = require("testing/packet_synthesis")
local TestStreamApp = require("testing/test_stream_app")
local TestCollectApp = require("testing/test_collect_app")
local ExpectedOutputApp = require("testing/expected_output_app")

local UnitTests = {}

function UnitTests:new(network_config)
   return setmetatable({
      network_config = network_config,
      synthesis = PacketSynthesisContext:new(network_config, false)
   }, {
      __index = UnitTests
   })
end

function UnitTests:run()
   local rewriting_config = {
      src_mac = self.network_config.spike_mac,
      dst_mac = self.network_config.router_mac,
   }
   rewriting_config.ipv4_addr = self.network_config.spike_internal_addr

   local c = config.new()
   config.app(c, "stream", TestStreamApp, {})

   config.app(c, "in_tee", B.Tee)
   config.app(c, "in_pcap_writer", P.PcapWriter, "test_in.pcap")
   config.app(c, "spike", Rewriting, rewriting_config)

   config.app(c, "out_tee", B.Tee)
   config.app(c, "expected_output_generator", ExpectedOutputApp, {
      synthesis = self.synthesis
   })
   config.app(c, "out_pcap_writer", P.PcapWriter, "test_out.pcap")
   config.app(c, "out_collect", TestCollectApp)

   config.app(c, "expected_tee", B.Tee)
   config.app(c, "expected_pcap_writer", P.PcapWriter, "test_expected.pcap")
   config.app(c, "expected_collect", TestCollectApp)


   config.link(c, "stream.output -> in_tee.input")
   config.link(c, "in_tee.output_pcap -> in_pcap_writer.input")
   config.link(c, "in_tee.output_spike -> spike.input")

   config.link(c, "spike.output -> out_tee.input")
   config.link(c, "out_tee.output_pcap -> out_pcap_writer.input")
   config.link(c, "out_tee.output_collect -> out_collect.input")
   config.link(c, "out_tee.output_expected_gen ->"..
      "expected_output_generator.input")

   config.link(c, "expected_output_generator.output -> expected_tee.input")
   config.link(c, "expected_tee.output_pcap -> expected_pcap_writer.input")
   config.link(c, "expected_tee.output_collect -> expected_collect.input")

   engine.configure(c)

   -- Note: app_table is undocumented, might break with Snabb updates.
   -- Another way to achieve this is to pass a callback function into
   -- the app constructor during config.app that passes a reference
   -- to the app out, though that would be a bit ugly.
   local stream_app = engine.app_table["stream"]
   local out_collect_app = engine.app_table["out_collect"]
   local expected_collect_app = engine.app_table["expected_collect"]

   stream_app.packets = {
      [1] = self.synthesis:make_in_packet_normal()
   }

   local expectedNumOutputPackets = 1

   local testStartTime = os.clock()
   engine.main({done = function()
      assert(os.clock() - testStartTime < 0.1,
         "Test timed out. Possibly too many packets were dropped.")
      return #out_collect_app.packets == expectedNumOutputPackets and
         #expected_collect_app.packets == expectedNumOutputPackets
   end})

   local out_datagram = Datagram:new(out_collect_app.packets[1])
   local expected_datagram = Datagram:new(expected_collect_app.packets[1])
   local data, data_len = out_datagram:data()
   local cmp_data, cmp_data_len = expected_datagram:data()
   assert(ffi.string(data, data_len) == ffi.string(cmp_data, cmp_data_len),
      "Output packet data does not match expected packet data.")
   print("Test passed!")
end

return UnitTests
