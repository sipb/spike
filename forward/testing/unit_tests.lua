local ffi = require("ffi")

local B = require("apps.basic.basic_apps")
local P = require("apps.pcap.pcap")
local IPV4 = require("lib.protocol.ipv4")
local IPV6 = require("lib.protocol.ipv6")
local Datagram = require("lib.protocol.datagram")
local Counter = require("core.counter")

local Rewriting = require("rewriting")
local godefs = require("godefs")

local PacketSynthesisContext = require("testing/packet_synthesis")
local TestStreamApp = require("testing/test_stream_app")
local TestCollectApp = require("testing/test_collect_app")
local ExpectedOutputApp = require("testing/expected_output_app")
local BackendTracker = require("testing/backend_tracker")

local UnitTests = {}

function UnitTests:new(network_config)
   return setmetatable({
      network_config = network_config,
      synthesis = PacketSynthesisContext:new(network_config, false),
      stream_app = nil,
      expected_output_generator_app = nil,
      out_collect_app = nil,
      expected_collect_app = nil,
      backend_tracker = BackendTracker:new()
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
function UnitTests:run_test(test_name, input_packets, output_generators, backends, max_num_breaths)
   if max_num_breaths == nil then
      max_num_breaths = 10
   end

   print("Running test: "..test_name)
   self.stream_app:init(input_packets)

   self.backend_tracker:set_backends(backends)
   self.expected_output_generator_app:init(
      output_generators, backends)
   local expected_num_output_packets = #output_generators

   local err = nil
   local flush_start_num_breaths = -1
   local start_num_breaths = Counter.read(engine.breaths)
   engine.main({done = function()
      local curr_num_breaths = Counter.read(engine.breaths)
      if curr_num_breaths - start_num_breaths > max_num_breaths then
         err = "Too many packets were dropped."
         return true
      end

      if flush_start_num_breaths ~= -1 then
         if #self.out_collect_app.packets ~=
            expected_num_output_packets then
            err = "Too may packets produced."
            return true
         end
         return curr_num_breaths - flush_start_num_breaths > 3
      end

      -- Wait for packets to be flushed to pcap.
      if #self.out_collect_app.packets ==
         expected_num_output_packets and
         #self.expected_collect_app.packets ==
         expected_num_output_packets then
         flush_start_num_breaths = curr_num_breaths
      end

      return false
   end, no_report = true})
   if self.expected_output_generator_app.err then
      err = self.expected_output_generator_app.err
   end

   -- Throw error outside of engine so that pcap files will be written to.
   if err then
      error(err)
   end

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
   local ipv4_backends = {
      {
         name = "backend1",
         addr = IPV4:pton("1.3.5.7"),
         addr_len = 4,
         health_check_type = godefs.HEALTH_CHECK_NONE
      },
      {
         name = "backend2",
         addr = IPV4:pton("2.4.6.8"),
         addr_len = 4,
         health_check_type = godefs.HEALTH_CHECK_NONE
      }
   }
   local ipv6_backends = {
      {
         name = "backend3",
         addr = IPV6:pton("2001:db8:a0b:12f0::1"),
         addr_len = 16,
         health_check_type = godefs.HEALTH_CHECK_NONE
      },
      {
         name = "backend4",
         addr = IPV6:pton("2001:db8:a0b:12f0::2"),
         addr_len = 16,
         health_check_type = godefs.HEALTH_CHECK_NONE
      }
   }
   local backends = {}
   for _, b in ipairs(ipv4_backends) do
      table.insert(backends, b)
   end
   for _, b in ipairs(ipv6_backends) do
      table.insert(backends, b)
   end

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

   self:run_test("single_packet_ipv4", {
      [1] = self.synthesis:make_in_packet_normal()
   }, {
      [1] = function(backend_addr)
         return self.synthesis:make_out_packet_normal({
            backend_addr = backend_addr
         })
      end
   }, ipv4_backends)

   self:run_test("ipv4_fragments",
      self.synthesis:make_in_packets_redirected_ipv4_fragments(), {
      [1] = function(backend_addr)
         return self.synthesis:make_out_packet_normal({
            backend_addr = backend_addr
         })
      end
   }, ipv4_backends)

   self:run_test("single_packet_ipv6", {
      [1] = self.synthesis:make_in_packet_normal()
   }, {
      [1] = function(backend_addr)
         return self.synthesis:make_out_packet_normal({
            outer_l3_prot = L3_IPV6,
            backend_addr = backend_addr
         })
      end
   }, ipv6_backends)
end

return UnitTests
