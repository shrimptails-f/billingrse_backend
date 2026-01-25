package fixture

// import (
// 	"business/test/modeltest"
// 	"business/tools/migrations/model"
// 	pkgmysql "business/internal/library/mysql"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// type Fixture struct {
// 	Users                        []*model.User
// 	UserJoinedChatRoom           []*model.UserJoinedChatRoom
// 	UserJoinedIndividualChatRoom []*model.UserJoinedIndividualChatRoom
// 	ChatRooms                    []*model.ChatRoom
// 	ChatMessages                 []*model.ChatMessage
// }

// // Setup　はfの内容に基づいてテスト用にDBをセットアップします。
// func (f *Fixture) Setup(t *testing.T, conn *pkgmysql.MySQL) {
// 	if len(f.Users) != 0 {
// 		err := conn.DB.Create(f.Users).Error
// 		assert.Empty(t, err)
// 	}
// 	if len(f.UserJoinedChatRoom) != 0 {
// 		err := conn.DB.Create(f.UserJoinedChatRoom).Error
// 		assert.Empty(t, err)
// 	}
// 	if len(f.UserJoinedIndividualChatRoom) != 0 {
// 		err := conn.DB.Create(f.UserJoinedIndividualChatRoom).Error
// 		assert.Empty(t, err)
// 	}
// 	if len(f.ChatRooms) != 0 {
// 		err := conn.DB.Create(f.ChatRooms).Error
// 		assert.Empty(t, err)
// 	}
// 	if len(f.ChatMessages) != 0 {
// 		err := conn.DB.Create(f.ChatMessages).Error
// 		assert.Empty(t, err)
// 	}
// }

// type ModelConnector struct {
// 	Model interface{}

// 	// 定義されるべきコールバック
// 	addToFixture func(t *testing.T, f *Fixture)
// 	connect      func(t *testing.T, f *Fixture, connectingModel interface{})

// 	// 状態
// 	addedToFixture bool
// 	connectings    []*ModelConnector
// }

// func (mc *ModelConnector) Connect(connectors ...*ModelConnector) *ModelConnector {
// 	mc.connectings = append(mc.connectings, connectors...)
// 	return mc // メソッドチェーンで記述できるようにする
// }

// func (mc *ModelConnector) addToFixtureAndConnect(t *testing.T, fixture *Fixture) {
// 	if mc.addedToFixture {
// 		return
// 	}

// 	if mc.addToFixture == nil {
// 		// addToFixtureは必ずセットされている必要がある
// 		t.Errorf("addToFixture field of %T is not properly initialized", mc.Model)
// 	}
// 	mc.addToFixture(t, fixture)

// 	for _, modelConnector := range mc.connectings {
// 		if mc.connect == nil {
// 			// どのモデルとも接続できない場合はconnectをnilにできる
// 			t.Errorf("%T cannot be connected to %T", modelConnector.Model, mc.Model)
// 		}

// 		mc.connect(t, fixture, modelConnector.Model)

// 		modelConnector.addToFixtureAndConnect(t, fixture)
// 	}

// 	mc.addedToFixture = true
// }

// // Build は引数からFixture構造体のフィールドに値をセットします。
// func Build(t *testing.T, modelConnectors ...*ModelConnector) *Fixture {
// 	fixture := &Fixture{}
// 	for _, modelConnector := range modelConnectors {
// 		modelConnector.addToFixtureAndConnect(t, fixture)
// 	}
// 	return fixture
// }

// func User(setter func(user *model.User)) *ModelConnector {
// 	user := modeltest.User(setter)
// 	return &ModelConnector{
// 		Model: user,
// 		addToFixture: func(t *testing.T, f *Fixture) {
// 			f.Users = append(f.Users, user)
// 		},
// 		connect: func(t *testing.T, f *Fixture, connectingModel interface{}) {
// 			switch connectingModel := connectingModel.(type) {
// 			case *model.UserJoinedChatRoom:
// 				connectingModel.UserID = user.UserID
// 			case *model.UserJoinedIndividualChatRoom:
// 			case *model.ChatRoom:
// 			case *model.ChatMessage:
// 				connectingModel.SenderUserID = user.UserID

// 			default:
// 				t.Errorf("%T cannot be connected to %T", connectingModel, user)
// 			}
// 		},
// 	}
// }

// func UserJoinedChatRoom(setter func(user *model.UserJoinedChatRoom)) *ModelConnector {
// 	userJoinedChatRoom := modeltest.UserJoinedChatRoom(setter)
// 	return &ModelConnector{
// 		Model: userJoinedChatRoom,
// 		addToFixture: func(t *testing.T, f *Fixture) {
// 			f.UserJoinedChatRoom = append(f.UserJoinedChatRoom, userJoinedChatRoom)
// 		},
// 		connect: func(t *testing.T, f *Fixture, connectingModel interface{}) {
// 			switch connectingModel := connectingModel.(type) {
// 			case *model.User:
// 			case *model.ChatRoom:
// 				userJoinedChatRoom.ChatRoomID = connectingModel.ChatRoomID
// 			case *model.ChatMessage:
// 			default:
// 				t.Errorf("%T cannot be connected to %T", connectingModel, userJoinedChatRoom)
// 			}
// 		},
// 	}
// }
// func UserJoinedIndividualChatRoom(setter func(user *model.UserJoinedIndividualChatRoom)) *ModelConnector {
// 	userJoinedIndividualChatRoom := modeltest.UserJoinedIndividualChatRoom(setter)
// 	return &ModelConnector{
// 		Model: userJoinedIndividualChatRoom,
// 		addToFixture: func(t *testing.T, f *Fixture) {
// 			f.UserJoinedIndividualChatRoom = append(f.UserJoinedIndividualChatRoom, userJoinedIndividualChatRoom)
// 		},
// 		connect: func(t *testing.T, f *Fixture, connectingModel interface{}) {
// 			switch connectingModel := connectingModel.(type) {
// 			case *model.ChatRoom:
// 				userJoinedIndividualChatRoom.ChatRoomID = connectingModel.ChatRoomID
// 			case *model.ChatMessage:
// 			default:
// 				t.Errorf("%T cannot be connected to %T", connectingModel, userJoinedIndividualChatRoom)
// 			}
// 		},
// 	}
// }

// func ChatRoom(setter func(user *model.ChatRoom)) *ModelConnector {
// 	chatRoom := modeltest.ChatRoom(setter)
// 	return &ModelConnector{
// 		Model: chatRoom,
// 		addToFixture: func(t *testing.T, f *Fixture) {
// 			f.ChatRooms = append(f.ChatRooms, chatRoom)
// 		},
// 		connect: func(t *testing.T, f *Fixture, connectingModel interface{}) {
// 			switch connectingModel := connectingModel.(type) {
// 			case *model.ChatMessage:
// 			default:
// 				t.Errorf("%T cannot be connected to %T", connectingModel, chatRoom)
// 			}
// 		},
// 	}
// }
// func ChatMessage(setter func(user *model.ChatMessage)) *ModelConnector {
// 	chatMessage := modeltest.ChatMessage(setter)
// 	return &ModelConnector{
// 		Model: chatMessage,
// 		addToFixture: func(t *testing.T, f *Fixture) {
// 			f.ChatMessages = append(f.ChatMessages, chatMessage)
// 		},
// 		connect: func(t *testing.T, f *Fixture, connectingModel interface{}) {
// 			switch connectingModel := connectingModel.(type) {
// 			case *model.ChatMessage:
// 			default:
// 				t.Errorf("%T cannot be connected to %T", connectingModel, chatMessage)
// 			}
// 		},
// 	}
// }
