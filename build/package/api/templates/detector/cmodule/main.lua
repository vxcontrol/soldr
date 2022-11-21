require("engine")
local ffi = require("ffi")
local curl = require("libcurl")
local cjson = require("cjson.safe")

-- variables to initialize event and action engines
local prefix_db = __gid .. "."
local fields_schema = __config.get_fields_schema()
local current_event_config = __config.get_current_event_config()
local module_info = __config.get_module_info()

-- event and action engines initialization
local event_engine = CEventEngine(fields_schema, current_event_config, module_info, prefix_db, true)
local action_engine = CActionEngine({}, true)

-- for overriding debug argument
local g_print = print
local print = function(...)
    if __args["debug"][1] == "true" then
        g_print(__config.ctx.name .. " : ", ...);
    end
end

-- for example ganarate test json document by json schema
local function generate_json(schema)
    local result = ""
    local function wrire(raw)
        result = ffi.string(ffi.cast("char*", raw))
        return #result
    end

    curl.easy({
        ["url"] = "https://json.vxcontrol.app/api/v1/schema",
        ["post"] = 1,
        ["httpheader"] = {
            "Content-Type: application/json",
        },
        ["postfields"] = schema,
        ["writefunction"] = wrire,
    })
    :perform()
    :close()

    return cjson.decode(result)
end

-- simple events generator by current event config
local function push_events()
    local events_config = cjson.decode(current_event_config)
    local event_schema = cjson.decode(fields_schema)
    for event_name, event_config in pairs(events_config or {}) do
        -- fetch fake event data from external server
        event_schema.required = event_config.fields
        local event_payload = generate_json(cjson.encode(event_schema))

        -- push some event to the engine
        local info = {
            ["name"] = event_name,
            ["data"] = event_payload,
            ["actions"] = {},
        }
        local result, list = event_engine:push_event(info)

        -- check result return variable as marker is there need to execute actions
        if result then
            for action_id, action_result in ipairs(action_engine:exec(__aid, list)) do
                print("action " .. tostring(action_id) .. " was requested: ", action_result)
            end
        end
    end
end

-- set default timeout to wait exit on blocking of recv_* functions
__api.set_recv_timeout(5000) -- 5s

__api.add_cbs({
    data = function(src, data)
        print("receive data: '" .. data .. "' from: " .. src)

        -- is internal communication from collector module
        if __imc.is_exist(src) then
            local mod_name, group_id = __imc.get_info(src)
            print("internal message received from: ", mod_name, group_id)
            return true
        else
            print("message received from the server")
        end

        -- network message from the server or other agents
        local msg = cjson.decode(data)
        if msg["type"] == "exec_events_req" then
            print("receive exec_events type message")
            push_events()
        else
            print("receive unknown type message", msg["type"])
        end
        return true
    end,

    -- file = function(src, path, name)
    -- text = function(src, text, name)
    -- msg = function(src, msg, mtype)
    -- action = function(src, data, name)

    control = function(cmtype, data)
        print("receive control msg: '" .. cmtype .. "' data: " .. data)

        -- cmtype: "quit"
        -- cmtype: "agent_connected"
        -- cmtype: "agent_disconnected"
        if cmtype == "update_config" then
            print("new config: ", __config.get_current_config())
            print("ctx: ")
            for i, k in pairs(__config.ctx) do
                print("\t", i, k)
            end

            -- renew current event engine instance
            local n_current_event_config = __config.get_current_event_config()
            local n_module_info = __config.get_module_info()
            if module_info ~= n_module_info or current_event_config ~= n_current_event_config then
                event_engine:free()
                module_info = n_module_info
                current_event_config = n_current_event_config
                event_engine = CEventEngine(fields_schema, current_event_config, module_info, prefix_db, true)
            end
        end
        return true
    end,
})

g_print("module " .. tostring(__config.ctx.name) .. " was started")

__api.await(-1)

g_print("module " .. tostring(__config.ctx.name) .. " was stopped")

return "success"
