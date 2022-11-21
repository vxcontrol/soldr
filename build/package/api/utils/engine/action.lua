require("yaci")
require("strict")
local pp    = require("pp")
local glue  = require("glue")
local cjson = require("cjson.safe")

CActionEngine = newclass("CActionEngine")

function CActionEngine:print(...)
    if self.is_debug then
        local t = glue.pack(...)
        for i, v in ipairs(t) do
            t[i] = pp.format(v)
        end
        if __log and __log.debug then
            __log.debug(glue.unpack(t))
        else
            print(glue.unpack(t))
        end
    end
end

function CActionEngine:init(_, is_debug)
    self.is_debug = false

    if type(is_debug) == "boolean" then
        self.is_debug = is_debug
    end
end

function CActionEngine:free()
    self:print("finalize CActionEngine object")
end

function CActionEngine:exec_log_to_db(agent_id, action, info)
    local sinfo = cjson.encode(info)
    self:print("execute action log to DB: ", agent_id, " : ", cjson.encode(action), " : ",  sinfo)
    return __api.push_event(agent_id, sinfo)
end

function CActionEngine:exec_module_action(agent_id, action, info)
    local sinfo = cjson.encode(info)
    self:print("send action to module on agent: ", agent_id, " : ", cjson.encode(action), " : ",  sinfo)
    if glue.indexof(action.module_name .. "." .. action.name, info.actions) ~= nil then
        self:print("action was already executed on this events chain: ", action.name)
        return false
    end

    for _, field in ipairs(action.fields) do
        if info.data[field] == nil then
            self:print("failed to find action field in event data: ", field)
            return false
        end
    end

    local token = __imc.make_token(action.module_name, __gid)
    local actions = glue.update({}, info.actions or {})
    table.insert(actions, action.module_name .. "." .. action.name)
    local info_data = glue.update({}, info.data or {})
    info_data.start_time = nil
    local action_data = cjson.encode({data = info_data, name = info.name, actions = actions, time = info.time})
    if not __api.send_action_to(token, action_data, action.name) then
        self:print("failed to execute action on module: ", action.module_name, " : ", action.name)
        return false
    end

    return true
end

function CActionEngine:exec(agent_id, list)
    local actions = {}

    for _, data in pairs(list) do
        table.sort(data.actions, function(a1, a2) return a1.priority > a2.priority end)

        for _, action in ipairs(data.actions) do
            local action_id = action.module_name .. "." .. action.name
            if action.module_name == "this" and action.name == "log_to_db" then
                actions[action_id] = self:exec_log_to_db(agent_id, action, data.info)
            elseif action.module_name and action.name then
                actions[action_id] = self:exec_module_action(agent_id, action, data.info)
            else
                self:print("unsupported action: ", cjson.encode(action))
                actions[action_id] = false
            end
        end
    end

    return actions
end
