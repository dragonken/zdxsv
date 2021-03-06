package lobby

import (
	"fmt"
	"strconv"
	"zdxsv/pkg/db"
	. "zdxsv/pkg/lobby/message"

	"github.com/golang/glog"
)

// 6007 : ServerFull サーバ選択画面へ
// 6003 : 強制ログイン画面へ メンテに使えるかも
//

var _ = register(0x6006, "Logout", func(p *AppPeer, m *Message) {
	p.app.OnUserLogout(p)
	p.SendMessage(NewServerAnswer(m))
})

var _ = register(0x6002, "GotoBattle", func(p *AppPeer, m *Message) {
	p.app.OnUserGotoBattle(p)
	p.SendMessage(NewServerAnswer(m))
})

func SendServerShutDown(p *AppPeer) {
	n := NewServerNotice(0x6003)
	w := n.Writer()
	w.WriteString("<BODY><LF=6><CENTER>サーバがシャットダウンしました<END>")
	p.SendMessage(n)
}

func RequestLineCheck(p *AppPeer) {
	m := NewServerQuestion(0x6001)
	p.SendMessage(m)
}

var _ = register(0x6001, "OnLineCheck", func(_ *AppPeer, _ *Message) {
})

var _ = register(0x600E, "EchoPacket", func(p *AppPeer, m *Message) {
	a := NewServerAnswer(m)
	w := a.Writer()
	w.Write(m.Body)
	p.SendMessage(a)
})

var _ = register(0x61A0, "NAZO 0x61A0", func(p *AppPeer, m *Message) {
	a := NewServerAnswer(m)
	w := a.Writer()
	// 自動切断猶予
	w.Write([]byte{0x00, 0x00, 0x0E, 0x10, 0x00, 0x00, 0x02, 0x58})
	p.SendMessage(a)
})

// Login Sequence
func RequestKeyPair(p *AppPeer) {
	m := NewServerQuestion(0x6101)
	m.Seq = 1
	w := m.Writer()
	w.Write16(0x2837)
	p.SendMessage(m)
}

var _ = register(0x6101, "ResponseKeyPair", func(p *AppPeer, m *Message) {
	r := m.Reader()
	key1 := r.ReadString()
	loginKey := r.ReadEncryptedString()
	glog.Infoln(key1)
	var1, err := strconv.ParseUint(key1, 10, 64)
	if err != nil {
		glog.Errorln(err)
	}
	var2 := fmt.Sprintf("%010d", var1-100001)
	sessionID := var2[1:5] + var2[6:]
	p.app.OnKeePair(p, loginKey, sessionID)
})

func RequestFirstData(p *AppPeer) {
	m := NewServerQuestion(0x6103)
	p.SendMessage(m)
}

var _ = register(0x6103, "FirstData", func(p *AppPeer, _ *Message) {
	// r := m.Reader()
	// TODO:レスポンスの詳細を調査する
	if p.LoginKey == "" {
		glog.Errorln("loginKey not set")
		return
	}
	p.app.OnFirstData(p)
})

func NoticeUserIDList(p *AppPeer, users []*db.User) {
	m := NewServerNotice(0x6131)
	w := m.Writer()
	w.Write8(byte(len(users)))
	for _, u := range users {
		w.WriteString(u.UserID)
		w.WriteString(u.Name)
		w.WriteString(u.Team)
	}
	p.SendMessage(m)
}

var _ = register(0x6132, "DecideUserID", func(p *AppPeer, m *Message) {
	r := m.Reader()
	userID := r.ReadEncryptedString()
	name := r.ReadEncryptedString()
	p.app.OnDecideUserID(p, userID, name)
	a := NewServerAnswer(m)
	w := a.Writer()
	w.WriteString(p.UserID) // これだけか？
	p.SendMessage(a)
})

var _ = register(0x6190, "DecideTeam", func(p *AppPeer, m *Message) {
	team := m.Reader().ReadEncryptedString()
	p.app.OnDecideTeam(p, team)
	p.SendMessage(NewServerAnswer(m))
})

var _ = register(0x614C, "GetTopInformation", func(p *AppPeer, m *Message) {
	a := NewServerAnswer(m)
	w := a.Writer()
	w.Write8(0)
	/*
		FIXME:
		It's works but buggy...
		w.Write8(1)
		w.WriteString("<BODY><LF=5><CENTER>ーパッド遅延軽減ハックを試験導入中ー<BR><BR>ＵＤＰプロキシ使用推奨です。<BR>不具合やラグの改善具合の報告にご協力ください。<END>")
	*/
	p.SendMessage(a)
})

func NoticeLoginOk(p *AppPeer) {
	p.SendMessage(NewServerNotice(0x6104))
}

var _ = register(0x6144, "GetParsonalRecordHeader", func(p *AppPeer, m *Message) {
	a := NewServerAnswer(m)
	w := a.Writer()
	// The response data is all 1 byte size,
	// the first one seems to represent the number that follows.
	w.Write8(0)
	p.SendMessage(a)
})

var _ = register(0x6145, "GetParsonalRecordData", func(p *AppPeer, m *Message) {
	a := NewServerAnswer(m)
	r := m.Reader()
	kind := r.Read8()
	page := r.Read8()

	rec := p.app.OnGetUserRanking(p, kind, page)

	w := a.Writer()
	w.Write8(kind)
	w.Write8(page)
	w.Write32(rec.EntireCount)
	w.Write32(0xAAAA) // Unknown
	w.Write32(rec.Invalid)
	w.Write8(rec.Class)
	w.Write32(rec.Rank)
	w.Write32(0xBBBB) // Unknown
	w.Write32(rec.Battle)
	w.Write32(rec.Win)
	w.Write32(rec.Lose)
	w.Write32(rec.Kill)
	w.Write32(0xCCCC) // Unknown
	w.Write32(0xDDDD) // Unknown
	w.Write32(0xEEEE) // Unknown
	w.Write32(0xFFFF) // Unknown

	p.SendMessage(a)
})

// *UNUSED*
var _ = register(0x6146, "GetParsonalRecordVide", func(p *AppPeer, m *Message) {
	a := NewServerAnswer(m)
	r := m.Reader()
	kind := r.Read8()
	page := r.Read8()

	w := a.Writer()
	w.Write8(kind)
	w.Write8(page)

	w.WriteString(fmt.Sprintf("Vide %v %v", kind, page))
	w.Write8(1)
	w.Write8(2)
	w.Write8(3)
	w.Write8(4)
	w.Write8(5)
	w.Write8(6)
})

var _ = register(0x6143, "SetUserBinary", func(p *AppPeer, m *Message) {
	r := m.Reader()
	bin := r.ReadEncryptedString()
	p.app.OnSetUserBinary(p, bin)
	p.SendMessage(NewServerAnswer(m))
})

// 0x6143のレスポンスを送ったらクライアントが送ってくる.
// 0バイトだから特に情報はないが, 目的は不明.
var _ = register(0x6141, "NoticeNazo", func(p *AppPeer, m *Message) {
})

// Custom API
var _ = register(0x0001, "RegisterOldProxy", func(p *AppPeer, m *Message) {
	a := NewServerAnswer(m)
	a.Category = CategoryCustom
	a.Writer().WriteString("OLD VERSION")
	p.SendMessage(a)
})
