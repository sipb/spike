module(..., package.seeall)

local S = require("syscall")

-- connect to rx port, initialize queue object
function RXQueue:new (ifname)
	assert(ifname)






