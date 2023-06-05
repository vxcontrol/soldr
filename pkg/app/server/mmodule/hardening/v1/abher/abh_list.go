package abher

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sync"

	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/models"
	"soldr/pkg/app/server/mmodule/hardening/cache"
	"soldr/pkg/filestorage"
	"soldr/pkg/vxproto"
)

type ABHList struct {
	Agents     map[string][][]byte
	Aggregates map[string][][]byte
	Browsers   map[string][][]byte
	Externals  map[string][][]byte
	mux        *sync.RWMutex
}

func NewABHList() *ABHList {
	return &ABHList{
		Agents:     make(map[string][][]byte),
		Aggregates: make(map[string][][]byte),
		Browsers:   make(map[string][][]byte),
		Externals:  make(map[string][][]byte),
		mux:        &sync.RWMutex{},
	}
}

var ErrABHNotFound = fmt.Errorf("ABH not found")

func (a *ABHList) Get(t vxproto.AgentType, abi string) ([][]byte, error) {
	a.mux.RLock()
	defer a.mux.RUnlock()

	var abhs [][]byte
	var ok bool
	switch t {
	case vxproto.VXAgent:
		abhs, ok = a.Agents[abi]
	case vxproto.Aggregate:
		abhs, ok = a.Aggregates[abi]
	case vxproto.Browser:
		abhs, ok = a.Browsers[abi]
	case vxproto.External:
		abhs, ok = a.Externals[abi]
	default:
		return nil, fmt.Errorf("unknown type %d passed", t)
	}
	if !ok {
		return nil, fmt.Errorf("%w for ABI %s", ErrABHNotFound, abi)
	}
	abhCopy := make([][]byte, len(abhs))
	copy(abhCopy, abhs)
	return abhCopy, nil
}

func (l *ABHList) UnmarshalJSON(data []byte) error {
	if l == nil {
		return fmt.Errorf("a nil container passed")
	}
	var objmap map[string]json.RawMessage
	if err := json.Unmarshal(data, &objmap); err != nil {
		return fmt.Errorf("failed to parse the passed data into an object map: %w", err)
	}
	const version = "v1"
	versionData, ok := objmap[version]
	if !ok {
		return fmt.Errorf("section \"%s\" has not been found in the passed json", version)
	}
	type fileABHListJSON struct {
		Agents     map[string]string `json:"agents"`
		Aggregates map[string]string `json:"aggregates"`
		Browsers   map[string]string `json:"browsers"`
		Externals  map[string]string `json:"externals"`
	}
	var dst fileABHListJSON
	if err := json.Unmarshal(versionData, &dst); err != nil {
		return err
	}
	copyMap := func(dst map[string][][]byte, src map[string]string) error {
		for abi, abhVal := range src {
			abh, err := hex.DecodeString(abhVal)
			if err != nil {
				return fmt.Errorf("failed to decode the ABH hex value %s: %w", abhVal, err)
			}
			dst[abi] = append(dst[abi], abh)
		}
		return nil
	}
	if err := copyMap(l.Agents, dst.Agents); err != nil {
		return err
	}
	if err := copyMap(l.Aggregates, dst.Aggregates); err != nil {
		return err
	}
	if err := copyMap(l.Browsers, dst.Browsers); err != nil {
		return err
	}
	if err := copyMap(l.Externals, dst.Externals); err != nil {
		return err
	}
	return nil
}

func getABHListFromFile(basePath string, dst *ABHList) cache.FetchDataFromFile {
	return func(ctx context.Context, connector filestorage.Reader) (interface{}, error) {
		filePath := path.Join(basePath, "hardening", "abh.json")
		file, err := connector.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read the file %s: %w", filePath, err)
		}
		if err := json.Unmarshal(file, dst); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the ABH list: %w", err)
		}
		return dst, nil
	}
}

func getFnABHListFromDB(ctx context.Context, connector *gorm.DB) (interface{}, error) {
	rows, err := connector.Raw("? UNION ?",
		connector.Table("binaries").Select("'agent', chksums").QueryExpr(),
		connector.Table("external_connections").Select("type, chksums").QueryExpr(),
	).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the ABH data: %w", err)
	}
	defer rows.Close()
	results := make([]*fetchABHListFromDBResult, 0)
	for rows.Next() {
		var connTypeStr string
		var chksumsJSON []byte
		if err := rows.Scan(&connTypeStr, &chksumsJSON); err != nil {
			return nil, fmt.Errorf("failed to read the returned row: %w", err)
		}
		var chksums map[string]models.BinaryChksum
		if err := json.Unmarshal(chksumsJSON, &chksums); err != nil {
			return nil, fmt.Errorf("failed to deserialize the received chksums JSON: %w", err)
		}
		connType, err := GetConnType(connTypeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to get the connection type: %w", err)
		}
		results = append(results, &fetchABHListFromDBResult{
			T:       connType,
			Chksums: chksums,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("an error occurred while reading the ABH rows: %w", err)
	}
	abhList, err := newABHListFromDBResult(results)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new ABH list: %w", err)
	}
	return abhList, nil
}

func GetConnType(t string) (vxproto.AgentType, error) {
	switch t {
	case "agent":
		return vxproto.VXAgent, nil
	case "aggregate":
		return vxproto.Aggregate, nil
	case "browser":
		return vxproto.Browser, nil
	case "external":
		return vxproto.External, nil
	default:
		return 0, fmt.Errorf("\"%s\" is an unknown connection type", t)
	}
}

type fetchABHListFromDBResult struct {
	T       vxproto.AgentType
	Chksums map[string]models.BinaryChksum
}

func newABHListFromDBResult(results []*fetchABHListFromDBResult) (*ABHList, error) {
	abhList := NewABHList()
	for _, r := range results {
		for filePath, c := range r.Chksums {
			var err error
			abi := filePath
			if r.T == vxproto.VXAgent {
				abi, err = ExtractABIFromDBBinaryPath(filePath)
				if err != nil {
					return nil, fmt.Errorf("failed to extract the ABI from the file path: %w", err)
				}
			}
			abh, err := hex.DecodeString(c.SHA256)
			if err != nil {
				return nil, fmt.Errorf("failed to decode the ABH hex value %s: %w", c.SHA256, err)
			}
			switch r.T {
			case vxproto.VXAgent:
				abhList.Agents[abi] = append(abhList.Agents[abi], abh)
			case vxproto.Aggregate:
				abhList.Aggregates[abi] = append(abhList.Aggregates[abi], abh)
			case vxproto.Browser:
				abhList.Browsers[abi] = append(abhList.Browsers[abi], abh)
			case vxproto.External:
				abhList.Externals[abi] = append(abhList.Externals[abi], abh)
			default:
				return nil, fmt.Errorf("unknown connection type %d passed", r.T)
			}
		}
	}
	return abhList, nil
}

// nolint: lll
var agentBinryIDRegexp = regexp.MustCompile(`(v)?[0-9]+\.[0-9]+(\.[0-9]+)?(\.[0-9]+)?(-[a-zA-Z0-9]+)?/(((linux|darwin|windows)/(amd64|386))|(aggregate|browser|external))`)

func ExtractABIFromDBBinaryPath(filePath string) (string, error) {
	binaryIDs := agentBinryIDRegexp.FindAllString(filePath, -1)
	if len(binaryIDs) != 1 {
		return "", fmt.Errorf(
			"the file path contains an unexpected number of agentBinaryIDs (%d): %v",
			len(binaryIDs),
			binaryIDs,
		)
	}
	return binaryIDs[0], nil
}
