package utils

import (
	"fmt"
	"testing"
)

func TestIsAgentIDValid(t *testing.T) {
	cases := []struct {
		Name   string
		InID   string
		OutErr error
	}{
		{
			Name:   "normal",
			InID:   "a7b65d4cd3ef21a1b23456c78d90e66f",
			OutErr: nil,
		},
		{
			Name:   "err: contains 'z'",
			InID:   "z7b65d4cd3ef21a1b23456c78d90e66f",
			OutErr: fmt.Errorf("passed agent ID is not valid"),
		},
		{
			Name:   "err: contains '*'",
			InID:   "*7b65d4cd3ef21a1b23456c78d90e66f",
			OutErr: fmt.Errorf("passed agent ID is not valid"),
		},
		{
			Name:   "err: not long enough",
			InID:   "a7b65d4cd3ef21a1b23456c7",
			OutErr: fmt.Errorf("passed agent ID is not valid"),
		},
		{
			Name:   "err: too long enough",
			InID:   "aa7b65d4cd3ef21a1b23456c78d90e66f",
			OutErr: fmt.Errorf("passed agent ID is not valid"),
		},
		{
			Name: "err: multiline",
			InID: `a7b65d4cd3ef21a1b23456c78d90e66f
a7b65d4cd3ef21a1b23456c78d90e66f`,
			OutErr: fmt.Errorf("passed agent ID is not valid"),
		},
		{
			Name:   "err: empty",
			InID:   "",
			OutErr: fmt.Errorf("passed agent ID is not valid"),
		},
	}

	for i, tc := range cases {
		i, tc := i, tc
		t.Run(fmt.Sprintf("test #%d: %s", i, tc.Name), func(t *testing.T) {
			t.Logf("passed ID: %s", tc.InID)
			actualErr := IsAgentIDValid(tc.InID)
			if actualErr == nil && tc.OutErr == nil {
				return
			}
			if actualErr == nil {
				t.Errorf("expected err %v, but got nil", tc.OutErr)
				return
			}
			if tc.OutErr == nil {
				t.Errorf("got err %v, but expected nil", actualErr)
				return
			}
			if actualErr.Error() != tc.OutErr.Error() {
				t.Errorf("expected err %v, got %v", tc.InID, actualErr)
				return
			}
		})
	}
}
