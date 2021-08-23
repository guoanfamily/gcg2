package orm

import (
	"database/sql"
	"fmt"
	"gcg2/gens/common"
	"gcg2/gens/funcs"
	"gcg2/gokits"
	sh "github.com/codeskyblue/go-sh"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var typeMap = [][]string{
	{"bit", "int", "*int"},
	{"int", "int", "*int"},
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
var isEs = false

// Gen gen
func Gen(dbFile string, name string, tbs string) {
	if name == "init" {
		isInit = true
	}
	dirs := []string{
		"api",
		"model",
	}
	//model := map[string]interface{}{
	//	"RootPath": genutils.Values["RootPath"],
	//}
	funcs.FuncMap["setDefault"] = SetDefault
	funcs.FuncMap["getTableFieldNames"] = GetTableFieldNames
	funcs.FuncMap["getTableFieldCounts"] = GetTableFieldCounts
	funcs.FuncMap["getObjColumn"] = GetObjColumn
	funcs.FuncMap["getUpdateColumn"] = GetUpdateColumn
	funcs.FuncMap["getInsertColumn"] = GetInsertColumn

	// mkdirs
	genutils.InitDirs(dirs)

	dbs := parseDBFile(dbFile)
	for _, v := range dbs {
		//生成service层代码
		db := v.(map[interface{}]interface{})
		//生成表service代码
		loadDBMetaInfo(tbs, db)
		// log.Debugf("%v", db)
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
			sh.Command("gofmt", "-w", ".", sh.Dir("router")).Run()
		}
	}
	//model["DB"] = dbs

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

// loadDBMetaInfo 查询db元信息
func loadDBMetaInfo(tables string, dbInfo map[interface{}]interface{}) {
	var (
		db         *sql.DB
		rows       *sql.Rows
		err        error
		tableMeta  interface{}
		tableMetas []interface{}
	)
	source := dbInfo["Source"].(string)
	driver := dbInfo["Driver"].(string)
	if db, err = sql.Open(driver, source); err != nil {
		log.Fatalf("%s", err)
	}
	defer db.Close()
	if rows, err = db.Query("SHOW TABLES"); err != nil {
		log.Fatalf("%s", err)
	}
	dbName := dbInfo["Name"].(string)
	//获取项目路径
	projectName := getProjectFolderName() //dbInfo["ProjectName"].(string)
	if tables == "" && isInit {
		tables = dbInfo["Table"].(string)
	}
	genTablesArray := strings.Split(tables, ",")
	for rows.Next() {
		var rowName string
		if err = rows.Scan(&rowName); err != nil {
			log.Fatalf("%s", err)
		}
		// 只加载配置的表
		if tables == "*" || utils.IsInArray(genTablesArray, rowName) {
			tableMeta = loadTableMetaInfo(db, rowName, dbName, projectName)
			model := tableMeta.(map[interface{}]interface{})
			//model层代码生成 数据对象生成
			genutils.GenFileWithTargetPath("model/model.go.tmpl", "model/"+model["TableName"].(string)+".go", tableMeta)
			tableMetas = append(tableMetas, tableMeta)
		}
	}
	dbInfo["TableMetas"] = tableMetas
}

// laodTableMetaInfo 查询表结构元信息
func loadTableMetaInfo(db *sql.DB, tableName, dbName string, projectName string) interface{} {
	var (
		rows              *sql.Rows
		err               error
		primaryKey        = "id"
		primaryKeyType    = "int"
		primaryKeyDefault = "0"
		primaryKeyExtra   = ""
		autoIncrement     = false
		columnInfoList    []*ColumnInfo
	)
	if rows, err = db.Query("SHOW FULL COLUMNS FROM `" + tableName + "`"); err != nil {
		log.Fatalf("%s", err)
	}

	for rows.Next() {
		c := new(ColumnInfo)
		if err = rows.Scan(&c.Field, &c.Type, &c.Collation, &c.Null, &c.Key, &c.Default, &c.Extra, &c.Privileges, &c.Comment); err != nil {
			log.Fatalf("%s", err)
		}
		c.GoType = toGoType(c.Type, c.Null)
		c.Required = "required"
		if c.Null == "YES" {
			c.Required = ""
		}
		// log.Debugf("%v", c)
		columnInfoList = append(columnInfoList, c)
		if c.Key == "PRI" {
			primaryKey = c.Field
			primaryKeyType = c.GoType
			primaryKeyExtra = c.Extra
			if primaryKeyType != "int" {
				primaryKeyDefault = "\"\""
			}
			if c.Extra == "auto_increment" {
				autoIncrement = true
			}
		}
	}
	return map[interface{}]interface{}{
		"ProjectName":       projectName,
		"DBName":            dbName,
		"TableName":         tableName,
		"PrimaryKey":        primaryKey,
		"PrimaryKeyType":    primaryKeyType,
		"PrimaryKeyDefault": primaryKeyDefault,
		"PrimaryKeyExtra":   primaryKeyExtra,
		"AutoIncrement":     autoIncrement,
		"Columns":           columnInfoList,
	}
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

func SetDefault(ci *ColumnInfo) string {
	if ci.Default == nil {
		return ""
	}
	if *ci.Default == "CURRENT_TIMESTAMP" {
		return ""
	}
	if strings.Contains(ci.GoType, "string") && *ci.Default == "" {
		return ""
	}
	if (strings.Contains(ci.GoType, "int") || strings.Contains(ci.GoType, "byte") || strings.Contains(ci.GoType, "float")) && !strings.Contains(ci.GoType, "*") && *ci.Default == "0" {
		return ""
	}
	r := funcs.UpperCamel(ci.Field) + ": "
	if ci.Null == "YES" {
		if strings.Contains(ci.GoType, "Time") {
			r += "utils.ToTime(\"2006-01-02 15:04:05\", \""
		} else {
			r += "utils." + funcs.CapitalizeFirst(ci.GoType[1:]) + "("
		}

	}
	if strings.Contains(ci.GoType, "string") {
		r += "\""
	}
	r += *ci.Default
	if strings.Contains(ci.GoType, "string") {
		r += "\""
	}
	if ci.Null == "YES" {
		if strings.Contains(ci.GoType, "Time") {
			r += "\""
		}
		r += ")"
	}
	r += ","
	return r
}

func GetTableFieldNames(args []*ColumnInfo) string {
	names := []string{}
	for _, a := range args {
		//if a.Field == "id"{
		//	continue
		//}
		names = append(names, fmt.Sprintf("`%s`", a.Field))
	}
	return strings.Join(names, ", ")
}

func GetTableFieldCounts(args []*ColumnInfo) string {
	names := []string{}
	for _, a := range args {
		if a.Field == "id" || a.Field == "updatetime" || a.Field == "deleted" {
			continue
		}
		if a.Field == "createtime" {
			names = append(names, "now()")
		} else {
			names = append(names, "?")
		}
	}
	return strings.Join(names, ", ")
}

func GetObjColumn(args []*ColumnInfo) string {
	names := []string{}
	for _, a := range args {
		if a.Field == "id" || a.Field == "createtime" || a.Field == "updatetime" || a.Field == "deleted" {
			continue
		}
		names = append(names, fmt.Sprintf("obj.%s", strFirstToUpper(a.Field)))
	}
	return strings.Join(names, ", ")
}

func GetUpdateColumn(args []*ColumnInfo) string {
	names := []string{}
	for _, a := range args {
		if a.Field == "id" || a.Field == "createtime" || a.Field == "updatetime" || a.Field == "deleted" {
			continue
		}
		names = append(names, fmt.Sprintf("%s=?", a.Field))
	}
	return strings.Join(names, ", ")
}

func GetInsertColumn(args []*ColumnInfo) string {
	names := []string{}
	for _, a := range args {
		if a.Field == "id" || a.Field == "updatetime" || a.Field == "deleted" {
			continue
		}
		names = append(names, fmt.Sprintf("`%s`", a.Field))
	}
	return strings.Join(names, ", ")
}

/**
 * 字符串首字母转化为大写 ios_bbbbbbbb -> IosBbbbbbbbb
 */
func strFirstToUpper(str string) string {
	temp := strings.Split(str, "_")
	var upperStr string
	for y := 0; y < len(temp); y++ {
		vv := []rune(temp[y])
		for i := 0; i < len(vv); i++ {
			if i == 0 && vv[i] > 96 {
				vv[i] -= 32
				upperStr += string(vv[i]) // + string(vv[i+1])
			} else {
				upperStr += string(vv[i])
			}
		}
	}
	return upperStr
}

/**
 * 获取项目文件夹名称
 */
func getProjectFolderName() string {
	s, _ := os.Getwd()
	folders := strings.Split(s, "\\")
	return folders[len(folders)-1]
}
