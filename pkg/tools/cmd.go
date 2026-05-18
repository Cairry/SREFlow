package tools

import "github.com/rs/xid"

func RandId() string {
	return xid.New().String()
}
