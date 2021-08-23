package main

import (
	"gcg2/gens/common"
	"gcg2/gens/orm"
)

func genORM(name, desc string) {
	genutils.SetValues(map[string]interface{}{
		"AppName":  appName,
		"AppPort":  appPort,
		"RootPath": rootPath,
	})
	orm.Gen(ormFile, name, tables)
}
