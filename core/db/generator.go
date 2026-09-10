package db

import (
	"gorm.io/gen"
	"gorm.io/gorm"
)

// Gorm 生成
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
