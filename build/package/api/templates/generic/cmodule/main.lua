local cjson = require "cjson.safe"

local stats = {}

function stats.reset(this)
    this.data = {
        last = os.time(os.date("!*t")),
        send = {},
        recv = {},
    }
end

function stats.update(this, token, direction, bytes)
    local agent = this.data[direction][token] or {}
    this.data[direction][token] = {
        bytes = (agent.bytes or 0) + bytes,
        packets = (agent.packets or 0) + 1,
    }
    this:print()
end

function stats.print(this)
    local now = os.time(os.date("!*t"))
    if this.data.last ~= now and __args["print_stats"][1] == "true" then
        local delta = now - this.data.last
        print("\nstats send:")
        for dst, cnt in pairs(this.data["send"]) do
            print("\tto", dst, cnt.bytes / delta, "Bps", cnt.packets / delta, "Pps")
        end
        print("\nstats recv:")
        for src, cnt in pairs(this.data["recv"]) do
            print("\tfrom", src, cnt.bytes / delta, "Bps", cnt.packets / delta, "Pps")
        end
        this:reset()
    end
end

-- for overriding debug argument
local g_print = print
local print = function(...)
    if __args["debug"][1] == "true" then
        g_print(...);
    end
end

-- for debugging used printing input arguments
for k, _ in pairs(__args) do
    local v = __args[k]
    print(k, v)
    if type(v) == 'table' then
        for i, p in pairs(v) do
            print("\t", i, p)
        end
    end
end

-- for debugging used printing initial agent connected list
local function print_agents()
    print("__agents:")
    for i, a in pairs(__agents.dump()) do
        print("\t", i, type(a))
        print("\t\t", "ID:", a.ID)
        print("\t\t", "IP:", a.IP)
        print("\t\t", "Src:", a.Src)
        print("\t\t", "Dst:", a.Dst)
        print("\t\t", "Type:", a.Type)
        print("\t\t", "Info:", type(a.Info))
        print("\t\t\t", "Info.Os:", type(a.Info.Os))
        print("\t\t\t\t", "Info.Os.Type:", a.Info.Os.Type)
        print("\t\t\t\t", "Info.Os.Name:", a.Info.Os.Name)
        print("\t\t\t\t", "Info.Os.Arch:", a.Info.Os.Arch)
        print("\t\t\t", "Info.Users:", type(a.Info.Users))
        for j, user in ipairs(a.Info.Users) do
            print("\t\t\t\t\t", "Info.Users[" .. tostring(j) .. "].Name:", user.Name)
            print("\t\t\t\t\t", "Info.Users[" .. tostring(j) .. "].Groups:", type(user.Groups))
            for k, group in ipairs(user.Groups) do
                print("\t\t\t\t\t\t", "Info.Users[" .. tostring(j) .. "].Groups[" .. tostring(k) .. "]:", group)
            end
        end
        print()
    end
    print()
end

-- for example handshake so sure that connection was established
local function handshake(src)
    if __args["handshake"][1] ~= "true" then
        print("handshake will be skipped with ", src)
        return
    end
    print("start handshake with ", src)
    local hs_agent_msg = cjson.encode({['type'] = 'hs_agent', ['data'] = "ping"})
    print("\tsent hs agent msg to ", src, ": ", __api.send_data_to(src, hs_agent_msg))
    -- use __api.recv_* functions only without default callback to prevent packet missing
    -- it's a just linear sample and delay on smodule side by __api.await function
    print("\treceived hs server msg from ", src, ": ", __api.recv_data_from(src))
    print("end handshake with ", src)
    print()
end

-- for example variadic data to send ping request
local function random_data(length)
    local res = ""
    for _ = 1, length do
        res = res .. string.char(math.random(97, 122))
    end
    return res
end

-- main function for business logic of module
local function run()
    while __api.is_close() == false and __args["ping_pong"][1] == "true" do
        for dst, src in pairs(__routes.dump()) do
            local ping_agent_req = cjson.encode({["type"] = "ping_req", ["data"] = random_data(10)})
            print("sent ping agent request from ", src, " to ", dst, ": ", __api.send_data_to(dst, ping_agent_req))
            stats:update(dst, "send", #ping_agent_req)
        end
        print()
        __api.await(10000)
    end
    if __args["ping_pong"][1] ~= "true" then
        __api.await(-1)
    end
end

-- set default timeout to wait exit on blocking of recv_* functions
__api.set_recv_timeout(5000) -- 5s

stats:reset()

__api.add_cbs({
    data = function(src, data)
        print('receive data: "' .. data .. '" from: ' .. src)
        local msg = cjson.decode(data)
        stats:update(src, "recv", #data)
        if msg['type'] == 'add_route' then
            local dst = msg['dst']
            local res = __routes.add(dst, __routes.get(src))
            print("add new route to ", dst, ": ", res)

            -- simple handshake to other agent
            local hs_req_msg = cjson.encode({['type'] = 'hs_request', ['data'] = "ping"})
            print("sent hs req msg to ", dst, ": ", __api.send_data_to(dst, hs_req_msg))
            stats:update(src, "send", #hs_req_msg)
        elseif msg['type'] == 'del_route' then
            local dst = msg['dst']
            local res = __routes.del(dst)
            print("del route to ", dst, ": ", res)
        elseif msg['type'] == 'hs_request' then
            local hs_resp_msg = cjson.encode({['type'] = 'hs_response', ['data'] = "pong"})
            print("sent hs resp msg to ", src, ": ", __api.send_data_to(src, hs_resp_msg))
            stats:update(src, "send", #hs_resp_msg)
        elseif msg['type'] == 'hs_response' then
            print("received hs resp msg from ", src)
        elseif msg['type'] == 'ping_req' then
            local server_resp_msg = cjson.encode({['type'] = 'ping_resp', ['data'] = random_data(10)})
            print("sent server response msg to ", src, ": ", __api.send_data_to(src, server_resp_msg))
            stats:update(src, "send", #server_resp_msg)
        elseif msg['type'] == 'ping_resp' then
            print("receive ping_req type message")
        else
            print("receive unknown type message", msg['type'])
        end
        print()
        return true
    end,

    file = function(src, path, name)
        print('receive file: "' .. path .. '" / "' .. name .. '" from: ' .. src)
        return true
    end,

    text = function(src, text, name)
        print('receive text: "' .. text .. '" / "' .. name .. '" from: ' .. src)
        return true
    end,

    msg = function(src, msg, mtype)
        print('receive msg: "' .. msg .. '" / "' .. tostring(mtype) .. '" from: ' .. src)
        return true
    end,

    action = function(src, data, name)
        print('receive action: "' .. data .. '" / "' .. name .. '" from: ' .. src)
        return true
    end,

    control = function(cmtype, data)
        local src = data
        print('receive control msg: "' .. cmtype .. '" from: ' .. src)
        print_agents()
        if cmtype == "agent_connected" then
            -- simple handshake with just connected agent
            handshake(src)
        end
        if cmtype == "agent_disconnected" then
            -- remove all current routes because main connection was closed
            for dst, _ in pairs(__routes.dump()) do
                local res = __routes.del(dst)
                print("del route to ", dst, ": ", res)
            end
        end
        return true
    end,
})

g_print("module " .. tostring(__api.get_name()) .. " was started")

print_agents()
for t, _ in pairs(__agents.dump()) do
    -- simple handshake with already connected agents
    handshake(t)
end

-- run main infinity loop to do module business logic
run()

g_print("module " .. tostring(__api.get_name()) .. " was stopped")

return 'success'
