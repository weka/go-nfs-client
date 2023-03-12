// Copyright © 2017 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
	"go-nfs-client/nfs"
	"go-nfs-client/nfs/rpc"
	"go-nfs-client/nfs/util"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	util.DefaultLogger.SetDebug(true)
	if len(os.Args) != 2 {
		util.Infof("%s <host>:<target path> ", os.Args[0])
		os.Exit(-1)
	}

	b := strings.Split(os.Args[1], ":")

	host := b[0]
	target := b[1]

	util.Infof("host=%s target=%s\n", host, target)

	log.Print("Attempting to dial PortMapper")
	mount, err := nfs.DialMount(host, false)
	if err != nil {
		log.Fatalf("unable to dial MOUNT service: %v", err)
	}
	defer func() {
		if err := mount.Close(); err != nil {
			log.Fatal("Failed to close RPCBIND")
		}
	}()

	auth := rpc.NewAuthUnix(host, 0, 0)

	v, err := mount.Mount(target, auth.Auth(), false)
	if err != nil {
		log.Fatalf("unable to mount volume: %v", err)
	}
	defer func() {
		if err := v.Close(); err != nil {
			log.Fatal("Failed to close MOUNT")
		}
	}()

	u, _ := uuid.NewUUID()
	testFileName := u.String()

	if err := testFileRW(v, testFileName, 1024); err != nil {
		log.Fatalf("Failed to create file for testing")
	}

	if err := v.Remove(testFileName); err != nil {
		log.Fatalf("Failed to delete file")
	}
	log.Print("All GOOD")
}

func testFileRW(v *nfs.Target, name string, filesize uint64) error {

	// create a temp file
	f, err := os.Open("/dev/urandom")
	if err != nil {
		util.Errorf("error openning random: %s", err.Error())
		return err
	}

	wr, err := v.OpenFile(name, 0777)
	if err != nil {
		util.Errorf("write fail: %s", err.Error())
		return err
	}

	// calculate the sha
	h := sha256.New()
	t := io.TeeReader(f, h)

	// Copy filesize
	n, err := io.CopyN(wr, t, int64(filesize))
	if err != nil {
		util.Errorf("error copying: n=%d, %s", n, err.Error())
		return err
	}
	expectedSum := h.Sum(nil)

	if err = wr.Close(); err != nil {
		util.Errorf("error committing: %s", err.Error())
		return err
	}

	//
	// get the file we wrote and calc the sum
	rdr, err := v.Open(name)
	if err != nil {
		util.Errorf("read error: %v", err)
		return err
	}

	h = sha256.New()
	t = io.TeeReader(rdr, h)

	_, err = io.ReadAll(t)
	if err != nil {
		util.Errorf("readall error: %v", err)
		return err
	}
	actualSum := h.Sum(nil)

	if bytes.Compare(actualSum, expectedSum) != 0 {
		log.Fatalf("sums didn't match. actual=%x expected=%s", actualSum, expectedSum) //  Got=0%x expected=0%x", string(buf), testdata)
	}

	log.Printf("Sums match %x %x", actualSum, expectedSum)
	return nil
}

func ls(v *nfs.Target, path string) ([]*nfs.EntryPlus, error) {
	dirs, err := v.ReadDirPlus(path)
	if err != nil {
		return nil, fmt.Errorf("readdir error: %s", err.Error())
	}

	util.Infof("dirs:")
	for _, dir := range dirs {
		util.Infof("\t%s\t%d:%d\t0%o", dir.FileName, dir.Attr.Attr.UID, dir.Attr.Attr.GID, dir.Attr.Attr.Mode)
	}

	return dirs, nil
}
