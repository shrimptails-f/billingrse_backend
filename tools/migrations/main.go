package main

import (
	"business/internal/library/mysql"
	"business/internal/library/oswrapper"
	"business/internal/library/secret"
	"business/tools/migrations/model"
	"context"
	"errors"
	"fmt"
	"os"
)

// main は引数からテーブル作成を行います
// 引数:
// - arg1: 接続環境の指定。期待する語群:dev or test
// - arg1: テーブルの作成か、削除の指定 期待する語群:create or drop
func main() {
	// コマンドラインのバリデーション
	err := CheckArgs()
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	ctx := context.Background()
	secretClient, err := secret.New(ctx)
	if err != nil {
		panic("シークレットクライアント初期化に失敗しました: " + err.Error())
	}
	osw := oswrapper.New(secretClient)

	var conn *mysql.MySQL
	if os.Args[1] == "dev" {
		conn, err = mysql.New(osw)
	} else if os.Args[1] == "test" {
		conn, err = mysql.NewTest(osw)
	}
	if err != nil {
		panic(err)
	}

	// connがnilでないことを確認
	if conn == nil || conn.DB == nil {
		panic("データベース接続が初期化されていません。")
	}

	if os.Args[2] == "create" {
		err = conn.DB.AutoMigrate(CreateArrayMigrationSlice()...)
	} else if os.Args[2] == "drop" {
		err = conn.DB.Migrator().DropTable(CreateArrayMigrationSlice()...)
	}
	if err != nil {
		fmt.Printf("エラーが発生しました。:%v \n", err)
		return
	}

	fmt.Printf("正常に終了しました。\n")
}

// CheckArgs はコマンドライン引数を確認する。
func CheckArgs() error {
	if len(os.Args) != 3 {
		return errors.New("期待している引数は2つです。引数を確認してください。")
	}

	if os.Args[1] != "dev" && os.Args[1] != "test" {
		return errors.New("第一引数が期待している語群は以下の通りです。\n1:dev\n2:test")
	}

	if os.Args[2] != "create" && os.Args[2] != "drop" {
		return errors.New("第二引数が期待している語群は以下の通りです。\n1:create\n2:drop")
	}

	return nil
}

// CreateArrayMigrationSlice はマイグレーション用の構造体が入った配列を返す。
func CreateArrayMigrationSlice() []interface{} {
	return []interface{}{
		model.User{},
		model.EmailVerificationToken{},
		model.EmailCredential{},
	}
}
