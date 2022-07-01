package orm

import (
	genutils "gcg2/gens/common"
	"gcg2/gens/funcs"
	utils "gcg2/gokits"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"strings"
)

const (
	PSQLTABLEQUERY  string = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'"
	MYSQLTABLEQUERY string = "SHOW TABLES"
	PSQLCOLUMNQUERY string = "SELECT COALESCE(col_description(a.attrelid, a.attnum),'') as comment, concat_ws ( '', T.typname, SUBSTRING ( format_type ( A.atttypid, A.atttypmod ) FROM '\\(.*\\)' ) ) AS TYPE, a.attname as field, CASE WHEN a.attnotnull='t' THEN 'NO' else 'YES' END as null, CASE WHEN p.contype = 'p' THEN 'YES' ELSE 'NO' END as key FROM pg_class c join pg_attribute a on a.attrelid = c.oid and a.attnum > 0\n\tLEFT JOIN pg_constraint p ON p.conrelid = c.oid AND a.attnum = ANY (p.conkey)\n\tjoin pg_type T on A.atttypid = T.oid \nwhere\n\tc.relname = '"
)

// loadDBMetaInfo 查询db元信息
func loadDBMetaInfo(tables string, dbInfo map[interface{}]interface{}) {
	var (
		db         *sqlx.DB
		err        error
		dbtables   []string
		tableMeta  interface{}
		tableMetas []interface{}
	)
	source := dbInfo["Source"].(string)
	driver := dbInfo["Driver"].(string)
	if dbInfo["IgnoreTablePrefix"] != nil {
		funcs.IgnoreTablePrefix = dbInfo["IgnoreTablePrefix"].(string)
	}
	db, err = sqlx.Connect(driver, source)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	//兼容psql
	if driver == "mysql" {
		if err = db.Select(&dbtables, MYSQLTABLEQUERY); err != nil {
			log.Fatalf("%s", err)
		}
	} else {
		if err = db.Select(&dbtables, PSQLTABLEQUERY); err != nil {
			log.Fatalf("%s", err)
		}
	}
	//获取项目路径
	projectName := getProjectFolderName() //dbInfo["ProjectName"].(string)
	if tables == "" && isInit {
		tables = dbInfo["Table"].(string)
	}
	genTablesArray := strings.Split(tables, ",")
	for _, dbt := range dbtables {
		// 只加载配置的表
		if tables == "*" || utils.IsInArray(genTablesArray, dbt) {
			tableMeta = loadTableMetaInfo(db, dbt, projectName, driver)
			model := tableMeta.(map[interface{}]interface{})
			//model层代码生成 数据对象生成
			genutils.GenFileWithTargetPath("model/model.go.tmpl", "model/"+strings.TrimLeft(model["TableName"].(string), funcs.IgnoreTablePrefix)+".go", tableMeta)
			tableMetas = append(tableMetas, tableMeta)
		}
	}
	dbInfo["TableMetas"] = tableMetas
}

// laodTableMetaInfo 查询表结构元信息
func loadTableMetaInfo(db *sqlx.DB, tableName, projectName string, driver string) interface{} {
	var (
		err            error
		primaryKey     = "id"
		primaryKeyType = "string"
		columnInfoList []*ColumnInfo
	)
	if driver == "mysql" {

		if err = db.Select(&columnInfoList, "SHOW FULL COLUMNS FROM `"+tableName+"`"); err != nil {
			log.Fatalf("%s", err)
		}
	} else {
		if err = db.Select(&columnInfoList, PSQLCOLUMNQUERY+tableName+"'"); err != nil {
			log.Fatalf("%s", err)
		}
	}
	for _, c := range columnInfoList {
		c.GoType = toGoType(c.Type, c.Null)
		c.Required = "required"
		if c.Null == "YES" {
			c.Required = ""
		}
		if c.Key == "PRI" || c.Key == "YES" {
			primaryKey = c.Field
			primaryKeyType = c.GoType
		}
	}
	return map[interface{}]interface{}{
		"ProjectName":    projectName,
		"TableName":      tableName,
		"PrimaryKey":     primaryKey,
		"PrimaryKeyType": primaryKeyType,
		"Columns":        columnInfoList,
	}
}
