package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"log"
	"strconv"
)

const (
	DytApiHost = "https://dytapi.ynhdkc.com/"
	KeyWord    = "九价"

	XUUID         = ""
	Authorization = ""
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

func main() {
	h, err := getHosList()
	if err != nil {
		log.Fatalln(err.Error())
		return
	}

	for _, d := range h.Data {
		for _, doctor := range d.Doctor {
			_, err = getHosDetailList(strconv.Itoa(doctor.DocId), strconv.Itoa(d.HosCode), strconv.Itoa(doctor.DepId))
			if err != nil {
				log.Fatalln(err.Error())
				return
			}
		}
	}

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

func getHosDetailList(docId string, hosCode string, depId string) (hd hosDetailResp, err error) {
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
	// 2713, ¬872003, 752

	if err != nil {
		return
	}

	respString := resp.String()
	err = json.Unmarshal([]byte(respString), &hd)

	if err != nil {
		return
	}

	fmt.Printf("%v:\t%v\n", hd.Data.HosName, hd.Data.DocName)
	return
}
