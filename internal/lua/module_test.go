package lua_test

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vxcontrol/luar"

	"soldr/internal/lua"
	"soldr/internal/protoagent"
	"soldr/internal/utils"
	"soldr/internal/vxproto"
	"soldr/internal/vxproto/tunnel"
	tunnelRC4 "soldr/internal/vxproto/tunnel/rc4"
)

func TestGetRunningExecutablePath(t *testing.T) {
	p, err := lua.GetRunningExecutablePath()
	if err != nil {
		t.Fatalf("failed to get the running executable path: %b", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("failed to get info on the running executable file: %v", err)
	}
}

// Connection string to test network communication via vxproto
const netConnectionString = "ws://127.0.0.1:28080"

func randString(nchars int) string {
	rbytes := make([]byte, nchars)
	if _, err := rand.Read(rbytes); err != nil {
		return ""
	}

	return hex.EncodeToString(rbytes)
}

type FakeMainModule struct{}

func (mModule *FakeMainModule) DefaultRecvPacket(ctx context.Context, packet *vxproto.Packet) error {
	return nil
}

func (mModule *FakeMainModule) HasAgentInfoValid(ctx context.Context, iasocket vxproto.IAgentSocket) error {
	iasocket.SetGroupID("")
	return nil
}

func (mModule *FakeMainModule) GetVersion() string {
	return "develop"
}

func (mModule *FakeMainModule) GetAgentConnectionValidator() vxproto.AgentConnectionValidator {
	return &agentConnectionValidator{}
}

func (mModule *FakeMainModule) GetConnectionValidatorFactory() vxproto.ConnectionValidatorFactory {
	return &connectionValidatorFactory{}
}

type agentConnectionValidator struct{}

func (acv *agentConnectionValidator) OnInitConnect(context.Context, vxproto.SyncWS, *protoagent.Information) error {
	return nil
}

func (acv *agentConnectionValidator) ProcessError(err error) error {
	return err
}

func (acv *agentConnectionValidator) OnConnect(
	ctx context.Context,
	iasocket vxproto.IAgentSocket,
	packEncryptor tunnel.PackEncryptor,
	configurePingee func(p vxproto.Pinger) error,
	info *protoagent.Information,
) error {
	configurePingee(&pinger{})
	seconds := time.Now().Unix()
	stoken := randString(20)
	user := &protoagent.Information_User{
		Name:   utils.GetRef("root"),
		Groups: []string{"root"},
	}
	iasocket.SetAuthReq(&protoagent.AuthenticationRequest{
		Timestamp: &seconds,
		Atoken:    utils.GetRef(""),
		Aversion:  utils.GetRef("develop"),
	})
	iasocket.SetAuthResp(&protoagent.AuthenticationResponse{
		Atoken:   utils.GetRef(iasocket.GetSource()),
		Stoken:   utils.GetRef(stoken),
		Sversion: utils.GetRef(iasocket.GetVersion()),
		Status:   utils.GetRef("authorized"),
	})
	iasocket.SetInfo(&protoagent.Information{
		Os: &protoagent.Information_OS{
			Type: utils.GetRef("linux"),
			Name: utils.GetRef("Ubuntu 16.04"),
			Arch: utils.GetRef("amd64"),
		},
		Net: &protoagent.Information_Net{
			Hostname: utils.GetRef("test_pc"),
			Ips:      []string{"127.0.0.1/8"},
		},
		Users: []*protoagent.Information_User{
			user,
		},
	})
	return nil
}

type connectionValidatorFactory struct{}

func (cvf *connectionValidatorFactory) NewValidator(version string) (vxproto.ServerConnectionValidator, error) {
	return &serverConnectionValidator{}, nil
}

type serverConnectionValidator struct{}

func (scv *serverConnectionValidator) GetAgentConnectionInfo(
	ctx context.Context,
	info *vxproto.AgentInfoForIDFetcher,
) (*vxproto.AgentConnectionInfo, error) {
	return &vxproto.AgentConnectionInfo{
		ID:         info.ID,
		GroupID:    0,
		AuthStatus: "authorized",
	}, nil
}

func (scv *serverConnectionValidator) CheckInitConnectionTLS(s *tls.ConnectionState) error {
	return nil
}

func (scv *serverConnectionValidator) OnInitConnect(
	ctx context.Context,
	tlsConnState *tls.ConnectionState,
	socket vxproto.SyncWS,
	connInfo *vxproto.InitConnectionInfo,
) error {
	return nil
}

func (scv *serverConnectionValidator) CheckConnectionTLS(s *tls.ConnectionState) error {
	return nil
}

func (scv *serverConnectionValidator) OnConnect(
	ctx context.Context,
	tlsConnState *tls.ConnectionState,
	socket vxproto.IAgentSocket,
	agentType vxproto.AgentType,
	configurePackEncryptor func(c *tunnel.Config) error,
	configurePinger func(p vxproto.Pinger),
) error {
	configurePinger(&pinger{})
	configurePackEncryptor(&tunnel.Config{
		RC4: &tunnelRC4.Config{
			Key: []byte{42},
		},
	})
	return nil
}

type pinger struct{}

func (p *pinger) Start(ctx context.Context, ping func(ctx context.Context, nonce []byte) error) error {
	return nil
}
func (p *pinger) Process(ctx context.Context, pingData []byte) error {
	return nil
}

func (p *pinger) Stop(ctx context.Context) error {
	return nil
}

type MFiles map[string][]byte
type MArgs map[string][]string

func init() {
	logrus.SetOutput(ioutil.Discard)
}

// Init module
func initModule(files MFiles, args MArgs, name string, proto vxproto.IVXProto) (*lua.Module, *lua.State) {
	if proto == nil {
		return nil, nil
	}

	moduleSocket := proto.NewModule(name, "")
	if moduleSocket == nil {
		return nil, nil
	}

	if !proto.AddModule(moduleSocket) {
		return nil, nil
	}

	state, err := lua.NewState(files)
	if err != nil {
		return nil, nil
	}

	module, err := lua.NewModule(args, state, moduleSocket)
	if err != nil {
		return nil, nil
	}

	luar.Register(state.L, "", luar.Map{
		"print": fmt.Println,
	})

	return module, state
}

// Run main logic of module
func runModule(module *lua.Module) string {
	if module == nil {
		return "internal error"
	}
	module.Start()
	return module.GetResult()
}

// Run simple test with arguments for state
func Example_args() {
	ctx := context.Background()
	arg1 := []string{"a", "b", "c", "d"}
	arg2 := []string{"e"}
	arg3 := []string{}
	arg4 := []string{"f", "g"}
	args := map[string][]string{
		"arg1": arg1,
		"arg2": arg2,
		"arg3": arg3,
		"arg4": arg4,
	}
	files := map[string][]byte{
		"main.lua": []byte(`
			local arg_keys = {}
			for k, _ in pairs(__args) do
				table.insert(arg_keys, k)
			end
			table.sort(arg_keys)
			for i, k in ipairs(arg_keys) do
				v = __args[k]
				print(i, k, table.getn(v))
				for j, p in pairs(v) do
					print(j, p)
				end
			end
			return "success"
		`),
	}

	proto, _ := vxproto.New(&FakeMainModule{})
	module, _ := initModule(files, args, "test_module", proto)
	fmt.Println("Result: ", runModule(module))
	module.Close()
	proto.Close(ctx)
	// Output:
	//1 arg1 4
	//1 a
	//2 b
	//3 c
	//4 d
	//2 arg2 1
	//1 e
	//3 arg3 0
	//4 arg4 2
	//1 f
	//2 g
	//Result:  success
}

// Run simple test with await function (1 second)
func Example_module_await() {
	ctx := context.Background()
	args := map[string][]string{}
	files := map[string][]byte{
		"main.lua": []byte(`
			print(__api.get_name())
			__api.await(1000) -- 1 sec
			return "success"
		`),
	}

	proto, _ := vxproto.New(&FakeMainModule{})
	module, _ := initModule(files, args, "test_module", proto)
	fmt.Println("Result: ", runModule(module))
	module.Close()
	proto.Close(ctx)
	// Output:
	//test_module
	//Result:  success
}

// Run simple test with await function (infinity)
func Example_module_await_inf() {
	ctx := context.Background()
	args := map[string][]string{}
	files := map[string][]byte{
		"main.lua": []byte(`
			print(__api.get_name())
			__api.await(-1)
			return "success"
		`),
	}

	var wg sync.WaitGroup
	proto, _ := vxproto.New(&FakeMainModule{})
	module, _ := initModule(files, args, "test_module", proto)
	wg.Add(1)
	go func() {
		fmt.Println("Result: ", runModule(module))
		wg.Done()
	}()

	time.Sleep(time.Second)
	module.Close()
	proto.Close(ctx)
	wg.Wait()
	// Output:
	//test_module
	//Result:  success
}

func Test_module_exepath(t *testing.T) {
	ctx := context.Background()
	args := map[string][]string{}
	files := map[string][]byte{
		"main.lua": []byte(`
			local exec_path = __api.get_exec_path()
			__api.await(-1)
			return exec_path
		`),
	}

	var wg sync.WaitGroup
	proto, _ := vxproto.New(&FakeMainModule{})
	module, _ := initModule(files, args, "test_module", proto)
	var actualExepath string
	wg.Add(1)
	go func() {
		actualExepath = runModule(module)
		wg.Done()
	}()

	expectedExepath, err := lua.GetRunningExecutablePath()
	if err != nil {
		t.Fatalf("failed to get a running executable path: %v", err)
	}
	time.Sleep(time.Second)
	module.Close()
	proto.Close(ctx)
	wg.Wait()

	if actualExepath != expectedExepath {
		t.Fatalf("expected path %v, got %v", expectedExepath, actualExepath)
	}
}

// Run simple test with all API functions
func Example_api() {
	ctx := context.Background()
	args := map[string][]string{}
	files := map[string][]byte{
		"main.lua": []byte(`
			__api.set_recv_timeout(10) -- 10 ms

			-- src is string
			-- data is string
			-- path is string
			-- name is string
			-- mtype is int:
			--   {DEBUG: 0, INFO: 1, WARNING: 2, ERROR: 3}
			-- cmtype is string
			-- data is string
			if __api.add_cbs({
				["data"] = function(src, data)
					return true
				end,
				["file"] = function(src, path, name)
					return true
				end,
				["text"] = function(src, text, name)
					return true
				end,
				["msg"] = function(src, msg, mtype)
					return true
				end,
				["action"] = function(src, data, name)
					return true
				end,
				["control"] = function(cmtype, data)
					return true
				end,
			}) == false then
				return "failed"
			end
			if __api.del_cbs({ "data", "file", "text", "msg", "action", "control" }) == false then
				return "failed"
			end

			-- res is boolean
			res = __api.send_data_to("def_token", "test_data")
			res = __api.send_file_to("def_token", "file_data", "file_name")
			res = __api.send_text_to("def_token", "text_data", "text_name")
			res = __api.send_msg_to("def_token", "msg_data", 0)
			res = __api.send_action_to("def_token", "action_data", "action_name")
			res = __api.send_file_from_fs_to("def_token", "file_path", "file_name")

			-- res and send_res are boolean
			local function result_cb(send_res)
				-- use send_res to check of sending result
			end
			res = __api.async_send_data_to("def_token", "test_data", result_cb)

			-- res is boolean
			src, data, res = __api.recv_data()
			src, path, name, res = __api.recv_file()
			src, text, name, res = __api.recv_text()
			src, msg, mtype, res = __api.recv_msg()
			src, data, name, res = __api.recv_action()

			-- res is boolean
			data, res = __api.recv_data_from("def_token")
			path, name, res = __api.recv_file_from("def_token")
			text, name, res = __api.recv_text_from("def_token")
			msg, mtype, res = __api.recv_msg_from("def_token")
			data, name, res = __api.recv_action_from("def_token")

			return "success"
		`),
	}

	proto, _ := vxproto.New(&FakeMainModule{})
	module, _ := initModule(files, args, "test_module", proto)
	fmt.Println("Result: ", runModule(module))
	module.Close()
	proto.Close(ctx)
	// Output:
	//Result:  success
}

// Run IMC test with the API functions
func Example_imc() {
	ctx := context.Background()
	var wg sync.WaitGroup
	mname1 := []string{"test_module1", "sender"}
	mname2 := []string{"test_module2", "receiver"}
	args1 := map[string][]string{
		"dst_module": mname2,
	}
	args2 := map[string][]string{
		"dst_module": mname1,
	}
	files := map[string][]byte{
		"main.lua": []byte(`
			local imc_token = __imc.make_token(__args["dst_module"][1], "")
			__api.set_recv_timeout(1000)

			local this_imc_token = __imc.get_token()
			local myModName, myGroupID = __imc.get_info(this_imc_token)
			if type(this_imc_token) ~= "string" or #this_imc_token ~= 40 then
				return "failed"
			end
			if __imc.is_exist(imc_token) == false then
				return "failed"
			end
			local modName, groupID, isExist = __imc.get_info(imc_token)
			if groupID ~= "" or modName ~= __args["dst_module"][1] or isExist ~= true then
				return "failed"
			end

			local global_groups = __imc.get_groups()
			local local_groups = __imc.get_groups_by_mid(myModName)
			if #global_groups ~= 1 or #local_groups ~= 1 then
				return "failed"
			end
			-- it's marker of shared lua state
			if global_groups[1] ~= "" or local_groups[1] ~= "" then
				return "failed"
			end

			local global_modules = __imc.get_modules()
			local local_modules = __imc.get_modules_by_gid(myGroupID)
			if #global_modules ~= 2 or #local_modules ~= 2 then
				return "failed"
			end
			if global_modules[1] ~= myModName and global_modules[2] ~= myModName then
				return "failed"
			end
			if global_modules[1] ~= modName and global_modules[2] ~= modName then
				return "failed"
			end
			if local_modules[1] ~= myModName and local_modules[2] ~= myModName then
				return "failed"
			end
			if local_modules[1] ~= modName and local_modules[2] ~= modName then
				return "failed"
			end

			local read_file = function(file)
				local fh = assert(io.open(file, "rb"))
				local content = fh:read("*all")
				fh:close()
				return content
			end

			local td = "test_data"
			local td_direct = "test_data_direct"
			local td_callback = "test_data_callback"
			local fl_name = "test_file_name.tmp"
			local msg_type = 2
			local recv_callbacks = 0
			local send_callbacks = 0

			__api.add_cbs({
				data = function(src, data)
					if src ~= imc_token or data ~= td_callback then
						return false
					end
					recv_callbacks = recv_callbacks + 1
					return true
				end,

				file = function(src, path, name)
					if src ~= imc_token or read_file(path) ~= td_callback or name ~= fl_name then
						return false
					end
					recv_callbacks = recv_callbacks + 1
					return true
				end,

				text = function(src, text, name)
					if src ~= imc_token or text ~= td_callback or name ~= fl_name then
						return false
					end
					recv_callbacks = recv_callbacks + 1
					return true
				end,

				msg = function(src, msg, mtype)
					if src ~= imc_token or msg ~= td_callback or mtype ~= msg_type then
						return false
					end
					recv_callbacks = recv_callbacks + 1
					return true
				end,

				action = function(src, data, name)
					if src ~= imc_token or data ~= td_callback or name ~= fl_name then
						return false
					end
					recv_callbacks = recv_callbacks + 1
					return true
				end,

				control = function(cmtype, data)
					return true
				end,
			})

			if __args["dst_module"][2] == "sender" then
				__api.await(100)

				if __api.send_data_to(imc_token, td) == false then
					return "failed"
				end

				if __api.send_data_to(imc_token, td_direct) == false then
					return "failed"
				end

				if __api.send_file_to(imc_token, td, fl_name) == false then
					return "failed"
				end

				if __api.send_file_to(imc_token, td_direct, fl_name) == false then
					return "failed"
				end

				if __api.send_text_to(imc_token, td, fl_name) == false then
					return "failed"
				end

				if __api.send_text_to(imc_token, td_direct, fl_name) == false then
					return "failed"
				end

				if __api.send_msg_to(imc_token, td, msg_type) == false then
					return "failed"
				end

				if __api.send_msg_to(imc_token, td_direct, msg_type) == false then
					return "failed"
				end

				if __api.send_action_to(imc_token, td, fl_name) == false then
					return "failed"
				end

				if __api.send_action_to(imc_token, td_direct, fl_name) == false then
					return "failed"
				end

				local function send_result_callback(res)
					if res then
						send_callbacks = send_callbacks + 1
					end
				end

				if __api.async_send_data_to(imc_token, td_callback, send_result_callback) == false then
					return "failed"
				end

				if __api.async_send_file_to(imc_token, td_callback, fl_name, send_result_callback) == false then
					return "failed"
				end

				if __api.async_send_text_to(imc_token, td_callback, fl_name, send_result_callback) == false then
					return "failed"
				end

				if __api.async_send_msg_to(imc_token, td_callback, msg_type, send_result_callback) == false then
					return "failed"
				end

				if __api.async_send_action_to(imc_token, td_callback, fl_name, send_result_callback) == false then
					return "failed"
				end

				for i=0,20 do
					__api.await(50)
					if send_callbacks == 5 then break end
				end
				if send_callbacks ~= 5 then return "failed" end
			end

			if __args["dst_module"][2] == "receiver" then
				__api.use_sync_mode()

				local src, data, path, name, res
				src, data, res = __api.recv_data()
				if src ~= imc_token or data ~= td or res ~= true then
					return "failed"
				end

				data, res = __api.recv_data_from(imc_token)
				if data ~= td_direct or res ~= true then
					return "failed"
				end

				src, path, name, res = __api.recv_file()
				if src ~= imc_token or read_file(path) ~= td or name ~= fl_name or res ~= true then
					return "failed"
				end

				path, name, res = __api.recv_file_from(imc_token)
				if read_file(path) ~= td_direct or name ~= fl_name or res ~= true then
					return "failed"
				end

				src, text, name, res = __api.recv_text()
				if src ~= imc_token or text ~= td or name ~= fl_name or res ~= true then
					return "failed"
				end

				text, name, res = __api.recv_text_from(imc_token)
				if text ~= td_direct or name ~= fl_name or res ~= true then
					return "failed"
				end

				src, msg, mtype, res = __api.recv_msg()
				if src ~= imc_token or msg ~= td or mtype ~= msg_type or res ~= true then
					return "failed"
				end

				msg, mtype, res = __api.recv_msg_from(imc_token)
				if msg ~= td_direct or mtype ~= msg_type or res ~= true then
					return "failed"
				end

				src, data, name, res = __api.recv_action()
				if src ~= imc_token or data ~= td or name ~= fl_name or res ~= true then
					return "failed"
				end

				data, name, res = __api.recv_action_from(imc_token)
				if data ~= td_direct or name ~= fl_name or res ~= true then
					return "failed"
				end

				__api.use_async_mode()

				for i=0,20 do
					__api.await(50)
					if recv_callbacks == 5 then break end
				end
				if recv_callbacks ~= 5 then return "failed" end
			end

			return "success"
		`),
	}

	proto, _ := vxproto.New(&FakeMainModule{})
	module1, _ := initModule(files, args1, mname1[0], proto)
	module2, _ := initModule(files, args2, mname2[0], proto)

	wg.Add(1)
	go func() {
		fmt.Println("Result1: ", runModule(module1))
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		fmt.Println("Result2: ", runModule(module2))
		wg.Done()
	}()
	wg.Wait()
	module1.Close()
	module2.Close()
	proto.Close(ctx)
	// Unordered output:
	//Result1:  success
	//Result2:  success
}

// Run IMC test with call send data from recv callback
func TestIMCSendDataFromCallback(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup
	mname1 := []string{"test_module1", "sender"}
	mname2 := []string{"test_module2", "receiver"}
	args1 := map[string][]string{
		"dst_module": mname2,
	}
	args2 := map[string][]string{
		"dst_module": mname1,
	}
	header := []byte(`
		local res
		local result = "failed"
		local steps = {"step1", "step2", "step3", "step4"}
		local imc_token = __imc.make_token(__args["dst_module"][1], "")
		__api.set_recv_timeout(1000)

		local this_imc_token = __imc.get_token()
		if type(this_imc_token) ~= "string" or #this_imc_token ~= 40 then
			return result
		end
		if __imc.is_exist(imc_token) == false then
			return result
		end
		local modName, groupID, isExist = __imc.get_info(imc_token)
		if groupID ~= "" or modName ~= __args["dst_module"][1] or isExist ~= true then
			return result
		end
	`)
	sender := []byte(`
		__api.await(200)
		if not __api.send_data_to(imc_token, steps[1]) then
			return "failed"
		end
	`)
	footer := []byte(`
		for i=0,10 do
			__api.await(50)
			if result == "success" then return result end
		end

		return result
	`)
	files1 := []map[string][]byte{
		{
			"main.lua": append(append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data ~= steps[2] then return false end
						if not __api.send_data_to(src, data) then return false end
						result = "success"
						return true
					end,
				})
			`)...), sender...), footer...),
		},
		{
			"main.lua": append(append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if not __api.send_data_to(src, data) then return false end
						if data == steps[4] then
							result = "success"
						end
						return true
					end,
				})
			`)...), sender...), footer...),
		},
		{
			"main.lua": append(append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data ~= steps[2] then return false end

						if not __api.send_data_to(src, data) then return false end
						data, res = __api.recv_data_from(src)
						if not res then return false end
						if data ~= steps[3] then return false end
						if not __api.send_data_to(src, data) then return false end

						if not __api.send_data_to(src, data) then return false end
						data, res = __api.recv_data_from(src)
						if not res then return false end
						if data ~= steps[4] then return false end
						if not __api.send_data_to(src, data) then return false end

						if not __api.send_data_to(src, data) then return false end
						if not __api.send_data_to(src, data) then return false end
						result = "success"
						return true
					end,
				})
			`)...), sender...), footer...),
		},
	}
	files2 := []map[string][]byte{
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data == steps[1] and __api.send_data_to(src, steps[2]) then
							return true
						elseif data == steps[2] then
							result = "success"
							return true
						end
						return false
					end,
				})
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data == steps[1] and __api.send_data_to(src, steps[2]) then
							return true
						elseif data == steps[2] and __api.send_data_to(src, steps[3]) then
							return true
						elseif data == steps[3] and __api.send_data_to(src, steps[4]) then
							return true
						elseif data == steps[4] then
							result = "success"
							return true
						end
						return false
					end,
				})
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data == steps[1] and __api.send_data_to(src, steps[2]) then
							return true
						elseif data == steps[2] and __api.send_data_to(src, steps[3]) then
							data, res = __api.recv_data_from(src)
							if not res or data ~= steps[3] then return false end
							return true
						elseif data == steps[3] and __api.send_data_to(src, steps[4]) then
							data, res = __api.recv_data_from(src)
							if not res or data ~= steps[4] then return false end
							return true
						elseif data == steps[4] then
							data, res = __api.recv_data_from(src)
							if not res or data ~= steps[4] then return false end
							result = "success"
							return true
						end
						return false
					end,
				})
			`)...), footer...),
		},
	}

	for idx := 0; idx < 3; idx++ {
		proto, _ := vxproto.New(&FakeMainModule{})
		module1, _ := initModule(files1[idx], args1, mname1[0], proto)
		module2, _ := initModule(files2[idx], args2, mname2[0], proto)

		wg.Add(1)
		go func() {
			if result := runModule(module1); result != "success" {
				t.Errorf("error in sender module iterate %d: '%s'", idx, result)
			}
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			if result := runModule(module2); result != "success" {
				t.Errorf("error in receiver module iterate %d: '%s'", idx, result)
			}
			wg.Done()
		}()
		wg.Wait()
		module1.Close()
		module2.Close()
		proto.Close(ctx)
	}
}

// Run IMC test with call send data from recv callback
func TestIMCAsyncSendData(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup
	mname1 := []string{"test_module1", "sender"}
	mname2 := []string{"test_module2", "receiver"}
	args1 := map[string][]string{
		"dst_module": mname2,
	}
	args2 := map[string][]string{
		"dst_module": mname1,
	}
	header := []byte(`
		local res
		local result = "failed"
		local steps = {"step1", "step2", "step3", "step4"}
		local imc_token = __imc.make_token(__args["dst_module"][1], "")
		__api.set_recv_timeout(1000)

		local this_imc_token = __imc.get_token()
		if type(this_imc_token) ~= "string" or #this_imc_token ~= 40 then
			return result
		end
		if __imc.is_exist(imc_token) == false then
			return result
		end
		local modName, groupID, isExist = __imc.get_info(imc_token)
		if groupID ~= "" or modName ~= __args["dst_module"][1] or isExist ~= true then
			return result
		end
	`)
	footer := []byte(`
		for i=0,10 do
			__api.await(50)
			if result == "success" then return result end
		end

		return result
	`)
	files1 := []map[string][]byte{
		{
			"main.lua": append(append(header, []byte(`
				__api.await(200)
				res = __api.async_send_data_to(imc_token, steps[1], function(r)
					if r then
						result = "success"
					end
				end)
				if not res then
					return "failed"
				end
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.await(200)
				do
					res = __api.async_send_data_to(imc_token, steps[1], function(r)
						if r then
							result = "success"
						end
					end)
				end
				collectgarbage("collect")
				__api.await(200)
				collectgarbage("collect")
				__api.await(200)
				collectgarbage("collect")
				if not res then
					return "failed"
				end
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.await(200)
				local step = 1
				local function send_result_callback(r)
					if step == 5 then
						result = "success"
						return
					end
					if r and __api.async_send_data_to(imc_token, steps[step], send_result_callback) then
						step = step + 1
					end
				end

				send_result_callback(true)
				for i=0,20 do
					__api.await(50)
					if step == 5 then break end
				end
				if step ~= 5 then
					return "failed"
				end
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if data == steps[2] then
							result = "success"
						end
						return true
					end,
				})

				__api.await(200)
				if not __api.async_send_data_to(imc_token, steps[1], nil) then
					return "failed"
				end
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						return __api.async_send_data_to(src, data, function(r)
							if r then
								result = "success"
							end
						end)
					end,
				})

				__api.await(200)
				if not __api.send_data_to(imc_token, steps[1]) then
					return "failed"
				end
			`)...), footer...),
		},
	}
	files2 := []map[string][]byte{
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data == steps[1] then
							result = "success"
							return true
						end
						return false
					end,
				})
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						__api.await(300)
						if src ~= imc_token then return false end
						if data == steps[1] then
							result = "success"
							return true
						end
						return false
					end,
				})
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data == steps[4] then
							result = "success"
							return true
						end
						return false
					end,
				})
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data == steps[1] and __api.async_send_data_to(src, steps[2], nil) then
							result = "success"
							return true
						end
						return false
					end,
				})
			`)...), footer...),
		},
		{
			"main.lua": append(append(header, []byte(`
				__api.add_cbs({
					["data"] = function(src, data)
						if src ~= imc_token then return false end
						if data == steps[1] and __api.send_data_to(src, steps[2]) then
							return true
						elseif data == steps[2] then
							result = "success"
							return true
						end
						return false
					end,
				})
			`)...), footer...),
		},
	}

	for idx := 0; idx < 4; idx++ {
		proto, _ := vxproto.New(&FakeMainModule{})
		module1, _ := initModule(files1[idx], args1, mname1[0], proto)
		module2, _ := initModule(files2[idx], args2, mname2[0], proto)

		wg.Add(1)
		go func() {
			if result := runModule(module1); result != "success" {
				t.Errorf("error in sender module iterate %d: '%s'", idx, result)
			}
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			if result := runModule(module2); result != "success" {
				t.Errorf("error in receiver module iterate %d: '%s'", idx, result)
			}
			wg.Done()
		}()
		wg.Wait()
		module1.Close()
		module2.Close()
		proto.Close(ctx)
	}
}

// Run simple test to start of stopped module
func TestLuaStartStoppedModule(t *testing.T) {
	ctx := context.Background()
	args := map[string][]string{}
	files := map[string][]byte{
		"main.lua": []byte(`return 'success'`),
	}

	proto, _ := vxproto.New(&FakeMainModule{})
	module, _ := initModule(files, args, "test_module", proto)
	if runModule(module) != "success" {
		t.Fatal("Error on getting result from module (step 1)")
	}
	module.Stop()
	if runModule(module) != "success" {
		t.Fatal("Error on getting result from module (step 2)")
	}
	module.Close()
	if err := proto.Close(ctx); err != nil {
		t.Fatal("Error on close vxproto object: ", err.Error())
	}
}

func BenchmarkLuaLoadModuleWithMainModule(b *testing.B) {
	ctx := context.Background()
	proto, _ := vxproto.New(&FakeMainModule{})
	serveModule := func() {
		name := "test"
		args := map[string][]string{}
		files := map[string][]byte{
			"main.lua": []byte(`return 'success'`),
		}

		socket := proto.NewModule(name, "")
		if socket == nil {
			b.Fatal("Error with making new module in vxproto")
		}

		if !proto.AddModule(socket) {
			b.Fatal("Error with adding new module into vxproto")
		}

		state, err := lua.NewState(files)
		if err != nil {
			b.Fatal("Error with making new lua state: ", err.Error())
		}

		module, err := lua.NewModule(args, state, socket)
		if err != nil {
			b.Fatal("Error with making new lua module: ", err.Error())
		}

		module.Start()
		result := module.GetResult()
		if result != "success" {
			b.Fatal("Error with getting result: ", result)
		}

		module.Stop()
		if !proto.DelModule(socket) {
			b.Fatal("Error with deleting module from vxproto")
		}

		module.Close()
	}

	for i := 0; i < b.N; i++ {
		serveModule()
		// force using GC
		runtime.GC()
	}

	if err := proto.Close(ctx); err != nil {
		b.Fatal("Failed to close vxproto object: ", err.Error())
	}
}

func BenchmarkLuaLinkAgentToModule(b *testing.B) {
	var (
		cntAccepted      int64
		limitConnections = 50
		mainModule       = &FakeMainModule{}
		wg               sync.WaitGroup
	)
	ctx := context.Background()
	args := map[string][]string{}
	files := map[string][]byte{
		"main.lua": []byte(`
			local con, dis = 0, 0
			__api.set_recv_timeout(10)
			if __api.add_cbs({
				["data"] = function(src, data)
					return true
				end,
				["control"] = function(cmtype, data)
					if cmtype == "agent_connected" then
						con = con + 1
					end
					if cmtype == "agent_disconnected" then
						dis = dis + 1
					end
					return true
				end,
			}) == false then
				return "failed"
			end

			__api.await(-1)
			if con == dis and con ~= 0 then
				return tostring(con)
			else
				return "failed"
			end
		`),
	}

	proto_agent, _ := vxproto.New(mainModule)
	if proto_agent == nil {
		b.Fatal("Failed initialize VXProto object for agent")
	}
	proto_server, _ := vxproto.New(mainModule)
	if proto_server == nil {
		b.Fatal("Failed initialize VXProto object for server")
	}
	module, _ := initModule(files, args, "test", proto_server)

	wg.Add(1)
	go func() {
		defer wg.Done()
		cfg := &vxproto.ServerConfig{
			CommonConfig: &vxproto.CommonConfig{
				Host: netConnectionString,
			},
			API: vxproto.ServerAPIVersionsConfig{
				"v1": {
					Version:          "v1",
					ConnectionPolicy: vxproto.EndpointConnectionPolicyAllow,
				},
			},
		}
		logEntry := logrus.NewEntry(logrus.New())
		cvf := mainModule.GetConnectionValidatorFactory()
		if err := proto_server.Listen(ctx, cfg, cvf, logEntry); err != nil {
			if !strings.HasSuffix(err.Error(), "Server closed") {
				b.Error("Failed to listen socket", err.Error())
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		result := runModule(module)
		if result == "failed" {
			b.Error("Failed to process agent connections on the lua module")
		}
		if result_conns, err := strconv.Atoi(result); err != nil {
			b.Error("Failed to result from the lua module")
		} else {
			cntAccepted += int64(result_conns)
		}
	}()

	time.Sleep(3 * time.Second)
	b.ResetTimer()
	var cnt, cntRequested int64
	var wgc sync.WaitGroup
	for i := 0; i < b.N; i++ {
		atomic.AddInt64(&cnt, 1)
		wgc.Add(1)
		go func() {
			defer wgc.Done()
			clientCfg := &vxproto.ClientConfig{
				ID:    randString(16),
				Token: randString(20),
				ClientInitConfig: vxproto.ClientInitConfig{
					CommonConfig: &vxproto.CommonConfig{
						Host: netConnectionString,
					},
					Type:            "agent",
					ProtocolVersion: "v1",
				},
			}
			acv := mainModule.GetAgentConnectionValidator()
			if err := proto_agent.Connect(context.Background(), clientCfg, acv, &MockPackEncryptor{}); err != nil {
				// TODO: here need to separate connection errors
				err1 := strings.HasSuffix(err.Error(), "use of closed network connection")
				err2 := strings.HasSuffix(err.Error(), "connection reset by peer")
				err3 := strings.HasSuffix(err.Error(), "broken pipe")
				err4 := strings.HasSuffix(err.Error(), "protocol wrong type for socket")
				err5 := strings.HasSuffix(err.Error(), "failed to connect to the server: vxProto is already closed")
				err6 := strings.Contains(err.Error(), "websocket: close ")
				if !err1 && !err2 && !err3 && !err4 && !err5 && !err6 {
					b.Error("Failed to connect to module socket: ", err.Error())
				}
				atomic.AddInt64(&cnt, -1)
			}
		}()
		if int(atomic.LoadInt64(&cnt)) >= limitConnections {
			for int(atomic.LoadInt64(&cnt)) != proto_agent.GetAgentsCount() {
				runtime.Gosched()
			}
			cntRequested += atomic.LoadInt64(&cnt)
			proto_agent.Close(ctx)
			wgc.Wait()
			cnt = 0
		}
	}
	if int(atomic.LoadInt64(&cnt)) != 0 {
		for int(atomic.LoadInt64(&cnt)) != proto_agent.GetAgentsCount() {
			runtime.Gosched()
		}
		cntRequested += atomic.LoadInt64(&cnt)
		proto_agent.Close(ctx)
		wgc.Wait()
		cnt = 0
	}
	b.StopTimer()

	time.Sleep(time.Second)
	module.Close()
	if err := proto_server.Close(ctx); err != nil {
		b.Fatal("Failed to close vxproto object: ", err.Error())
	}
	wg.Wait()

	if cntRequested != cntAccepted {
		b.Fatal("Mismatch amount connections | requested: ", cntRequested, " | accepted: ", cntAccepted)
	}
}

func BenchmarkIMCSendData(b *testing.B) {
	ctx := context.Background()
	var wg sync.WaitGroup
	mname1 := []string{"test_module1", "sender1"}
	mname2 := []string{"test_module2", "sender2"}
	args1 := map[string][]string{
		"dst_module": mname2,
	}
	args2 := map[string][]string{
		"dst_module": mname1,
	}
	header := []byte(`
		local is_stop = false
		local syn, ack, fin, total = 0, 0, 0, 0
		local imc_token = __imc.make_token(__args["dst_module"][1], "")
		__api.set_recv_timeout(1000)

		local this_imc_token = __imc.get_token()
		if type(this_imc_token) ~= "string" or #this_imc_token ~= 40 then
			return "failed"
		end
		if __imc.is_exist(imc_token) == false then
			return "failed"
		end
		local modName, groupID, isExist = __imc.get_info(imc_token)
		if groupID ~= "" or modName ~= __args["dst_module"][1] or isExist ~= true then
			return "failed"
		end
		local function handle_data(src, data)
			if src ~= imc_token then return false end
			if data == "syn" and __api.send_data_to(src, "ack") then
				ack = ack + 1
				return true
			elseif data == "ack" then
				fin = fin + 1
				return true
			end
			return false
		end
	`)
	footer := []byte(`
		while not is_stop do
			__api.await(50)
		end

		return (syn == ack and ack == fin and fin == total) and "success" or "failed"
	`)
	files1 := map[string][]byte{
		"main.lua": append(append(header, []byte(`
			__api.add_cbs({
				["data"] = handle_data,
				["control"] = function(cmtype, data)
					if cmtype == "send" then
						total = total + 1
						if __api.send_data_to(imc_token, data) then
							syn = syn + 1
							return true
						end
						return false
					elseif cmtype == "stop" then
						is_stop = true
						return true
					end
					return false
				end,
			})
		`)...), footer...),
	}
	files2 := map[string][]byte{
		"main.lua": append(append(header, []byte(`
			__api.add_cbs({
				["data"] = handle_data,
				["control"] = function(cmtype, data)
					if cmtype == "send" then
						total = total + 1
						if __api.send_data_to(imc_token, data) then
							syn = syn + 1
							return true
						end
						return false
					elseif cmtype == "stop" then
						is_stop = true
						return true
					end
					return false
				end,
			})
		`)...), footer...),
	}

	proto, _ := vxproto.New(&FakeMainModule{})
	module1, _ := initModule(files1, args1, mname1[0], proto)
	module2, _ := initModule(files2, args2, mname2[0], proto)

	wg.Add(1)
	go func() {
		if result := runModule(module1); result != "success" {
			b.Errorf("error in %s module: '%s'", mname1[1], result)
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		if result := runModule(module2); result != "success" {
			b.Errorf("error in %s module: '%s'", mname2[1], result)
		}
		wg.Done()
	}()

	time.Sleep(2 * time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !module1.ControlMsg(ctx, "send", "syn") {
			b.Fatalf("error in %s module for sending packet", mname1[1])
		}
		if !module2.ControlMsg(ctx, "send", "syn") {
			b.Fatalf("error in %s module for sending packet", mname2[1])
		}
	}
	b.StopTimer()

	time.Sleep(3 * time.Second)
	if !module1.ControlMsg(ctx, "stop", "") {
		b.Fatalf("error in %s module for stopping module", mname1[1])
	}
	if !module2.ControlMsg(ctx, "stop", "") {
		b.Fatalf("error in %s module for stopping module", mname2[1])
	}

	wg.Wait()
	module1.Close()
	module2.Close()
	proto.Close(ctx)
}

func BenchmarkIMCAsyncSendData(b *testing.B) {
	ctx := context.Background()
	var wg sync.WaitGroup
	mname1 := []string{"test_module1", "sender"}
	mname2 := []string{"test_module2", "receiver"}
	args1 := map[string][]string{
		"dst_module": mname2,
	}
	args2 := map[string][]string{
		"dst_module": mname1,
	}
	header := []byte(`
		local is_stop = false
		local syn, ack, total = 0, 0, 0
		local imc_token = __imc.make_token(__args["dst_module"][1], "")
		__api.set_recv_timeout(1000)

		local this_imc_token = __imc.get_token()
		if type(this_imc_token) ~= "string" or #this_imc_token ~= 40 then
			return "failed"
		end
		if __imc.is_exist(imc_token) == false then
			return "failed"
		end
		local modName, groupID, isExist = __imc.get_info(imc_token)
		if groupID ~= "" or modName ~= __args["dst_module"][1] or isExist ~= true then
			return "failed"
		end
		local function handle_data(src, data)
			if src ~= imc_token then return false end
			local function send_result_callback(r)
				if r then
					ack = ack + 1
				end
			end
			if data == "syn" and __api.async_send_data_to(src, "ack", send_result_callback) then
				syn = syn + 1
				return true
			elseif data == "ack" then
				ack = ack + 1
				return true
			end
			return false
		end
	`)
	footer := []byte(`
		while not is_stop do
			__api.await(50)
		end

		return (syn == ack and ack == total) and "success" or "failed"
	`)
	files1 := map[string][]byte{
		"main.lua": append(append(header, []byte(`
			__api.add_cbs({
				["data"] = handle_data,
				["control"] = function(cmtype, data)
					if cmtype == "send" then
						total = total + 1
						return __api.async_send_data_to(imc_token, data, function(r)
							if r then
								syn = syn + 1
							end
						end)
					elseif cmtype == "stop" then
						is_stop = true
						return true
					end
					return false
				end,
			})
		`)...), footer...),
	}
	files2 := map[string][]byte{
		"main.lua": append(append(header, []byte(`
			__api.add_cbs({
				["data"] = handle_data,
				["control"] = function(cmtype, data)
					if cmtype == "send" then
						total = total + 1
						return true
					elseif cmtype == "stop" then
						is_stop = true
						return true
					end
					return false
				end,
			})
		`)...), footer...),
	}

	proto, _ := vxproto.New(&FakeMainModule{})
	module1, _ := initModule(files1, args1, mname1[0], proto)
	module2, _ := initModule(files2, args2, mname2[0], proto)

	wg.Add(1)
	go func() {
		if result := runModule(module1); result != "success" {
			b.Errorf("error in %s module: '%s'", mname1[1], result)
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		if result := runModule(module2); result != "success" {
			b.Errorf("error in %s module: '%s'", mname2[1], result)
		}
		wg.Done()
	}()

	time.Sleep(2 * time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !module1.ControlMsg(ctx, "send", "syn") {
			b.Fatalf("error in %s module for sending packet", mname1[1])
		}
		if !module2.ControlMsg(ctx, "send", "syn") {
			b.Fatalf("error in %s module for sending packet", mname2[1])
		}
	}
	b.StopTimer()

	time.Sleep(3 * time.Second)
	if !module1.ControlMsg(ctx, "stop", "") {
		b.Fatalf("error in %s module for stopping module", mname1[1])
	}
	if !module2.ControlMsg(ctx, "stop", "") {
		b.Fatalf("error in %s module for stopping module", mname2[1])
	}

	wg.Wait()
	module1.Close()
	module2.Close()
	proto.Close(ctx)
}

type MockPackEncryptor struct{}

func (e *MockPackEncryptor) Encrypt(data []byte) ([]byte, error) {
	return data, nil
}

func (e *MockPackEncryptor) Decrypt(data []byte) ([]byte, error) {
	return data, nil
}

func (e *MockPackEncryptor) Reset(config *protoagent.TunnelConfig) error {
	return nil
}
