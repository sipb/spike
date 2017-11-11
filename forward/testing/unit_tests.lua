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
      synthesis = PacketSynthesisContext:new(network_config, false),
      stream_app = nil,
      expected_output_generator_app = nil,
      out_collect_app = nil,
      expected_collect_app = nil
   }, {
      __index = UnitTests
   })
end

-- Arguments:
-- test_name (string) -- Name of test.
-- input_packets (array of packets) -- Packets to stream into Spike.
-- output_generators (array of functions) -- Generators that produce
--    the expected output from Spike. These are functions that take
--    in the backend IP address and produce a packet.
function UnitTests:run_test(test_name, input_packets, output_generators, valid_backend_addrs)
   valid_backend_addrs =
      valid_backend_addrs or self.network_config.backend_addrs

   print("Running test: "..test_name)
   self.stream_app:init(input_packets)

   self.expected_output_generator_app:init(
      output_generators, valid_backend_addrs)
   local expected_num_output_packets = #output_generators

   local test_start_time = os.clock()
   engine.main({done = function()
      assert(os.clock() - test_start_time < 0.1,
         "Test timed out. Possibly too many packets were dropped.")
      return #self.out_collect_app.packets ==
         expected_num_output_packets and
         #self.expected_collect_app.packets ==
         expected_num_output_packets
   end, no_report = true})

   local out_datagram = Datagram:new(self.out_collect_app.packets[1])
   local expected_datagram = Datagram:new(
      self.expected_collect_app.packets[1])
   local data, data_len = out_datagram:data()
   local cmp_data, cmp_data_len = expected_datagram:data()
   if data_len ~= cmp_data_len then
      error("Output packet length incorrect.")
   end
   for i = 0, data_len - 1 do
      if data[i] ~= cmp_data[i] then
         error("Output packet data does not match expected packet data"..
         " at index "..i..".")
      end
   end
   print("Test passed!")
   print()

   self.out_collect_app:clear()
   self.expected_collect_app:clear()
end

function UnitTests:run()
   local rewriting_config = {
      src_mac = self.network_config.spike_mac,
      dst_mac = self.network_config.router_mac,
      ipv4_addr = self.network_config.spike_internal_addr,
      ipv6_addr = self.network_config.spike_internal_ipv6_addr
   }

   local c = config.new()
   config.app(c, "stream", TestStreamApp, {})

   config.app(c, "in_tee", B.Tee)
   config.app(c, "in_pcap_writer", P.PcapWriter, "test_in.pcap")
   config.app(c, "spike", Rewriting, rewriting_config)

   config.app(c, "out_tee", B.Tee)
   config.app(c, "expected_output_generator", ExpectedOutputApp, {
      synthesis = self.synthesis,
      backend_addrs = self.backend_addrs
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
   self.stream_app = engine.app_table["stream"]
   self.expected_output_generator_app =
      engine.app_table["expected_output_generator"]
   self.out_collect_app = engine.app_table["out_collect"]
   self.expected_collect_app = engine.app_table["expected_collect"]

   self:run_test("single_ipv4_packet", {
      [1] = self.synthesis:make_in_packet_normal()
   }, {
      [1] = function(backend_addr)
         return self.synthesis:make_out_packet_normal({
            backend_addr = backend_addr
         })
      end
   })

   self:run_test("ipv4_fragments",
      self.synthesis:make_in_packets_redirected_ipv4_fragments(), {
      [1] = function(backend_addr)
         return self.synthesis:make_out_packet_normal({
            backend_addr = backend_addr
         })
      end
   })
end

return UnitTests
