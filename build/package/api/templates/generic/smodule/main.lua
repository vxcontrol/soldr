require("engine")
local ffi = require'ffi'
local curl = require'libcurl'
local cjson = require "cjson.safe"
local prefix_db = __gid .. "."
local event_data_schema = __config.get_fields_schema()
local current_event_config = __config.get_current_event_config()
local module_info = __config.get_module_info()
local event_engine = CEventEngine(event_data_schema, current_event_config, module_info, prefix_db, true)
local action_engine = CActionEngine({}, true)

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

-- for debugging used printing current config API
local function print_config()
    if __args["print_config"][1] ~= "true" then
        return
    end

    print("__config API:")
    for attr, val in pairs(__config) do
        print("\t", attr, type(val))
        if type(val) == "userdata" and attr:find("set_", 1, true) ~= 1 then
            print("\t\t", cjson.decode(val()))
        elseif type(val):find('^' .. "table") ~= nil then
            for i, k in pairs(val) do
                print("\t\t", i, k)
            end
        else
            print("\t\t", val)
        end
    end
    print()
end

-- for example ganarate test json document by json schema
local function generate_json(schema)
    local result = ""
    local function wrire(raw)
        result = ffi.string(ffi.cast("char*", raw))
        return #result
    end

    curl.easy{
        url = 'https://json.vxcontrol.app/api/v1/schema',
        post = 1,
        httpheader = {
            "Content-Type: application/json",
        },
        postfields = schema;
        writefunction = wrire
      }
      :perform()
    :close()

    return cjson.decode(result)
end

-- for example variadic data to store event daat and to send pong response
local function random_data(length)
    local res = ""
    for _ = 1, length do
        res = res .. string.char(math.random(97, 122))
    end
    return res
end

-- for example push of test events for ones from current config
local function push_events(id)
    local event_config = cjson.decode(__config.get_current_event_config())
    for event_name, _ in pairs(event_config or {}) do
        local info = { name = event_name, data = generate_json(__config.get_fields_schema()) }
        local result, list = event_engine:push_event(info)
        if result then
            for action_id, action_result in ipairs(action_engine:exec(id, list)) do
                print("action " .. tostring(action_id) .. " was executed: ", action_result)
            end
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
        if a.Info ~= nil then
            print("\t\t\t", "Info.Os:", type(a.Info.Os))
            if a.Info.Os ~= nil then
                print("\t\t\t\t", "Info.Os.Type:", a.Info.Os.Type)
                print("\t\t\t\t", "Info.Os.Name:", a.Info.Os.Name)
                print("\t\t\t\t", "Info.Os.Arch:", a.Info.Os.Arch)
            end
            print("\t\t\t", "Info.User:", type(a.Info.User))
            if a.Info.User ~= nil then
                print("\t\t\t\t", "Info.User.Name:", a.Info.User.Name)
                print("\t\t\t\t", "Info.User.Group:", a.Info.User.Group)
            end
        end
        print()
    end
    print()
end

-- for example getting agent ID by dst token on agent connected event
local function get_agent_id(dst)
    for i, a in pairs(__agents.dump()) do
        if i == dst then
            return a.ID
        end
    end
    return ""
end

-- notify all alive agents when new agent was connected that new route is available now
local function add_routes(src)
    local agents = __agents.dump()
    for t, _ in pairs(agents) do
        if t ~= src then
            local route_msg_f = cjson.encode({['type'] = 'add_route', ['dst'] = src})
            print("notify ", t, " about new route to: ", src, ": ", __api.send_data_to(t, route_msg_f))
            local route_msg_r = cjson.encode({['type'] = 'add_route', ['dst'] = t})
            print("notify ", src, " about new route to: ", t, ": ", __api.send_data_to(src, route_msg_r))
        end
    end
end

-- notify all alive agents when an agent was disconnected that old route is unavailable now
local function del_routes(src)
    local agents = __agents.dump()
    for t, _ in pairs(agents) do
        if t ~= src then
            local route_msg_f = cjson.encode({['type'] = 'del_route', ['dst'] = src})
            print("notify ", t, " about new route to: ", src, ": ", __api.send_data_to(t, route_msg_f))
        end
    end
end

-- set default timeout to wait exit on blocking of recv_* functions
__api.set_recv_timeout(5000) -- 5s

__api.add_cbs({
    data = function(src, data)
        print('receive data: "' .. data .. '" from: ' .. src)
        local msg = cjson.decode(data)
        if msg['type'] == 'hs_agent' then
            local hs_server_msg = cjson.encode({['type'] = 'hs_server', ['data'] = "pong"})
            __api.await(100)
            print("sent hs server msg to ", src, ": ", __api.send_data_to(src, hs_server_msg))
            add_routes(src)
        elseif msg['type'] == 'hs_browser' then
            local hs_server_msg = cjson.encode({['type'] = 'hs_server', ['data'] = "pong"})
            print("sent hs server msg to ", src, ": ", __api.send_data_to(src, hs_server_msg))
        elseif msg['type'] == 'ping_req' then
            local server_resp_msg = cjson.encode({['type'] = 'ping_resp', ['data'] = random_data(10)})
            print("sent server response msg to ", src, ": ", __api.send_data_to(src, server_resp_msg))
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
        print('receive control msg: "' .. cmtype .. '" from: ' .. data)
        print_agents()
        if cmtype == "agent_connected" then
            -- notify all alive agents that new agent was connected
            -- routes are sould be fixed on handshake if it is available
            if __args["handshake"][1] ~= "true" then
                add_routes(data)
            end
            push_events(get_agent_id(data))
        end
        if cmtype == "agent_disconnected" then
            -- notify all alive agents that agent was disconnected
            del_routes(data)
        end
        return true
    end,
})

g_print("module " .. tostring(__api.get_name()) .. " was started")

print_config()

for dst, a in pairs(__agents.dump()) do
    add_routes(dst)
    push_events(a.ID)
end

__api.await(-1)

g_print("module " .. tostring(__api.get_name()) .. " was stopped")

return 'success'
