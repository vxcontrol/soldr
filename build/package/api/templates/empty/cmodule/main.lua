local cjson = require "cjson.safe"

-- for overriding debug argument
local g_print = print
local print = function(...)
    if __args["debug"][1] == "true" then
        g_print(...);
    end
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

-- main function for business logic of module
local function run()
    __api.await(-1)
end

-- set default timeout to wait exit on blocking of recv_* functions
__api.set_recv_timeout(5000) -- 5s

__api.add_cbs({
    data = function(src, data)
        print('receive data: "' .. data .. '" from: ' .. src)
        local msg = cjson.decode(data)
        if msg['type'] == 'hs_request' then
            local hs_resp_msg = cjson.encode({['type'] = 'hs_response', ['data'] = "pong"})
            print("sent hs resp msg to ", src, ": ", __api.send_data_to(src, hs_resp_msg))
        elseif msg['type'] == 'hs_response' then
            print("received hs resp msg from ", src)
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
        if cmtype == "agent_connected" then
            print("agent_connected")
            -- simple handshake with just connected agent
            handshake(src)
        end
        if cmtype == "agent_disconnected" then
            print("agent_disconnected")
        end
        return true
    end,
})

g_print("module " .. tostring(__api.get_name()) .. " was started")

for t, _ in pairs(__agents.dump()) do
    -- simple handshake with already connected agents
    handshake(t)
end

-- run main infinity loop to do module business logic
run()

g_print("module " .. tostring(__api.get_name()) .. " was stopped")

return 'success'
