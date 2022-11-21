require("yaci")
require("strict")
require("engines.base_engine")

CActsEngine = newclass("CActsEngine", CBaseEngine)

--[[
    cfg top keys:
    * config - module arguments (hard limits)
]]
function CActsEngine:init(cfg)
    __log.debug("init CActsEngine object")
    assert(type(cfg) == "table", "configuration object has invalid type")

    cfg.engine = "acts_engine"
    self.super:init(cfg)

    -- initialization of object after base class constructing
    self:update_config_cb()
end

-- in: nil
-- out: nil
function CActsEngine:free()
    __log.debug("finalize CActsEngine object")

    -- here will be triggered after closing vxproto object (destructor of the state)
end

-- in: nil
-- out: number
--      amount of milliseconds timeout to wait next call of the timer_cb
function CActsEngine:timer_cb()
    __log.debug("timer_cb CActsEngine")

    -- return infinity waiting next timer call
    -- otherways here need use milliseconds timeout to wait next call
    return -1
end

-- in: nil
-- out: nil
function CActsEngine:quit_cb()
    __log.debug("quit_cb CActsEngine")

    -- here will be triggered before closing vxproto object and destroying the state
end

-- in: string
--      destination token (string) of server module side
-- out: nil
function CActsEngine:agent_connected_cb(dst)
    __log.debugf("agent_connected_cb CActsEngine with token '%s'", dst)
end

-- in: string
--      destination token (string) of server module side
-- out: nil
function CActsEngine:agent_disconnected_cb(dst)
    __log.debugf("agent_disconnected_cb CActsEngine with token '%s'", dst)
end

-- in: nil
-- out: nil
function CActsEngine:update_config_cb()
    __log.debug("update_config_cb CActsEngine")

    -- actual current configuration contains into next fields
    -- self.config.actions
    -- self.config.events
    -- self.config.module
end

-- in: string, string
--      source token (string) of sender module side
--      data payload (string) as a custom string serialized struct object (json)
-- out: boolean
--      result of data processing from business logic
function CActsEngine:recv_data_cb(src, data)
    __log.debugf("perform custom logic for data with payload '%s' from '%s'", data, src)
    return true
end

-- in: string, string, string
--      source token (string) of sender module side
--      file path (string) on local FS where received file was stored
--      file name (string) is a original file name which was set on sender side
-- out: boolean
--      result of file processing from business logic
function CActsEngine:recv_file_cb(src, path, name)
    __log.debugf("perform custom logic for file with path '%s' and name '%s' from '%s'", path, name, src)
    return true
end

-- in: string, string, table
--      source token (string) of sender module side
--      action name (string) to execute it into the acts_engine
--      action data (table) as a arguments to execute action via acts_engine
--        e.x. {"data": {"key": "val"}, "actions": ["mod1.act1"]}
-- out: boolean
--      result of action processing from business logic
function CActsEngine:recv_action_cb(src, data, name)
    __log.debugf("perform custom logic for action '%s' from '%s'", name, src)

    local acts_engine = CActsEngine:cast(self)
    if name == "some_action" then
        return acts_engine:dummy(data.data["arg"])
    end

    return false
end

-- in: any
-- out: boolean
function CActsEngine:dummy(arg)
    __log.debugf("dummy CActsEngine with arg type: %s", type(arg))
    return true
end
