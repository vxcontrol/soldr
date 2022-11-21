require("yaci")
require("strict")
require("engine")
local glue  = require("glue")
local crc32 = require("crc32")
local cjson = require("cjson.safe")
math.randomseed(crc32(tostring({})))

CBaseEngine = newclass("CBaseEngine")

--[[
    cfg top keys:
    * config - module arguments (hard limits)
]]
function CBaseEngine:init(cfg)
    __log.debug("init CBaseEngine object")
    assert(type(cfg) == "table", "configuration object has invalid type")

    self.config = glue.merge(self.config or {}, cfg.config or {})
    self.config.actions = cjson.decode(__config.get_current_action_config())
    self.config.events = cjson.decode(__config.get_current_event_config())
    self.config.module = cjson.decode(__config.get_current_config())
    self.mod_name = __config.ctx.name
    self.is_debug = __args["debug_engines"][1] == "true"
    self.prefix_db = __gid .. "."
    self.server_token = ""

    self.action_engine = CActionEngine(
        {},
        __args["debug_correlator"][1] == "true"
    )
    self.event_engine = CEventEngine(
        __config.get_fields_schema(),
        __config.get_current_event_config(),
        __config.get_module_info(),
        self.prefix_db,
        __args["debug_correlator"][1] == "true"
    )
end

-- in: nil
-- out: nil
function CBaseEngine:free()
    __log.debug("finalize CBaseEngine object")

    if self.event_engine ~= nil then
        self.event_engine:free()
        self.event_engine = nil
    end

    if self.action_engine ~= nil then
        self.action_engine:free()
        self.action_engine = nil
    end
end

-- in: nil
-- out: nil
function CBaseEngine:run()
    __log.debug("run CBaseEngine started")
    while not __api.is_close() do
        local timeout = self:timer_cb()
        assert(type(timeout) == "number", "await timeout has invalid type")
        __api.await(timeout)
    end
    __log.debug("run CBaseEngine stopped")
end

-- in: string
--      destination token (string) of server module side
-- out: nil
function CBaseEngine:agent_connected(dst)
    __log.debugf("agent_connected CBaseEngine with token '%s'", dst)
    self:agent_connected_cb(dst)
end

-- in: string
--      destination token (string) of server module side
-- out: nil
function CBaseEngine:agent_disconnected(dst)
    __log.debugf("agent_disconnected CBaseEngine with token '%s'", dst)
    self:agent_disconnected_cb(dst)
end

-- in: nil
-- out: nil
function CBaseEngine:quit()
    __log.debug("quit CBaseEngine")
    self:quit_cb()
end

-- in: nil
-- out: nil
function CBaseEngine:update_config()
    __log.debug("update_config CBaseEngine")

    -- renew actions and events config according by the module configuration
    self.config.actions = cjson.decode(__config.get_current_action_config())
    self.config.events = cjson.decode(__config.get_current_event_config())
    self.config.module = cjson.decode(__config.get_current_config())

    -- renew current event engine instance
    if self.event_engine ~= nil then
        self.event_engine:free()
        self.event_engine = CEventEngine(
            __config.get_fields_schema(),
            __config.get_current_event_config(),
            __config.get_module_info(),
            self.prefix_db,
            __args["debug_correlator"][1] == "true"
        )
    end

    -- notify engine about update config from the server
    self:update_config_cb()
end

-- in: string, string, string, table
-- out: nil
function CBaseEngine:commit_event(src, event_name, action_name, action_data)
    __log.debug("commit_event CBaseEngine")

    if self.config.events[event_name] == nil then
        __log.errorf("requested event '%s' does not exists in current event config", event_name)
        return
    end

    local is_imc, mod_name, group_id = self:get_sender_info(src)
    action_data.data.action = {
        ["name"] = action_name,
        ["source"] = {
            ["is_imc"] = is_imc,
            ["mod_name"] = mod_name,
            ["group_id"] = group_id,
        },
    }
    self:push_event(event_name, action_data.data, action_data.actions)
end

-- in: string, string, table
-- out: nil
function CBaseEngine:commit_success(src, action_name, action_data)
    __log.debug("commit_success CBaseEngine")

    local event_name = tostring(self.mod_name) .. "_action_exec_success"
    action_data.data.result = true
    action_data.data.reason = "action_exec_success"

    self:commit_event(src, event_name, action_name, action_data)

    -- case to notify other side about action execution result
    if type(action_data.retaddr) == "string" and action_data.retaddr ~= "" then
        local data = cjson.encode(glue.merge({status = "success"}, action_data))
        __api.send_data_to(src, data)
    end
end

-- in: string, string, table
-- out: nil
function CBaseEngine:commit_failed(src, action_name, action_data)
    __log.debug("commit_failed CBaseEngine")

    local event_name = tostring(self.mod_name) .. "_action_exec_failed"
    action_data.data.result = false
    action_data.data.reason = "action_exec_failed"

    self:commit_event(src, event_name, action_name, action_data)

    -- case to notify other side about action execution result
    if type(action_data.retaddr) == "string" and action_data.retaddr ~= "" then
        local data = cjson.encode(glue.merge({status = "error"}, action_data))
        __api.send_data_to(src, data)
    end
end

-- in: string
--      source token (string) of sender module side
-- out: boolean, string, string
--      is it inter modules communication packet (boolean)
--      module name (string) which was a sender the packet
--      group id (string) which contained the sender module
function CBaseEngine:get_sender_info(src)
    if __imc.is_exist(src) then
        local mod_name, group_id = __imc.get_info(src)
        __log.debugf("received internal from module '%s' and group '%s'", mod_name, group_id)
        return true, mod_name, group_id
    end

    __log.debugf("message received from the server")
    return false, self.mod_name, __gid
end

-- in: nil
-- out: string
--      destination token (string) it'll be empty if agent disconnected
function CBaseEngine:get_server_token()
    local tablelength = function(t)
        local count = 0
        for _ in pairs(t) do count = count + 1 end
        return count
    end

    if tablelength(__agents.get_by_dst(self.server_token)) == 0 then
        self.server_token = ""
    else
        return self.server_token
    end
    for client_id, client_info in pairs(__agents.dump()) do
        if tostring(client_info.Type) == "VXAgent" then
            self.server_token = tostring(client_id)
            return self.server_token
        end
    end

    return ""
end

-- in: string, table, table or nil
--      event name (string) to execute it into event engine
--      event data (table) to store key-value struct into the event
--      previous actions array (table or nil) to avoid loop actions execution
-- out: boolean
--      result of event processing received from event engine (true if event exists)
function CBaseEngine:push_event(name, data, actions)
    assert(type(name) == "string", "input event name type is invalid")
    assert(type(data) == "table", "input event data type is invalid")
    assert(not actions or type(actions) == "table", "input event data type is invalid")
    assert(self.event_engine ~= nil, "event engine is not initialized")
    assert(self.action_engine ~= nil, "action engine is not initialized")
    __log.debugf("try push event '%s' with data json: %s", name, cjson.encode(data))

    if self.config.events[name] == nil then
        __log.errorf("requested event '%s' does not exists in current event config", name)
        return false
    end

    -- build event info structure
    local info = {
        ["name"] = name,
        ["data"] = data,
        ["actions"] = actions or {},
    }
    local event_result, action_list = self.event_engine:push_event(info)
    __log.debugf("result of pushing event '%s' is %s", name, event_result)

    -- check result return variable as marker is there need to execute actions
    if event_result then
        __log.debugf("try to exec event '%s' actions list %s", name, cjson.encode(action_list))
        for action_id, action_result in ipairs(self.action_engine:exec(__aid, action_list)) do
            __log.debugf("result of executing action '%s' is %s", action_id, action_result)
        end
    end

    return event_result
end

-- in: string, string
--      source token (string) of sender module side
--      data payload (string) as a custom string serialized struct object (json)
-- out: boolean
--      result of data processing received from acts_engine
function CBaseEngine:recv_data(src, data)
    assert(type(src) == "string", "sender module source token type is invalid")
    assert(type(data) == "string", "input action data type is invalid")
    __log.debugf("try process data with payload '%s' from '%s'", data, src)

    return self:recv_data_cb(src, data)
end

-- in: string, string, string
--      source token (string) of sender module side
--      file path (string) on local FS where received file was stored
--      file name (string) is a original file name which was set on sender side
-- out: boolean
--      result of file processing received from acts_engine
function CBaseEngine:recv_file(src, path, name)
    assert(type(src) == "string", "sender module source token type is invalid")
    assert(type(path) == "string", "input file path type is invalid")
    assert(type(name) == "string", "input file name type is invalid")
    __log.debugf("try process file with path '%s' and name '%s' from '%s'", path, name, src)

    return self:recv_file_cb(src, path, name)
end

-- in: string, string, string
--      source token (string) of sender module side
--      action data (string) json string as a arguments to execute action via acts_engine
--      action name (string) to execute it into the acts_engine
-- out: boolean
--      result of action processing received from acts_engine
function CBaseEngine:recv_action(src, data, name)
    assert(type(src) == "string", "sender module source token type is invalid")
    assert(type(data) == "string", "input action data type is invalid")
    assert(type(name) == "string", "input action name type is invalid")
    __log.debugf("try exec action '%s' with data json '%s' from '%s'", name, data, src)

    if self.config.actions[name] == nil then
        __log.errorf("requested action '%s' does not exists in current action config", name)
        return false
    end

    -- normalize action data payload
    local action_name, action_data = name, cjson.decode(data)
    action_data.data = action_data.data or {}
    action_data.actions = action_data.actions or {}

    -- add the action name to executed actions list to avoid infinity recursive calls
    local action_full_name = tostring(self.mod_name) .. "." .. action_name
    if glue.indexof(action_full_name, action_data.actions) == nil then
        table.insert(action_data.actions, action_full_name)
    end

    local action_result = self:recv_action_cb(src, action_data, action_name)
    if action_result then
        self:commit_success(src, action_name, action_data)
    else
        self:commit_failed(src, action_name, action_data)
    end
    return action_result
end

-- virtual methods
CBaseEngine:virtual("timer_cb")
CBaseEngine:virtual("quit_cb")
CBaseEngine:virtual("agent_connected_cb")
CBaseEngine:virtual("agent_disconnected_cb")
CBaseEngine:virtual("update_config_cb")
CBaseEngine:virtual("recv_data_cb")
CBaseEngine:virtual("recv_file_cb")
CBaseEngine:virtual("recv_action_cb")
