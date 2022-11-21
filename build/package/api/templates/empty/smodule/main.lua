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

-- set default timeout to wait exit on blocking of recv_* functions
__api.set_recv_timeout(5000) -- 5s

__api.add_cbs({
    data = function(src, data)
        print('receive data: "' .. data .. '" from: ' .. src)
        local msg = cjson.decode(data)
        if msg['type'] == 'hs_agent' then
            local hs_server_msg = cjson.encode({['type'] = 'hs_server', ['data'] = "pong"})
            if __args["handshake"][1] == "true" then
                __api.await(100)
                print("sent hs server msg to ", src, ": ", __api.send_data_to(src, hs_server_msg))
            end
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
            print("agent_connected")
            push_events(get_agent_id(src))
        end
        if cmtype == "agent_disconnected" then
            print("agent_disconnected")
        end
        return true
    end,
})

g_print("module " .. tostring(__api.get_name()) .. " was started")

for _, a in pairs(__agents.dump()) do
    push_events(a.ID)
end

__api.await(-1)

g_print("module " .. tostring(__api.get_name()) .. " was stopped")

return 'success'
