// Copyright © 2017 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package xdr

import (
	"github.com/rasky/go-xdr/xdr2"
	"io"
)

func Write(w io.Writer, val interface{}) error {
	_, err := xdr.Marshal(w, val)
	return err
}
