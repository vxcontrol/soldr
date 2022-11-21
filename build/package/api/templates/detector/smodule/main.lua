local cjson = require("cjson.safe")

-- for overriding debug argument
local g_print = print
local print = function(...)
    if __args["debug"][1] == "true" then
        g_print(__config.ctx.name .. " : ", ...);
    end
end

-- set default timeout to wait exit on blocking of recv_* functions
__api.set_recv_timeout(5000) -- 5s

__api.add_cbs({
    data = function(src, data)
        print("receive data: '" .. data .. "' from: " .. src)
        local msg = cjson.decode(data)
        if msg["type"] == "exec_events_req" then
            local response = {
                ["type"] = "exec_events_resp",
                ["data"] = "events submitted",
                ["agents"] = {},
            }
            for dst, a in pairs(__agents.dump()) do
                if tostring(a.Type) == "VXAgent" then
                    __api.send_data_to(dst, data)
                    table.insert(response.agents, a.ID)
                end
            end
            print("sent server resp msg: ", __api.send_data_to(src, cjson.encode(response)))
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
        end
        return true
    end,
})

g_print("module " .. tostring(__config.ctx.name) .. " was started")

__api.await(-1)

g_print("module " .. tostring(__config.ctx.name) .. " was stopped")

return "success"
