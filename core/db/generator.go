package db

import (
	"gorm.io/gen"
	"gorm.io/gorm"
)

// GromGen 依据数据库表结构生成 Model 与 Query 代码。
//
// path 是输出根目录，模型写到 path/model，查询代码写到 path/query；
// 例如传 "." 会得到 ./model 与 ./query。
//
// 函数名 GromGen 是历史拼写错误（正确拼写为 GormGen），为保持向后兼容保留。
//
// 注意：底层 gorm.io/gen 在生成失败时直接 panic（Generator.Execute 没有
// error 返回值，内部 panic 而不是返回错误），所以本函数同样以 panic 报告失败，
// 调用方如有需要请自行 recover。
func GromGen(db *gorm.DB, path string) {
	// 创建query时，创建在指定数据库目录下
	g := gen.NewGenerator(gen.Config{
		OutPath:      path + "/query",
		ModelPkgPath: path + "/model",
		Mode:         gen.WithoutContext | gen.WithDefaultQuery, // generate mode
	})

	g.UseDB(db) // reuse your gorm db

	g.ApplyBasic(
		// g.GenerateModel("users"),
		// Generate structs from all tables of current database
		g.GenerateAllTable()...,
	)
	// Generate the code
	g.Execute()
}
