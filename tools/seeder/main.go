package main

import (
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/oswrapper"
	"business/internal/library/secret"
	"business/tools/seeder/seeders"
	"context"
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"
)

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

	tx, cleanUP := mysql.Transactional(conn.DB, logger.NewNop())
	defer cleanUP()

	err = Seed(tx)
	if err != nil {
		tx.Error = err
		fmt.Printf("データ投入中にエラーが発生しました。\n")
		return
	}

	fmt.Printf("正常に終了しました。\n")
}

// CheckArgs はコマンドライン引数を確認する。
func CheckArgs() error {
	if len(os.Args) != 2 {
		return errors.New("期待している引数は1つです。引数を確認してください。")
	}

	if os.Args[1] != "dev" && os.Args[1] != "test" {
		return errors.New("第一引数が期待している語群は以下の通りです。\n1:dev\n2:test")
	}

	return nil
}

// Seed　はサンプルデータを投入する。
func Seed(tx *gorm.DB) error {
	var err error
	// ユーザーのシーダー（依存関係順に実行）
	// 0. マスタデータ
	if err = seeders.CreateUser(tx); err != nil {
		return err
	}
	// メール関連のシーダー（依存関係順に実行）
	// 1. マスタデータ
	if err = seeders.CreateKeywordGroupWordLink(tx); err != nil {
		return err
	}
	if err = seeders.CreateKeywordGroup(tx); err != nil {
		return err
	}
	if err = seeders.CreateKeyWord(tx); err != nil {
		return err
	}
	if err = seeders.CreatePositionGroup(tx); err != nil {
		return err
	}
	if err = seeders.CreatePositionWord(tx); err != nil {
		return err
	}
	if err = seeders.CreateWorkTypeGroup(tx); err != nil {
		return err
	}
	if err = seeders.CreateWorkTypeWord(tx); err != nil {
		return err
	}

	// 2. メールデータ（共通ヘッダー）
	if err = seeders.CreateEmail(tx); err != nil {
		return err
	}

	// 3. メール種別専用データ
	// CreateEmailProjectは現在使用しません（createEmail.goでEmailProjectをネストして作成するため）
	// if err = seeders.CreateEmailProject(tx); err != nil {
	// 	return err
	// }

	// EntryTimingは現在createEmail.goでEmailProjectのリレーションとして作成されます
	// if err = seeders.CreateEntryTiming(tx); err != nil {
	// 	return err
	// }

	if err = seeders.CreateEmailCandidate(tx); err != nil {
		return err
	}

	// 4. 関連テーブル
	if err = seeders.CreateEmailKeywordGroup(tx); err != nil {
		return err
	}
	if err = seeders.CreateEmailPositionGroup(tx); err != nil {
		return err
	}
	if err = seeders.CreateEmailWorkTypeGroup(tx); err != nil {
		return err
	}

	return nil
}
