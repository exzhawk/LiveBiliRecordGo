package main

//todo auto retry http get

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
	//"path"
	"io"
	"path"
)

const (
	BaseRoomUrl = "http://live.bilibili.com/%d"
	//BaseCidUrl      = "http://live.bilibili.com/api/player?id=cid:%d"
	BaseCidUrl      = "https://api.live.bilibili.com/room/v1/Room/get_info?device=phone&platform=ios&scale=3&build=10000&room_id=%d"
	BasePlayUrl     = "https://api.live.bilibili.com/room/v1/Room/playUrl?cid=%d&qn=0&platform=web"
	VideoUrlChoice  = "url"
	RoomIdReString  = "ROOMID: (\\d+)"
	BaseRoomIdUrl   = "http://api.live.bilibili.com/room/v1/Room/room_init?id=%d"
	ProtocolVersion = 1
	BaseFilename    = "{}_{}.flv"
	WssUrl          = "wss://broadcastlv.chat.bilibili.com/sub"
	BaseWsUrl       = "%s://%s:%d/sub"
	BaseConfUrl     = "https://api.live.bilibili.com/room/v1/Danmu/getConf?room_id=%d&platform=h5"
	BaseJoinJson    = "{\"uid\":0,\"roomid\":%d,\"protover\":2,\"platform\":\"web\",\"clientver\":\"1.10.6\",\"type\":2}"
)

var roomIdRe, _ = regexp.Compile(RoomIdReString)

type BiliConn struct {
	*websocket.Conn
}
type Header struct {
	Length          uint32
	HeaderLength    uint16
	ProtocolVersion uint16
	Operation       uint32
	SequenceId      uint32
}
type BiliMsgJson struct {
	Cmd string
}
type BiliPlayUrlJson struct {
	Data struct {
		Durl []struct {
			Url string
		}
	}
}
type BiliXml struct {
	Url string `xml:"durl>url"`
}

type BiliRoomJson struct {
	Data struct {
		Room_id     int
		Live_status int
	}
}

type BiliConfJson struct {
	Data struct {
		Host_server_list []struct {
			Host     string
			port     int
			Ws_port  int
			Wss_port int
		}
	}
}

//go download(){
// download<-chan
// download!
// }

//get real room id
//get cid url to get status
//if living, chan<-download

//connect websocket
//go receiving(){if live, chan<-download}
//go heartbeat()
var urlId int = 281

//var urlId int = 220416

//var urlId int = 220416
var downloadDir string = "r"
var downloadChan chan time.Time
var roomId int = 49728

func main() {
	//var urlId int = 220416

	initLog()

	downloadChan = make(chan time.Time, 10)
	roomId = getRoomId(urlId)
	playerInfo := getPlayerInfo(roomId)
	go download(downloadChan, roomId)

	for {
		runWs(playerInfo)
	}

}
func runWs(playerInfo map[string]string) {
	conn := connectWs(playerInfo)
	defer conn.Close()
	conn.joinRoom(roomId)
	go conn.startHeartbeat()
	conn.receiving(downloadChan)
}

func initLog() {
	logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func (biliConn *BiliConn) receiving(downloadChan chan<- time.Time) {
	for {
		_, message, err := biliConn.ReadMessage()
		if err != nil {
			log.Println("read error: ", err)
			biliConn.Close()
			return
		}
		parseMessage(message, downloadChan)
	}
}
func parseMessage(message []byte, downloadChan chan<- time.Time) {
	if len(message) <= 16 {
		return
	}
	var header Header
	headerBuffer := bytes.NewBuffer(message[:16])
	err := binary.Read(headerBuffer, binary.BigEndian, &header)
	if err != nil {
		log.Println("binary read error: ", err)
	}
	length := header.Length
	bodyBytes := message[16:length]
	switch protocol := int(header.ProtocolVersion); protocol {
	case 0:
		switch operation := int(header.Operation); operation {
		case 8:
			log.Println("joined room")
		case 3:
			//log.Println("online number")
		case 5:
			var j BiliMsgJson
			err := json.Unmarshal(bodyBytes, &j)
			if err != nil {
				log.Println(string(bodyBytes))
				log.Println("json unmarshal error", err)
				break
			}
			switch cmd := j.Cmd; cmd {
			case "LIVE":
				downloadChan <- time.Now()
			case "PREPARING":
				log.Println("living stopped")
			case "DANMU_MSG":
				//log.Println("receive danmu")
			case "SEND_GIFT", "WELCOME", "SYS_MSG", "SYS_GIFT", "WELCOME_GUARD", "SPECIAL_GIFT", "BUY_GUARD",
				"ROOM_BLOCK_MSG", "RAFFLE_START", "RAFFLE_END", "EVENT_CMD", "TV_START", "GUARD_BUY", "ACTIVITY_EVENT",
				"ROOM_SILENT_OFF", "ROOM_RANK", "COMBO_END", "COMBO_SEND", "ROOM_SILENT_ON", "GUARD_MSG", "ENTRY_EFFECT",
				"NOTICE_MSG", "ROOM_REAL_TIME_MESSAGE_UPDATE", "USER_TOAST_MSG", "GUARD_LOTTERY_START", "TV_END",
				"ACTIVITY_BANNER_RED_NOTICE_CLOSE", "WEEK_STAR_CLOCK", "DAILY_QUEST_NEWDAY", "GUIARD_MSG", "ROOM_CHANGE",
				"ROOM_BOX_MASTER", "HOUR_RANK_AWARDS", "ROOM_SKIN_MSG", "CHANGE_ROOM_INFO", "ACTIVITY_BANNER_UPDATE_V2",
				"DANMU_MSG:4:0:2:2:2:0", "VOICE_JOIN_ROOM_COUNT_INFO", "VOICE_JOIN_LIST", "PK_BATTLE_ENTRANCE",
				"ROOM_BANNER", "PANEL", "INTERACT_WORD", "ONLINERANK", "ONLINE_RANK_V2", "ONLINE_RANK_TOP3",
				"ONLINE_RANK_COUNT", "WIDGET_BANNER", "HOT_RANK_CHANGED", "STOP_LIVE_ROOM_LIST", "HOT_RANK_SETTLEMENT",
				"COMMON_NOTICE_DANMAKU", "LIVE_INTERACTIVE_GAME", "HOT_RANK_CHANGED_V2", "LIVE_PANEL_CHANGE",
				"HOT_RANK_SETTLEMENT_V2", "VOICE_JOIN_STATUS", "THERMAL_STORM_DANMU_BEGIN",
				"THERMAL_STORM_DANMU_UPDATE", "THERMAL_STORM_DANMU_OVER", "THERMAL_STORM_DANMU_CANCEL", "TRADING_SCORE",
				"WATCHED_CHANGE", "POPULARITY_RED_POCKET_WINNER_LIST", "POPULARITY_RED_POCKET_NEW",
				"POPULARITY_RED_POCKET_START", "GUARD_HONOR_THOUSAND", "GUARD_ACHIEVEMENT_ROOM":
				//log.Println("receive message: ", j.Cmd)
			default:
				log.Println("receive unknown: ", j.Cmd)

			}
		}
	case 1:
		return
	case 2:
		buffer := bytes.NewBuffer(bodyBytes[2:]) //remove two useless byte
		flateReader := flate.NewReader(buffer)
		defer flateReader.Close()
		decompressedBytes, err := ioutil.ReadAll(flateReader)
		if err != nil {
			log.Println("decompress failed", err)
			break
		}
		parseMessage(decompressedBytes, downloadChan)
	}
	//there is another message here
	if len(message) > int(length) {
		parseMessage(message[length:], downloadChan)
	}
}
func (biliConn *BiliConn) startHeartbeat() {
	log.Print("starting heartbeat")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		//log.Print("sending heartbeat")
		err := biliConn.sendData(2, "[object Object]")
		if err != nil {
			log.Println("sending error", err)
			return
		}
	}
}
func (biliConn *BiliConn) joinRoom(roomId int) {
	log.Print("joining room: ", roomId)
	joinBody := fmt.Sprintf(BaseJoinJson, roomId)
	err := biliConn.sendData(7, joinBody)
	if err != nil {
		log.Fatal("join fail: ", err)
	}
}
func (biliConn *BiliConn) sendData(action uint32, bodyString string) (err error) {
	length := 16 + len([]byte(bodyString))

	header := Header{uint32(length), 16, 1, action, 1}
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, header)
	messageBytes := append(buffer.Bytes(), []byte(bodyString)...)
	err = biliConn.WriteMessage(websocket.BinaryMessage, messageBytes)
	return
}

func download(downloadChan <-chan time.Time, roomId int) {
	log.Print("init download for room: ", roomId)
	err := os.Mkdir(downloadDir, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) {
			log.Fatal(err)
		}
	}
	for downloadTime := range downloadChan {
		log.Print("get download instruction ", downloadTime)
		if time.Now().Sub(downloadTime) > 20*time.Second {
			continue
		}
		url := fmt.Sprintf(BasePlayUrl, roomId)
		body := httpGet(url)
		var j BiliPlayUrlJson
		err := json.Unmarshal(body, &j)
		if err != nil {
			log.Println("json unmarshal error: ", err)
		}
		videoUrl := j.Data.Durl[0].Url
		log.Println("download video from: ", videoUrl)
		downloadBigFile(videoUrl, downloadTime)

		go getPlayerInfo(roomId)
	}

}
func downloadBigFile(downloadUrl string, downloadTime time.Time) {
	filename := downloadTime.Format("2006-01-02 15-04-04") + ".flv"
	out, err := os.Create(path.Join(downloadDir, filename))
	if err != nil {
		log.Fatal("create file error: ", err)
	}
	defer out.Close()

	resp, err := http.Get(downloadUrl)
	if err != nil {
		log.Println("download file error: ", err)
		return
	}
	defer resp.Body.Close()

	var written uint64 = 0
	var lastWritten uint64 = 0
	buffer := make([]byte, 1*1024*1024)
	ticker := time.NewTicker(20 * time.Second)

IoCopy:
	for {
		select {
		case <-ticker.C:
			log.Printf("downloading %s", humanize.IBytes(written))
			if written == lastWritten {
				break IoCopy
			}
			lastWritten = written
		default:
		}
		nr, er := resp.Body.Read(buffer)
		if nr > 0 {
			nw, ew := out.Write(buffer[0:nr])
			if nw > 0 {
				written += uint64(nw)
			}
			if ew != nil {
				log.Println("write error: ", ew)
				break
			}
			if nr != nw {
				log.Println("write short error")
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				log.Println("read error: ", er)
				break
			}
		}
	}
	log.Printf("downloaded %s", humanize.Bytes(written))

}

func httpGet(url string) (body []byte) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("http get: ", err)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Fatal("http read: ", err)
	}
	body = bodyBytes
	return
}

func getRoomId(UrlId int) (roomId int) {
	log.Print("getting real room id")
	url := fmt.Sprintf(BaseRoomIdUrl, UrlId)
	body := httpGet(url)
	var j BiliRoomJson
	err := json.Unmarshal(body, &j)
	if err != nil {
		log.Println("json unmarshal error: ")
	}
	roomId = j.Data.Room_id
	log.Print("got real room id: ", roomId)
	return
}

func getPlayerInfo(roomId int) (playerInfo map[string]string) {
	log.Print("getting player info")
	url := fmt.Sprintf(BaseRoomIdUrl, roomId)
	body := httpGet(url)
	var j BiliRoomJson
	err := json.Unmarshal(body, &j)
	if err != nil {
		log.Println("json unmarshal error: ")
	}
	liveStatus := j.Data.Live_status
	log.Println("got player info")
	if liveStatus == 1 {
		log.Println("is living")
		downloadChan <- time.Now()
	} else {
		log.Println("not living")
	}
	playerInfo = make(map[string]string)
	return

	//url := fmt.Sprintf(BaseCidUrl, roomId)
	//body := httpGet(url)
	//xmlReString := "<(\\w*)>(.*?)</\\w+>"
	//xmlRe, _ := regexp.Compile(xmlReString)
	//matchStrings := xmlRe.FindAllSubmatch(body, -1)
	//playerInfo = make(map[string]string)
	//for _, result := range matchStrings {
	//	playerInfo[string(result[1])] = string(result[2])
	//}
	//log.Print("got player info", playerInfo)
	//log.Print("current living status: ", playerInfo["state"])

	//if playerInfo["state"] == "LIVE" {
	//	log.Println("is living")
	//	downloadChan <- time.Now()
	//}
	//return
}

func connectWs(playerInfo map[string]string) (biliConn *BiliConn) {
	log.Print("getting conf info")
	url := fmt.Sprintf(BaseConfUrl, roomId)
	body := httpGet(url)
	var j BiliConfJson
	err := json.Unmarshal(body, &j)
	if err != nil {
		log.Println("json unmarshal error: ")
	}
	log.Println("connecting websocket")
	//wsUrl := fmt.Sprintf(BaseWsUrl, "wss", j.Data.Host_server_list[0].Host, j.Data.Host_server_list[0].Wss_port)
	wsUrl := WssUrl
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	biliConn = &BiliConn{conn}
	return
}
