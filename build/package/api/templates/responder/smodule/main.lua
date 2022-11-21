local cjson = require("cjson.safe")

-- getting agent ID by dst token and agent type
local function get_agent_id_by_dst(dst, atype)
    for client_id, client_info in pairs(__agents.get_by_dst(dst)) do
        if client_id == dst then
            if tostring(client_info.Type) == atype or atype == "any" then
                return tostring(client_info.ID), client_info
            end
        end
    end
    return "", {}
end

-- getting agent source token by ID and agent type
local function get_agent_src_by_id(id, atype)
    for client_id, client_info in pairs(__agents.get_by_id(id)) do
        if tostring(client_info.Type) == atype or atype == "any" then
            return tostring(client_id), client_info
        end
    end
    return "", {}
end

-- set default timeout to wait exit on blocking of recv_* functions
__api.set_recv_timeout(5000) -- 5s

__api.add_cbs({
    data = function(src, data)
        __log.debugf("receive data from '%s' with data %s", src, data)

        local payload = cjson.decode(data)
        assert(type(payload) == "table", "input data type is invalid")

        local retaddr = payload.retaddr
        if type(retaddr) == "string" and retaddr ~= "" then
            payload.retaddr = nil
            __log.debugf("send response data to '%s'", retaddr)
            __api.send_data_to(retaddr, cjson.encode(payload))
            return true
        end

        return true
    end,

    -- file = function(src, path, name)
    -- text = function(src, text, name)
    -- msg = function(src, msg, mtype)

    action = function(src, data, name)
        __log.debugf("receive action '%s' from '%s' with data %s", name, src, data)

        local action_data = cjson.decode(data)
        assert(type(action_data) == "table", "input action data type is invalid")
        action_data.retaddr = src
        local id, _ = get_agent_id_by_dst(src, "any")
        local dst, _ = get_agent_src_by_id(id, "VXAgent")
        if dst ~= "" then
            __log.debugf("send action request to '%s'", dst)
            __api.send_action_to(dst, cjson.encode(action_data), name)
        else
            local payload = {
                status = "error",
                error = "connection_error",
            }
            __log.debugf("send response data to '%s'", src)
            __api.send_data_to(src, cjson.encode(payload))
        end

        return true
    end,

    control = function(cmtype, data)
        __log.debugf("receive control msg '%s' with data %s", cmtype, data)

        -- cmtype: "quit"
        -- cmtype: "agent_connected"
        -- cmtype: "agent_disconnected"
        -- cmtype: "update_config"

        return true
    end,
})

__log.infof("module '%s' was started", __config.ctx.name)
__api.await(-1)
__log.infof("module '%s' was stopped", __config.ctx.name)

return "success"
