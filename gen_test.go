package main

import (
	"fmt"
	"gcg2/gens/common"
	"gcg2/gens/orm"
	"os"
	"strings"
	"testing"
)

//go:generate go-bindata -o ./bindata.go -prefix "./template/" ./template/...
func TestGen(t *testing.T) {
	genutils.Asset = Asset
	genutils.SetValues(map[string]interface{}{
		"RootPath": "./",
	})
	orm.Gen("config.yaml", "init", "")
}

func TestProjectFolderName(t *testing.T) {
	s, _ := os.Getwd()
	folders := strings.Split(s, "\\")
	rs := folders[len(folders)-1]
	fmt.Println(rs)
}
