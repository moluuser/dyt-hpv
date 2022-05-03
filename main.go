package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/jordan-wright/email"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DytApiHost    = "https://dytapi.ynhdkc.com/"
	KeyWord       = "九价"
	XUuid         = ""
	Authorization = ""
	EmailUser     = ""
	EmailPass     = ""
	EmailTo1      = ""
	EmailTo2      = ""

	IsAppointment = false
	IsSending     = true
)

// Response json of hospital list
type hosResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		HosLogo    string `json:"hos_logo"`
		HosName    string `json:"hos_name"`
		HosAddress string `json:"hos_address"`
		HosId      string `json:"hos_id"`
		Status     int    `json:"status"`
		Doctor     []struct {
			DepId int `json:"dep_id"`
			DocId int `json:"doc_id"`
		} `json:"doctor"`
		HosCode  int         `json:"hos_code"`
		HosCode2 interface{} `json:"hos_code2"`
		Sort     int         `json:"sort"`
	} `json:"data"`
}

// Response json of hospital detail list
type hosDetailResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ClassList       []interface{} `json:"class_list"`
		DepId           string        `json:"dep_id"`
		DepName         string        `json:"dep_name"`
		DocAvatar       string        `json:"doc_avatar"`
		DocGood         string        `json:"doc_good"`
		DocId           string        `json:"doc_id"`
		DocInfo         string        `json:"doc_info"`
		DocName         string        `json:"doc_name"`
		HosId           string        `json:"hos_id"`
		HosName         string        `json:"hos_name"`
		HospitalType    int           `json:"hospital_type"`
		HosType         int           `json:"hos_type"`
		IsPage          int           `json:"is_page"`
		LevelName       string        `json:"level_name"`
		ReservationType int           `json:"reservation_type"`
		HospitalRule    string        `json:"hospital_rule"`
		IsDatepart      int           `json:"is_datepart"`
		Favorite        int           `json:"favorite"`
		StopInfo        string        `json:"stop_info"`
		IsInnerSystem   int           `json:"is_inner_system"`
	} `json:"data"`
}

type hpvScheduleResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ScheduleId int    `json:"schedule_id"`
		TimeType   string `json:"time_type"`
		SchDate    string `json:"sch_date"`
		SrcMax     int    `json:"src_max"`
		SrcNum     int    `json:"src_num"`
		CateName   string `json:"cate_name"`
		Ghf        int    `json:"ghf"`
		Zlf        int    `json:"zlf"`
		Zjf        int    `json:"zjf"`
		Amt        int    `json:"amt"`
		DocId      string `json:"doc_id"`
		IsDatepart int    `json:"is_datepart"`
	} `json:"data"`
}

type mailInfo struct {
	SchDate  string
	CateName string
	docName  string
	hosName  string
	SrcMax   int
	SrcNum   int
}

func main() {
	log.Println("↓====================↓")
	// Initialize DB to storage HosDetail
	db, err := sql.Open("sqlite3", "file:hpv.db?mode=memory")
	_, err = db.Exec("CREATE TABLE hos_detail(hos_name VARCHAR(1024), doc_name VARCHAR(1024), doc_good VARCHAR(1024), hos_id VARCHAR(32), doc_id VARCHAR(32), dep_id VARCHAR(32))")
	if err != nil {
		panic(err)
	}

	h, err := getHosList()
	if err != nil {
		log.Fatalln(err.Error())
	}

	var ms []mailInfo

	for _, d := range h.Data {
		for _, doctor := range d.Doctor {
			// Catch all hpv programme
			_, err = getHosDetail(db, strconv.Itoa(doctor.DocId), strconv.Itoa(d.HosCode), strconv.Itoa(doctor.DepId))

			// Catch hpv remaining
			var m mailInfo
			_, m, _, err = getHpvSchedule(db, strconv.Itoa(doctor.DocId), strconv.Itoa(d.HosCode), strconv.Itoa(doctor.DepId))
			ms = append(ms, m)

			if err != nil {
				log.Fatalln(err.Error())
			}
		}
	}

	db.Close()
	log.Println("↑====================↑")
	return
}

func getHosList() (h hosResp, err error) {
	client := resty.New()
	resp, err := client.R().Get(DytApiHost + "Vaccine/hpvhoslist")
	if err != nil {
		return
	}

	respString := resp.String()
	err = json.Unmarshal([]byte(respString), &h)
	if err != nil {
		return
	}

	return
}

func getHosDetail(db *sql.DB, docId string, hosCode string, depId string) (hd hosDetailResp, err error) {
	client := resty.New()
	resp, err := client.R().
		SetHeaders(map[string]string{
			"Host":            "newdytapi.ynhdkc.com",
			"Origin":          "https://appv2.ynhdkc.com",
			"Accept-Encoding": "gzip, deflate, br",
			"Connection":      "keep-alive",
			"Accept":          "application/json, text/plain, */*",
			"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) MicroMessenger/6.8.0(0x16080000) MacWechat/3.4(0x13040010) MiniProgramEnv/Mac MiniProgram",
			"Referer":         "https://appv2.ynhdkc.com/",
			"Accept-Language": "zh-CN,zh-Hans;q=0.9",
			"Content-Type":    "text/plain",
		}).
		Get(DytApiHost + "index/doctor/" + docId + "?hos_code=" + hosCode + "&dep_id=" + depId + "&vip=0")
	// Such as: 2713, 872003, 752

	if err != nil {
		return
	}

	respString := resp.String()
	err = json.Unmarshal([]byte(respString), &hd)

	if err != nil {
		return
	}

	_, err = db.Exec(fmt.Sprintf("INSERT INTO hos_detail VALUES('%v', '%v', '%v', '%v', '%v', '%v')", hd.Data.HosName, hd.Data.DocName, hd.Data.DocGood, hd.Data.HosId, hd.Data.DocId, hd.Data.DepId))
	if err != nil {
		panic(err)
	}

	// fmt.Printf("%v:\t%v\n", hd.Data.HosName, hd.Data.DocName)
	return
}

func getHpvSchedule(db *sql.DB, docId string, hosCode string, depId string) (hs hpvScheduleResp, m mailInfo, str string, err error) {
	client := resty.New()
	resp, err := client.R().
		SetHeaders(map[string]string{
			"Host":            "newdytapi.ynhdkc.com",
			"Origin":          "https://appv2.ynhdkc.com",
			"Accept-Encoding": "gzip, deflate, br",
			"Connection":      "keep-alive",
			"Accept":          "application/json, text/plain, */*",
			"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) MicroMessenger/6.8.0(0x16080000) MacWechat/3.4(0x13040010) MiniProgramEnv/Mac MiniProgram",
			"Referer":         "https://appv2.ynhdkc.com/",
			"Accept-Language": "zh-CN,zh-Hans;q=0.9",
			"Content-Type":    "text/plain",
		}).
		Get(DytApiHost + "index/schedule?hos_code=" + hosCode + "&dep_id=" + depId + "&doc_id=" + docId + "&hyid=&vip=0")

	if err != nil {
		return
	}

	respString := resp.String()
	err = json.Unmarshal([]byte(respString), &hs)

	if err != nil {
		return
	}

	for _, d := range hs.Data {
		var rows *sql.Rows
		rows, err = db.Query(fmt.Sprintf("SELECT * FROM hos_detail WHERE doc_id = %v LIMIT 1", d.DocId))
		var (
			hosName string
			docName string
			docGood string
			hosId   string
			docId   string
			depId   string
		)
		for rows.Next() {
			err := rows.Scan(&docName, &hosName, &docGood, &hosId, &docId, &depId)
			if err != nil {
				panic(err)
			}
		}
		rows.Close()

		str = fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v", d.SchDate, d.CateName, docName, hosName, d.SrcMax, d.SrcNum)
		fmt.Println(str)

		m = mailInfo{d.SchDate, d.CateName, docName, hosName, d.SrcMax, d.SrcNum}

		if d.SrcNum > 0 && strings.Contains(hosName, KeyWord) {
			fmt.Println("====================")
			fmt.Println(str)
			fmt.Println("====================")

			// Send Email
			if IsSending {
				e := email.NewEmail()
				e.From = EmailUser
				e.To = []string{EmailTo1}
				if EmailTo2 != "" {
					e.To = append(e.To, EmailTo2)
				}
				e.Subject = "[重要]滇医通HPV疫苗余量提示"
				e.Text = []byte(fmt.Sprintf("时间：%v\t%v\n地点：%v\n项目：%v\n计划：%v\n剩余：%v", d.SchDate, d.CateName, docName, hosName, d.SrcMax, d.SrcNum))
				err = e.Send("smtp.88.com:25", smtp.PlainAuth("", EmailUser, EmailPass, "smtp.88.com"))
				if err != nil {
					log.Fatalln(err.Error())
				}
			}

			// Appointment
			if IsAppointment {
				appointmentHpv()
			}
		}
	}

	return
}

func appointmentHpv() {
	// Todo
	client := resty.New()
	resp, err := client.R().
		SetHeaders(map[string]string{
			"Host":            "newdytapi.ynhdkc.com",
			"Origin":          "https://appv2.ynhdkc.com",
			"Accept-Encoding": "gzip, deflate, br",
			"Connection":      "keep-alive",
			"Accept":          "application/json, text/plain, */*",
			"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) MicroMessenger/6.8.0(0x16080000) MacWechat/3.4(0x13040010) MiniProgramEnv/Mac MiniProgram",
			"Referer":         "https://appv2.ynhdkc.com/",
			"Accept-Language": "zh-CN,zh-Hans;q=0.9",
			"Content-Type":    "text/plain",
		}).
		Post("")
	if err != nil {
		return
	}
	respString := resp.String()
	fmt.Println(respString)
}
