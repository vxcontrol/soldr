local cjson = require("cjson.safe")

-- variables to initialize raw data sender
local receivers = {}

-- for overriding debug argument
local g_print = print
local print = function(...)
    if __args["debug"][1] == "true" then
        g_print(__config.ctx.name .. " : ", ...);
    end
end

-- simple interface to update global receivers list
local function update_receivers(current_config)
    receivers = {}
    local current_config_model = cjson.decode(current_config)
    for _, module_name in ipairs(current_config_model.receivers or {}) do
        receivers[module_name] = __imc.make_token(module_name, __gid)
    end
end

-- simple events generator by current event config
local function push_line(line)
    local payload = cjson.encode({
        ["type"] = "raw_collector_event",
        ["data"] = line,
        ["time"] = os.date("!%d.%m.%y %H:%M:%S"),
    })
    print("try to sent payload from collector: ", payload)
    for module_name, module_token in pairs(receivers) do
        local result = __api.send_data_to(module_token, payload)
        print("raw data sent to module '" .. tostring(module_name) .. "' result: ", result)
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
        if msg["type"] == "log_line_req" then
            print("receive log_line type message")
            push_line(msg.line or "")
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

            -- renew receivers list from new current config
            update_receivers(__config.get_current_config())
        end
        return true
    end,
})

g_print("module " .. tostring(__config.ctx.name) .. " was started")

update_receivers(__config.get_current_config())
__api.await(-1)

g_print("module " .. tostring(__config.ctx.name) .. " was stopped")

return "success"
