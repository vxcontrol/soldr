//go:build windows
// +build windows

package system

import (
	"fmt"
	"os/user"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows/registry"

	"soldr/internal/protoagent"
)

type user_info_2 struct {
	Usri2_name           *uint16
	Usri2_password       *uint16
	Usri2_password_age   uint32
	Usri2_priv           uint32
	Usri2_home_dir       *uint16
	Usri2_comment        *uint16
	Usri2_flags          uint32
	Usri2_script_path    *uint16
	Usri2_auth_flags     uint32
	Usri2_full_name      *uint16
	Usri2_usr_comment    *uint16
	Usri2_parms          *uint16
	Usri2_workstations   *uint16
	Usri2_last_logon     uint32
	Usri2_last_logoff    uint32
	Usri2_acct_expires   uint32
	Usri2_max_storage    uint32
	Usri2_units_per_week uint32
	Usri2_logon_hours    uintptr
	Usri2_bad_pw_count   uint32
	Usri2_num_logons     uint32
	Usri2_logon_server   *uint16
	Usri2_country_code   uint32
	Usri2_code_page      uint32
}

func getOSName() string {
	return "Microsoft Windows"
}

func getOSVer() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.QUERY_VALUE)
	if err != nil {
		return "0.0"
	}
	defer k.Close()

	_cv, _, err := k.GetStringValue("CurrentVersion")
	if err != nil {
		return "0.0"
	}

	cv, err := strconv.ParseFloat(_cv, 64)
	if err != nil {
		return "0.0"
	}

	if cv == 6.3 {
		_cb, _, err := k.GetStringValue("CurrentBuild")
		if err != nil {
			return "0.0"
		}
		cb, err := strconv.ParseInt(_cb, 10, 64)
		if err != nil {
			return "0.0"
		}
		if cb > 9800 {
			return "10.0"
		}
	}
	return fmt.Sprintf("%.1f", cv)
}

func getUsersInformation() []*protoagent.Information_User {
	const (
		USER_FILTER_NORMAL_ACCOUNT = 0x0002
		USER_MAX_PREFERRED_LENGTH  = 0xFFFFFFFF

		NET_API_STATUS_NERR_Success = 0

		USER_UF_ACCOUNTDISABLE = 2
		USER_UF_LOCKOUT        = 16
	)

	var (
		modNetapi32         = syscall.NewLazyDLL("netapi32.dll")
		usrNetUserEnum      = modNetapi32.NewProc("NetUserEnum")
		usrNetApiBufferFree = modNetapi32.NewProc("NetApiBufferFree")

		dataPointer  uintptr
		resumeHandle uintptr
		entriesRead  uint32
		entriesTotal uint32
		sizeTest     user_info_2
		users        = make([]*protoagent.Information_User, 0)
	)

	ret, _, _ := usrNetUserEnum.Call(
		uintptr(0),
		uintptr(uint32(2)),
		uintptr(uint32(USER_FILTER_NORMAL_ACCOUNT)),
		uintptr(unsafe.Pointer(&dataPointer)),
		uintptr(uint32(USER_MAX_PREFERRED_LENGTH)),
		uintptr(unsafe.Pointer(&entriesRead)),
		uintptr(unsafe.Pointer(&entriesTotal)),
		uintptr(unsafe.Pointer(&resumeHandle)),
	)
	if ret != NET_API_STATUS_NERR_Success {
		return users
	} else if dataPointer == uintptr(0) {
		return users
	}

	iter := dataPointer
	for i := uint32(0); i < entriesRead; i++ {
		data := (*user_info_2)(unsafe.Pointer(iter))
		name := utf16ToString(data.Usri2_name)
		item := &protoagent.Information_User{
			Name:   &name,
			Groups: []string{},
		}

		for {
			if (data.Usri2_flags & USER_UF_ACCOUNTDISABLE) == USER_UF_ACCOUNTDISABLE {
				break
			}
			if (data.Usri2_flags & USER_UF_LOCKOUT) == USER_UF_LOCKOUT {
				break
			}

			us, err := user.Lookup(name)
			if err != nil {
				break
			}
			gs, err := us.GroupIds()
			if err != nil {
				break
			}
			for _, g := range gs {
				ug, err := user.LookupGroupId(g)
				if err != nil {
					continue
				}
				if ug.Name != "" && ug.Name != "None" {
					item.Groups = append(item.Groups, ug.Name)
				}
			}

			users = append(users, item)
			break
		}
		iter = uintptr(unsafe.Pointer(iter + unsafe.Sizeof(sizeTest)))
	}
	usrNetApiBufferFree.Call(dataPointer)

	return users
}

func utf16ToString(p *uint16) string {
	return syscall.UTF16ToString((*[4096]uint16)(unsafe.Pointer(p))[:])
}

func getMachineID() (string, error) {
	sp, _ := getSystemProduct()
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Cryptography`,
		registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return sp, err
	}
	defer k.Close()

	s, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return sp, err
	}
	return s + ":" + sp, nil
}

func getSystemProduct() (string, error) {
	var err error
	var classID *ole.GUID

	IID_ISWbemLocator, _ := ole.CLSIDFromString("{76A6415B-CB41-11D1-8B02-00600806D9B6}")

	err = ole.CoInitialize(0)
	if err != nil {
		return "", fmt.Errorf("OLE initialize error: %v", err)
	}
	defer ole.CoUninitialize()

	classID, err = ole.ClassIDFrom("WbemScripting.SWbemLocator")
	if err != nil {
		return "", fmt.Errorf("CreateObject WbemScripting.SWbemLocator returned with %v", err)
	}

	comserver, err := ole.CreateInstance(classID, ole.IID_IUnknown)
	if err != nil {
		return "", fmt.Errorf("CreateInstance WbemScripting.SWbemLocator returned with %v", err)
	}
	if comserver == nil {
		return "", fmt.Errorf("CreateObject WbemScripting.SWbemLocator not an object")
	}
	defer comserver.Release()

	dispatch, err := comserver.QueryInterface(IID_ISWbemLocator)
	if err != nil {
		return "", fmt.Errorf("context.iunknown.QueryInterface returned with %v", err)
	}
	defer dispatch.Release()

	wbemServices, err := dispatch.CallMethod("ConnectServer")
	if err != nil {
		return "", fmt.Errorf("ConnectServer failed with %v", err)
	}
	defer wbemServices.Clear()

	query := "SELECT * FROM Win32_ComputerSystemProduct"
	objectset, err := wbemServices.ToIDispatch().CallMethod("ExecQuery", query)
	if err != nil {
		return "", fmt.Errorf("ExecQuery failed with %v", err)
	}
	defer objectset.Clear()

	enum_property, err := objectset.ToIDispatch().GetProperty("_NewEnum")
	if err != nil {
		return "", fmt.Errorf("Get _NewEnum property failed with %v", err)
	}
	defer enum_property.Clear()

	enum, err := enum_property.ToIUnknown().IEnumVARIANT(ole.IID_IEnumVariant)
	if err != nil {
		return "", fmt.Errorf("IEnumVARIANT() returned with %v", err)
	}
	if enum == nil {
		return "", fmt.Errorf("Enum is nil")
	}
	defer enum.Release()

	for tmp, length, err := enum.Next(1); length > 0; tmp, length, err = enum.Next(1) {
		if err != nil {
			return "", fmt.Errorf("Next() returned with %v", err)
		}
		tmp_dispatch := tmp.ToIDispatch()
		defer tmp_dispatch.Release()

		props, err := tmp_dispatch.GetProperty("Properties_")
		if err != nil {
			return "", fmt.Errorf("Get Properties_ property failed with %v", err)
		}
		defer props.Clear()

		props_enum_property, err := props.ToIDispatch().GetProperty("_NewEnum")
		if err != nil {
			return "", fmt.Errorf("Get _NewEnum property failed with %v", err)
		}
		defer props_enum_property.Clear()

		props_enum, err := props_enum_property.ToIUnknown().IEnumVARIANT(ole.IID_IEnumVariant)
		if err != nil {
			return "", fmt.Errorf("IEnumVARIANT failed with %v", err)
		}
		defer props_enum.Release()

		class_variant, err := tmp_dispatch.GetProperty("UUID")
		if err != nil {
			return "", fmt.Errorf("Get UUID property failed with %v", err)
		}
		defer class_variant.Clear()

		class_name := class_variant.ToString()
		return class_name, nil
	}

	return "", fmt.Errorf("not found")
}
