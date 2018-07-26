package jsondiff

import (
	"encoding/json"
	"fmt"

	"github.com/go-test/deep"
)

func Diff(a, b []byte) ([]string, error) {
	ma := map[string]interface{}{}
	if err := json.Unmarshal(a, &ma); err != nil {
		return nil, fmt.Errorf("unmarshal a: %v", err)
	}

	mb := map[string]interface{}{}
	if err := json.Unmarshal(b, &mb); err != nil {
		return nil, fmt.Errorf("unmarshal b: %v", err)
	}

	return deep.Equal(ma, mb), nil
}
