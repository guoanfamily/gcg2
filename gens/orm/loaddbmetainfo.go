package orm

import (
	genutils "gcg2/gens/common"
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
	PSQLCOLUMNQUERY string = "SELECT A\n\t.attname AS field,\n\tconcat_ws ( '', T.typname, SUBSTRING ( format_type ( A.atttypid, A.atttypmod ) FROM '\\(.*\\)' ) ) AS TYPE,\n\te.is_nullable AS NULL,\n\te.is_identity AS KEY,\n\td.description AS COMMENT \nFROM\n\tpg_class C,\n\tpg_attribute A,\n\tpg_type T,\n\tpg_description d,\n\tinformation_schema.COLUMNS e \nWHERE\n\tC.relname = e.TABLE_NAME \n\tAND e.table_schema = 'public' \n\tAND e.COLUMN_NAME = A.attname \n\tAND A.attnum > 0 \n\tAND A.attrelid = C.oid \n\tAND A.atttypid = T.oid \n\tAND d.objoid = A.attrelid \n\tAND d.objsubid = A.attnum \n\tAND C.relname = '"
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
			genutils.GenFileWithTargetPath("model/model.go.tmpl", "model/"+model["TableName"].(string)+".go", tableMeta)
			tableMetas = append(tableMetas, tableMeta)
		}
	}
	dbInfo["TableMetas"] = tableMetas
}

// laodTableMetaInfo 查询表结构元信息
func loadTableMetaInfo(db *sqlx.DB, tableName, projectName string, driver string) interface{} {
	var (
		err               error
		primaryKey        = "id"
		primaryKeyType    = "int"
		primaryKeyDefault = "0"
		columnInfoList    []*ColumnInfo
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
			if primaryKeyType != "int" {
				primaryKeyDefault = "\"\""
			}
		}
	}
	return map[interface{}]interface{}{
		"ProjectName":       projectName,
		"TableName":         tableName,
		"PrimaryKey":        primaryKey,
		"PrimaryKeyType":    primaryKeyType,
		"PrimaryKeyDefault": primaryKeyDefault,
		"Columns":           columnInfoList,
	}
}
