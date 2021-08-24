package orm

import (
	"fmt"
	"gcg2/gens/common"
	sh "github.com/codeskyblue/go-sh"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var typeMap = [][]string{
	{"bit", "int", "*int"},
	{"int", "int", "*int"},
	{"int4", "int", "*int"},
	{"smallint", "int", "*int"},
	{"mediumint", "int", "*int"},
	{"tinyint", "byte", "*byte"},
	{"bigint", "int64", "*int64"},
	{"varchar", "string", "*string"},
	{"char", "string", "*string"},
	{"text", "string", "*string"},
	{"mediumtext", "string", "*string"},
	{"longtext", "string", "*string"},
	{"longblob", "string", "*string"},
	{"blob", "string", "*string"},
	{"set", "string", "*string"},
	{"json", "string", "*string"},
	{"tinytext", "string", "*string"},
	{"datetime", "time.Time", "*time.Time"},
	{"date", "time.Time", "*time.Time"},
	{"timestamp", "time.Time", "*time.Time"},
	{"decimal", "float64", "*float64"},
	{"float", "float64", "*float64"},
	{"double", "float64", "*float64"},
}

// ColumnInfo table column info
type ColumnInfo struct {
	Field      string
	Type       string
	Collation  *string
	Null       string
	Key        string
	Default    *string
	Extra      string
	Privileges string
	Comment    string
	GoType     string
	ThriftType string
	Required   string
}

var isInit = false

// Gen gen
func Gen(dbFile string, name string, tbs string) {
	if name == "init" {
		isInit = true
	}
	dirs := []string{
		"api",
		"model",
	}

	// mkdirs
	genutils.InitDirs(dirs)

	dbs := parseDBFile(dbFile)
	for _, v := range dbs {
		//生成service层代码
		db := v.(map[interface{}]interface{})
		//生成表service代码
		loadDBMetaInfo(tbs, db)
		sh.Command("gofmt", "-w", ".", sh.Dir("model")).Run()
		//生成api层表相关代码
		for _, tableMeta := range db["TableMetas"].([]interface{}) {
			model := tableMeta.(map[interface{}]interface{})
			genutils.GenFileWithTargetPath("api/gen_api.go.tmpl", "api/gen_"+model["TableName"].(string)+".go", tableMeta)
		}
		sh.Command("gofmt", "-w", ".", sh.Dir("api")).Run()
		if isInit {
			//生成api层公共代码
			genutils.GenFileWithTargetPath("api/api.go.tmpl", "api/api.go", nil)
			//生成main
			db["ProjectName"] = getProjectFolderName()
			genutils.GenFileWithTargetPath("main.go.tmpl", "main.go", db)
			sh.Command("gofmt", "-w", ".", sh.Dir("")).Run()
			if err := sh.Command("go", "mod", "init", getProjectFolderName(), sh.Dir("")).Run(); err != nil {
				fmt.Println(err.Error())
			}
			if err := sh.Command("go", "mod", "tidy", sh.Dir("")).Run(); err != nil {
				fmt.Println(err.Error())
			}
		}
	}
}

// parseDBFile 解析出db配置文件信息
func parseDBFile(dbFile string) []interface{} {
	var bs []byte
	var err error
	var dbs interface{}
	if bs, err = ioutil.ReadFile(dbFile); err != nil {
		log.Fatalf("%s", err)
	}
	if err = yaml.Unmarshal(bs, &dbs); err != nil {
		log.Fatalf("%s", err)
	}
	return dbs.([]interface{})
}

/*
转换数据库类型为go类型
*/
func toGoType(s, null string) string {
	for _, v := range typeMap {
		if strings.HasPrefix(s, v[0]) {
			if null == "YES" {
				return v[2]
			}
			return v[1]
		}
	}
	log.Fatalf("unsupport type %s", s)
	return ""
}

/**
 * 获取项目文件夹名称
 */
func getProjectFolderName() string {
	s, _ := os.Getwd()
	folders := strings.Split(s, "\\")
	return folders[len(folders)-1]
}
