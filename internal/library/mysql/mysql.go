package mysql

import (
	"fmt"
	"log"
	"os"
	"time"

	"business/internal/library/logger"
	"business/internal/library/oswrapper"

	"github.com/aidarkhanov/nanoid/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type MySQL struct {
	DB *gorm.DB
}

// New はGORMを使用してMySQLデータベースに接続するための新しいMySQLインスタンスを生成します。
func New(osw oswrapper.OsWapperInterface) (*MySQL, error) {
	newLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold: time.Second,
			LogLevel:      gormlogger.Info,
			Colorful:      true,
		},
	)
	// SQL非表示設定か確認する（オプション扱いのためエラーにはしない）
	isHiddenSql, err := osw.GetEnv("IS_HIDDEN_SQL")
	if err != nil {
		return nil, err
	}
	if isHiddenSql == "true" {
		newLogger = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	dbEnv, err := getdbEnv(osw, false)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbEnv.MySQLUer, dbEnv.MYSQL_PASSWORD, dbEnv.DB_HOST, dbEnv.DB_PORT, dbEnv.MYSQL_DATABASE)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}

	return &MySQL{DB: db}, nil
}

// NewTest はGORMを使用してMySQLデータベースに接続するための新しいMySQLインスタンスを生成します。
func NewTest(osw oswrapper.OsWapperInterface) (*MySQL, error) {
	newLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold: time.Second,
			LogLevel:      gormlogger.Info,
			Colorful:      true,
		},
	)
	// SQL非表示設定か確認する（オプション扱いのためエラーにはしない）
	isHiddenSql, err := osw.GetEnv("IS_HIDDEN_SQL")
	if err != nil {
		return nil, err
	}
	if isHiddenSql == "true" {
		newLogger = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	dbEnv, err := getdbEnv(osw, true)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbEnv.MySQLUer, dbEnv.MYSQL_PASSWORD, dbEnv.DB_HOST, dbEnv.DB_PORT, dbEnv.MYSQL_DATABASE)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}

	return &MySQL{DB: db}, nil
}

// CreateNewTestDB はランダムな名前でDBを作成し、そのインスタンスを返します。
// また、deferで削除を予約します。
func CreateNewTestDB() (*MySQL, func() error, error) {
	osw := oswrapper.New(nil)

	newLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold: time.Second,
			LogLevel:      gormlogger.Info,
			Colorful:      true,
		},
	)
	if os.Getenv("IS_HIDDEN_TEST_SQL") == "true" {
		newLogger = gormlogger.Default.LogMode(gormlogger.Silent)
	}
	randomDbName, err := generateUniqueID()
	if err != nil {
		return nil, nil, err
	}

	dbName := fmt.Sprintf("%s_test", randomDbName)
	if err := createMySQLDatabase(dbName, osw); err != nil {
		return nil, nil, fmt.Errorf("failed to create database: %w", err)
	}

	user, err := osw.GetEnv("MYSQL_USER")
	if err != nil {
		return nil, nil, err
	}
	password, err := osw.GetEnv("MYSQL_PASSWORD")
	if err != nil {
		return nil, nil, err
	}
	host, err := osw.GetEnv("DB_HOST")
	if err != nil {
		return nil, nil, err
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, dbName)

	// 作成したDBに接続
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 作成したDBを削除
	cleanUp := func() error {
		err := deleteMySQLDatabase(dbName, osw)
		return err
	}

	return &MySQL{DB: db}, cleanUp, nil
}

// createMySQLDatabase DBを作成する。
func createMySQLDatabase(dbName string, osw oswrapper.OsWapperInterface) (err error) {
	// rootユーザーでの接続情報を設定
	rootPassword, err := osw.GetEnv("MYSQL_PASSWORD")
	if err != nil {
		return err
	}
	host, err := osw.GetEnv("DB_HOST")
	if err != nil {
		return err
	}
	user, err := osw.GetEnv("MYSQL_USER")
	if err != nil {
		return err
	}

	dsn := fmt.Sprintf("root:%s@tcp(%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		rootPassword, host)

	// GormでrootユーザーとしてDBに接続
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database as root: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get generic database object: %w", err)
	}
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close db: %w", closeErr)
		}
	}()

	// テスト用のDBを作成
	if err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)).Error; err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// テストで使うユーザーに権限を付与
	if err = db.Exec(fmt.Sprintf("GRANT ALL ON `%s`.* TO '%s'@'%%'", dbName, user)).Error; err != nil {
		return fmt.Errorf("failed to grant privileges: %w", err)
	}

	return nil
}

func deleteMySQLDatabase(dbName string, osw oswrapper.OsWapperInterface) error {
	newLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold: time.Second,
			LogLevel:      gormlogger.Info,
			Colorful:      true,
		},
	)
	if os.Getenv("IS_HIDDEN_TEST_SQL") == "true" {
		newLogger = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	rootPassword, err := osw.GetEnv("MYSQL_PASSWORD")
	if err != nil {
		return err
	}
	host, err := osw.GetEnv("DB_HOST")
	if err != nil {
		return err
	}

	dsn := fmt.Sprintf("root:%s@tcp(%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		rootPassword, host)

	// GormでrootユーザーとしてDBに接続
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database as root: %w", err)
	}

	// データベース削除
	if err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)).Error; err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	return nil
}

// generateUniqueID はランダムな文字列を生成します。
func generateUniqueID() (string, error) {
	// カスタムアルファベットを定義 (特定の文字を除外)
	alphabet := "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz0123456789"

	// IDの長さを設定
	size := 21 // UUIDと同じ長さに設定

	// カスタムアルファベットとサイズを使用してIDを生成
	id, err := nanoid.GenerateString(alphabet, size)
	if err != nil {
		return "", err
	}

	return id, nil
}

// Transactional は新しいトランザクションを開始しインスタンスを返す関数です。
// 戻り値の関数はcleanUPとして受け取り、"defer cleanUP()"を直下の行に記述してください。
// log が nil の場合は logger.NewNop() を使用します。
func Transactional(db *gorm.DB, log logger.Interface) (*gorm.DB, func()) {
	if log == nil {
		log = logger.NewNop()
	}

	tx := db.Begin()
	if tx.Error != nil {
		panic("トランザクションの開始に失敗しました。")
	}

	// エラーハンドリングとロールバックを行うクロージャを返す。
	return tx, func() {
		// panicによるエラーの場合
		if r := recover(); r != nil {
			log.Error("panic recovered during transaction",
				logger.Any("panic", r))
			tx.Rollback()
			return
		}

		// tx.Errorが設定されている場合（明示的なエラー設定）
		if tx.Error != nil {
			log.Error("transaction error detected, rolling back",
				logger.Err(tx.Error))
			tx.Rollback()
			return
		}

		tx.Commit()
	}
}

type dbEnv struct {
	MySQLUer       string
	MYSQL_PASSWORD string
	DB_HOST        string
	DB_PORT        string
	MYSQL_DATABASE string
}

func getdbEnv(osw oswrapper.OsWapperInterface, isTest bool) (dbEnv, error) {
	user, err := osw.GetEnv("MYSQL_USER")
	if err != nil {
		return dbEnv{}, err
	}
	password, err := osw.GetEnv("MYSQL_PASSWORD")
	if err != nil {
		return dbEnv{}, err
	}
	host, err := osw.GetEnv("DB_HOST")
	if err != nil {
		return dbEnv{}, err
	}
	port, err := osw.GetEnv("DB_PORT")
	if err != nil {
		return dbEnv{}, err
	}
	var dbName string
	if isTest {
		dbName, err = osw.GetEnv("MYSQL_TEST_DATABASE")
		if err != nil {
			return dbEnv{}, err
		}
	} else {
		dbName, err = osw.GetEnv("MYSQL_DATABASE")
		if err != nil {
			return dbEnv{}, err
		}

	}

	return dbEnv{
		MySQLUer:       user,
		MYSQL_PASSWORD: password,
		DB_HOST:        host,
		DB_PORT:        port,
		MYSQL_DATABASE: dbName,
	}, nil
}
